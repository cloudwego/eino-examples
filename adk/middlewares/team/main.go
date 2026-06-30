/*
 * Copyright 2026 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

// Package main demonstrates the Agent Teams middleware for multi-agent collaboration.
//
// This demo creates a Leader agent that spawns three Teammate agents (security-reviewer,
// perf-reviewer, test-reviewer) to review a PR. The flow closely follows the real
// Claude Code Agent Teams protocol, using an in-memory backend for storage.
//
// Architecture:
//
//	Leader (team-lead)
//	  ├── TaskCreate tool  → creates review tasks
//	  ├── Agent tool       → spawns teammates (run_in_background)
//	  └── SendMessage tool → shutdown_request to teammates
//
//	Teammate (security-reviewer / perf-reviewer / test-reviewer)
//	  ├── TaskUpdate tool  → claim and complete tasks
//	  └── SendMessage tool → send report to leader / shutdown_approved
//
// Flow:
//  1. The team is created automatically when the Runner is constructed.
//  2. Leader creates 3 review tasks
//  3. Leader spawns 3 teammates via Agent tool (run_in_background=true)
//  4. Each teammate: TaskUpdate(in_progress) → TaskUpdate(completed) → SendMessage(report)
//  5. Leader receives reports, sends shutdown_request to each teammate
//  6. Each teammate responds with shutdown_approved
//  7. Leader receives teammate_terminated notifications; the team is cleaned up
//     automatically when the Runner exits.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/team"
	"github.com/cloudwego/eino/schema"
)

type agentHistory struct {
	mu       sync.Mutex
	messages map[string][]*schema.Message
}

func main() {
	ctx := context.Background()

	traceCloseFn, startSpanFn := AppendCozeLoopCallbackIfConfigured(ctx)
	defer traceCloseFn(ctx)

	baseDir := "./base_dir"
	os.RemoveAll(baseDir)
	if _, err := os.Stat(baseDir); err != nil {
		os.MkdirAll(baseDir, 0755)
	}

	backend := newFileBackend(baseDir) // 或留空使用系统临时目录

	leaderQuery := `I have three questions:
Question 1: Why is the sky blue?
Question 2: Why is seawater salty?
Question 3: What is the meaning of human existence?
Please create 3 expert teammates, each responsible for answering one of these questions. Keep the answers brief—just one sentence each.
The team leader should not answer the questions directly, but should wait until all three experts have responded, and then provide me with a summary of their answers.`

	ctx, endSpanFn := startSpanFn(ctx, "team", leaderQuery)

	modelImpl, err := NewChatModel(ctx)
	if err != nil {
		log.Fatalf("create chat model: %v", err)
	}

	history := newAgentHistory()

	runner, err := team.NewRunner(ctx, &team.RunnerConfig{
		AgentConfig: &adk.ChatModelAgentConfig{
			Name:          "team-lead",
			Description:   "Leader that coordinates a research team",
			Instruction:   "You are a helpful assistant.",
			Model:         modelImpl,
			MaxIterations: 1000,
			Handlers: []adk.ChatModelAgentMiddleware{
				NewToolWrapMiddleware(),
			},
		},
		TeamConfig: &team.Config{
			Backend:          backend,
			BaseDir:          baseDir,
			Interval:         10,
			RetainDataOnExit: false,
		},
		TeammateRoles: []team.TeammateRole{
			{
				Name:        "geo-expert",
				Description: "A experienced geography expert.",
				Instruction: "You are a experienced geography expert.",
			},
			{
				Name:        "philosopher",
				Description: "A philosopher.",
				Instruction: "You are a experienced philosopher.",
			},
		},
		GenInput: func(_ context.Context, _ *adk.TurnLoop[team.TurnInput, adk.Message], items []team.TurnInput) (*adk.GenInputResult[team.TurnInput, adk.Message], error) {
			target := ""
			if len(items) > 0 {
				target = items[0].TargetAgent
			}

			for _, item := range items {
				if item.TargetAgent != target {
					panic("GenInput: items must have the same target agent")
				}
				log.Printf("Received GenInput [%s]: %v", target, item)
				msgs := inboxToMessages(item.Messages)
				history.append(target, msgs...)
			}
			allMessages := history.snapshot(target)

			log.Printf("GenInput [%s]: messages.len : %d ， items.len : %d", target, len(allMessages), len(items))

			return &adk.GenInputResult[team.TurnInput, adk.Message]{
				Input: &adk.AgentInput{
					Messages:        allMessages,
					EnableStreaming: false,
				},
				Consumed: items,
			}, nil
		},
		OnAgentEvents: func(_ context.Context, tc *adk.TurnContext[team.TurnInput, adk.Message], events *adk.AsyncIterator[*adk.AgentEvent]) error {
			target := ""
			if len(tc.Consumed) == 0 {
				panic("OnAgentEvents: consumed must not be empty")
			}

			target = tc.Consumed[0].TargetAgent

			for {
				msg, ok := events.Next()
				if !ok {
					break
				}
				if msg.Err != nil {
					return msg.Err
				}

				if msg.Output == nil || msg.Output.MessageOutput == nil {
					continue
				}

				mo := msg.Output.MessageOutput
				if mo.Message != nil {
					history.append(target, mo.Message)
				}
				printEvent(msg)
			}

			return nil
		},
	})
	if err != nil {
		log.Fatalf("create leader runner: %v", err)
	}

	runner.Push(team.TurnInput{
		TargetAgent: team.LeaderAgentName,
		Messages:    []string{leaderQuery},
	})

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("shutting down...")
		runner.Stop()
	}()

	fmt.Println("=== Agent Teams Demo ===")
	fmt.Println()

	runner.Run(ctx)
	exitState := runner.Wait()
	if exitState != nil && exitState.ExitReason != nil {
		log.Printf("leader runner exit reason: %v", exitState.ExitReason)
		endSpanFn(ctx, fmt.Sprintf("team exit with reason: %v", exitState.ExitReason))
	} else {
		endSpanFn(ctx, "team exit with unexpected state")
	}

	// wait for all span to be ended
	time.Sleep(5 * time.Second)

	fmt.Println()
	fmt.Println("=== Demo Complete ===")

}

// ---------------------------------------------------------------------------
// Helper: convert inbox messages to schema messages
// ---------------------------------------------------------------------------

func inboxToMessages(msgs []string) []*schema.Message {
	if len(msgs) == 0 {
		return nil
	}
	var sb strings.Builder
	for _, msg := range msgs {
		sb.WriteString(msg)
		sb.WriteString("\n")
	}
	return []*schema.Message{schema.UserMessage(sb.String())}
}
