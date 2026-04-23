/*
 * Copyright 2025 CloudWeGo Authors
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

package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// RalphLoopConfig configures a Ralph Loop — an autonomous agent loop where the
// same prompt is fed to an agent repeatedly. Each turn, the agent gets a fresh
// context window but discovers prior work via persistent tools (e.g. filesystem).
//
// The loop exits when:
//  1. The agent outputs the CompletionPromise AND VerifyCompletion accepts, or
//  2. MaxTurns is reached.
type RalphLoopConfig struct {
	// Agent is the agent to run each turn. The same agent instance is reused
	// across turns; the TurnLoop feeds fresh input each turn so the agent
	// sees a clean context window.
	Agent adk.Agent

	// Prompt is the task description re-injected every turn.
	Prompt string

	// MaxTurns is the maximum number of turns before forced stop. Required.
	MaxTurns int

	// CompletionPromise is the string the agent must output to signal completion.
	// Defaults to "<COMPLETE/>" if empty.
	CompletionPromise string

	// VerifyCompletion is called when the agent outputs the CompletionPromise.
	// Return nil to accept completion, or a non-nil error to reject and continue.
	// The error message is logged for observability.
	// Optional. If nil, the completion promise is always accepted.
	VerifyCompletion func(ctx context.Context) error

	// OnEvent is called for each agent event during a turn, for observability.
	// Optional.
	OnEvent func(event *adk.AgentEvent)

	// IdleTimeout is the safety-net duration: if the loop becomes idle for this
	// long (no items in buffer, no agent running), it stops automatically.
	// Defaults to 30s if zero.
	IdleTimeout time.Duration
}

// RalphLoopResult contains the outcome of a Ralph Loop execution.
type RalphLoopResult struct {
	// Complete is true if the agent output the CompletionPromise and
	// VerifyCompletion accepted.
	Complete bool

	// Turns is the number of turns executed.
	Turns int

	// MaxTurns is the configured maximum.
	MaxTurns int

	// StopCause is the reason the loop stopped.
	StopCause string

	// Err is non-nil if the loop exited due to an error.
	Err error
}

// RalphLoop is an autonomous agent loop that re-injects the same prompt each
// turn. Create with NewRalphLoop, then call Run to execute.
type RalphLoop struct {
	config RalphLoopConfig
}

// NewRalphLoop creates a new Ralph Loop with the given configuration.
func NewRalphLoop(config RalphLoopConfig) *RalphLoop {
	if config.CompletionPromise == "" {
		config.CompletionPromise = "<COMPLETE/>"
	}
	if config.IdleTimeout == 0 {
		config.IdleTimeout = 30 * time.Second
	}
	return &RalphLoop{config: config}
}

// Run executes the Ralph Loop synchronously, returning when the loop completes.
func (rl *RalphLoop) Run(ctx context.Context) *RalphLoopResult {
	cfg := rl.config

	type task struct {
		turn     int
		complete bool
	}
	t := &task{}

	loop := adk.NewTurnLoop(adk.TurnLoopConfig[*task, *schema.Message]{
		GenInput: func(ctx context.Context, lp *adk.TurnLoop[*task, *schema.Message], items []*task) (*adk.GenInputResult[*task, *schema.Message], error) {
			t := items[0]

			if t.complete {
				lp.Stop(adk.WithStopCause("completion promise accepted"))
				return &adk.GenInputResult[*task, *schema.Message]{
					Input:     &adk.AgentInput{Messages: []adk.Message{schema.UserMessage("done")}},
					Remaining: items,
				}, nil
			}

			if t.turn >= cfg.MaxTurns {
				lp.Stop(adk.WithStopCause("max turns reached"))
				return &adk.GenInputResult[*task, *schema.Message]{
					Input:     &adk.AgentInput{Messages: []adk.Message{schema.UserMessage("done")}},
					Remaining: items,
				}, nil
			}

			t.turn++
			fmt.Printf("\n=== Turn %d/%d ===\n", t.turn, cfg.MaxTurns)

			return &adk.GenInputResult[*task, *schema.Message]{
				Input:    &adk.AgentInput{Messages: []adk.Message{schema.UserMessage(cfg.Prompt)}},
				Consumed: items,
			}, nil
		},

		PrepareAgent: func(ctx context.Context, lp *adk.TurnLoop[*task, *schema.Message], consumed []*task) (adk.Agent, error) {
			return cfg.Agent, nil
		},

		OnAgentEvents: func(ctx context.Context, tc *adk.TurnContext[*task, *schema.Message], events *adk.AsyncIterator[*adk.AgentEvent]) error {
			t := tc.Consumed[0]
			var response strings.Builder

			for {
				event, ok := events.Next()
				if !ok {
					break
				}
				if event.Err != nil {
					return event.Err
				}

				if cfg.OnEvent != nil {
					cfg.OnEvent(event)
				}

				if event.Output != nil && event.Output.MessageOutput != nil {
					if m := event.Output.MessageOutput.Message; m != nil && m.Content != "" {
						response.WriteString(m.Content)
					}
				}
			}

			output := response.String()

			if strings.Contains(output, cfg.CompletionPromise) {
				if cfg.VerifyCompletion != nil {
					if err := cfg.VerifyCompletion(ctx); err != nil {
						fmt.Printf("\n✗ Completion rejected: %v. Re-pushing for another turn.\n", err)
						tc.Loop.Push(t)
						return nil
					}
				}

				fmt.Printf("\n✓ Completion accepted!\n")
				t.complete = true
				tc.Loop.Stop(adk.WithStopCause("completion promise accepted"))
				return nil
			}

			fmt.Printf("\n⟳ No completion promise. Re-pushing for next turn.\n")
			tc.Loop.Push(t)
			return nil
		},
	})

	loop.Push(t)
	loop.Run(ctx)
	loop.Stop(adk.UntilIdleFor(cfg.IdleTimeout))
	exitState := loop.Wait()

	return &RalphLoopResult{
		Complete:  t.complete,
		Turns:     t.turn,
		MaxTurns:  cfg.MaxTurns,
		StopCause: exitState.StopCause,
		Err:       exitState.ExitReason,
	}
}
