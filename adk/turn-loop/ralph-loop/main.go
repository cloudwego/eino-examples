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

// Package main demonstrates the "Ralph Loop" pattern using the TurnLoop API.
//
// The Ralph Loop (originated by Geoffrey Huntley) is an autonomous agent loop
// where the SAME prompt is fed to an AI agent repeatedly. Each turn, the agent
// gets a fresh context window but discovers prior work via the filesystem (files
// persist across turns via InMemoryBackend).
//
// The loop exits only when:
//  1. The agent outputs a completion promise (<COMPLETE/>) AND a verification
//     gate confirms all BUG: markers in the project have been resolved.
//  2. Or max turns are reached.
//
// This example pre-seeds a buggy URL shortener project with ~15 intentional
// bugs marked with BUG: comments. The agent must iteratively find, fix, and
// remove these markers across multiple turns.
//
// Key TurnLoop features demonstrated:
//   - TurnLoop[*Task] to drive the push-based event loop
//   - InMemoryBackend + filesystem middleware for persistent file tools
//   - OnAgentEvents as the "Stop Hook" with a verification gate
//   - ToolCallMiddleware to make tool errors non-fatal (returns error as string)
//   - ModelRetryConfig for resilient API calls with exponential backoff
package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/filesystem"
	fsmiddleware "github.com/cloudwego/eino/adk/middlewares/filesystem"
	cmodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	commonmodel "github.com/cloudwego/eino-examples/adk/common/model"
	"github.com/cloudwego/eino-examples/adk/common/prints"
)

// Task is the item type for TurnLoop. A single Task circulates through the
// loop — consumed each turn by GenInput, then re-pushed by OnAgentEvents if
// the agent hasn't output the completion promise.
type Task struct {
	Prompt   string // The task description (same every turn)
	Turn     int    // Current turn count
	MaxTurns int    // Max turns before forced stop
	Complete bool   // True when <COMPLETE/> is detected
}

const completionPromise = "<COMPLETE/>"

const taskPrompt = `You are an autonomous AI developer working in a loop. Each iteration you receive this same prompt.
You have filesystem tools (ls, read_file, write_file, edit_file, grep, glob) to read and write files.

YOUR TASK:
The /project/ directory contains a Go HTTP URL shortener service with MULTIPLE bugs and missing features.
Your job is to review, fix, and complete the code until it is production-ready.

DELIVERABLES (all must be present and correct):
1. /project/store.go       — URLStore with Shorten, Resolve, Stats methods
2. /project/handler.go     — HTTP handlers: POST /shorten, GET /:code (redirect), GET /stats/:code
3. /project/handler_test.go — At least 8 table-driven tests using httptest
4. /project/main.go        — Wire everything, listen on :8080
5. /project/README.md      — API docs with curl examples

KNOWN ISSUES TO FIX (some may already be fixed from prior iterations):
- store.go: Shorten() does not validate URLs (must require scheme + host)
- store.go: Shorten() does not check for duplicate URLs
- store.go: Stats() is not implemented (returns 0 always)
- handler.go: POST /shorten returns 200 instead of 201
- handler.go: GET /:code does not increment the hit counter
- handler.go: error responses are plain text, should be JSON {"error":"..."}
- handler_test.go: only has 3 test cases, needs at least 8
- handler_test.go: does not test error responses (400, 404, 409)
- main.go: missing graceful shutdown
- README.md: missing curl examples for error cases

CONSTRAINTS:
- Use ONLY the Go standard library.
- Do NOT rewrite files from scratch. Use edit_file for targeted fixes.
- Fix at most 2-3 issues per iteration. Do NOT try to fix everything at once.
- After fixing a bug, REMOVE the corresponding BUG: comment that described it.
- After each edit, re-read the file to verify your change is correct.

WORKFLOW:
1. Use ls and read_file to check what exists and find remaining BUG: comments.
2. Pick 2-3 BUG: comments to fix. Apply targeted edits with edit_file.
3. IMPORTANT: Remove the BUG: comment line(s) for each bug you fix.
4. Re-read the edited files to verify your changes are correct.
5. If ALL BUG: comments have been removed AND all 5 deliverables are complete, output EXACTLY: <COMPLETE/>
   Do NOT output <COMPLETE/> until you have grep'd for "BUG:" and confirmed zero results.
`

func main() {
	ctx := context.Background()

	chatModel := commonmodel.NewChatModel()
	backend := filesystem.NewInMemoryBackend()

	// Seed the filesystem with a buggy starter project.
	// The agent's job is to find and fix all the issues iteratively.
	seedBuggyProject(ctx, backend)

	task := &Task{
		Prompt:   taskPrompt,
		MaxTurns: 10,
	}

	loop := adk.NewTurnLoop(adk.TurnLoopConfig[*Task]{
		GenInput:      makeGenInput(),
		PrepareAgent:  makePrepareAgent(chatModel, backend),
		OnAgentEvents: makeOnAgentEvents(backend),
	})

	// Push the single task before starting.
	loop.Push(task)

	// Start the loop (non-blocking).
	loop.Run(ctx)

	// Block until the loop exits.
	// The loop will stop when either:
	// - OnAgentEvents detects the completion promise and calls Stop()
	// - GenInput detects max turns exceeded and calls Stop()
	result := loop.Wait()

	// --- Summary ---
	fmt.Println()
	fmt.Println("=== Ralph Loop Complete ===")
	fmt.Printf("Stop cause:  %s\n", result.StopCause)
	fmt.Printf("Turns:       %d/%d\n", task.Turn, task.MaxTurns)
	fmt.Printf("Complete:    %v\n", task.Complete)
	if result.ExitReason != nil {
		fmt.Printf("Exit error:  %v\n", result.ExitReason)
	}

	// Print the final state of the in-memory filesystem.
	fmt.Println()
	fmt.Println("=== Final Filesystem State ===")
	const projectDir = "/project"
	files, err := backend.LsInfo(ctx, &filesystem.LsInfoRequest{Path: projectDir})
	if err != nil {
		log.Printf("ls %s: %v", projectDir, err)
		return
	}
	for _, f := range files {
		fullPath := projectDir + "/" + f.Path
		fmt.Printf("  %s", f.Path)
		if f.IsDir {
			fmt.Println("/")
			continue
		}
		content, err := backend.Read(ctx, &filesystem.ReadRequest{FilePath: fullPath})
		if err != nil {
			fmt.Printf("  (read error: %v)\n", err)
			continue
		}
		lines := strings.Count(content.Content, "\n") + 1
		fmt.Printf("  (%d lines)\n", lines)
	}
}

// makeGenInput returns the GenInput callback.
// It acts as the turn gate: checks if the task is already complete or if
// max turns have been reached, and stops the loop accordingly.
// Otherwise it re-injects the same prompt for another agent turn.
func makeGenInput() func(ctx context.Context, loop *adk.TurnLoop[*Task], items []*Task) (*adk.GenInputResult[*Task], error) {
	return func(ctx context.Context, loop *adk.TurnLoop[*Task], items []*Task) (*adk.GenInputResult[*Task], error) {
		task := items[0]

		// Already marked complete by OnAgentEvents?
		if task.Complete {
			loop.Stop(adk.WithStopCause("completion promise detected"))
			return &adk.GenInputResult[*Task]{
				Input:     &adk.AgentInput{Messages: []adk.Message{schema.UserMessage("done")}},
				Remaining: items,
			}, nil
		}

		// Max turns exceeded?
		if task.Turn >= task.MaxTurns {
			loop.Stop(adk.WithStopCause("max turns reached"))
			return &adk.GenInputResult[*Task]{
				Input:     &adk.AgentInput{Messages: []adk.Message{schema.UserMessage("done")}},
				Remaining: items,
			}, nil
		}

		task.Turn++
		fmt.Printf("\n=== Turn %d/%d (same prompt re-injected) ===\n", task.Turn, task.MaxTurns)
		fmt.Println("The agent gets a fresh context window but sees prior work via the filesystem.")

		// Feed the SAME prompt every iteration. The agent discovers prior work
		// by using the filesystem tools (ls, read_file, etc.).
		return &adk.GenInputResult[*Task]{
			Input:    &adk.AgentInput{Messages: []adk.Message{schema.UserMessage(task.Prompt)}},
			Consumed: items,
		}, nil
	}
}

// makePrepareAgent returns the PrepareAgent callback.
// Each turn creates a FRESH ChatModelAgent (fresh context window) with
// filesystem tools, but the InMemoryBackend is shared — so files written in
// previous iterations are visible.
func makePrepareAgent(
	chatModel cmodel.BaseChatModel,
	backend *filesystem.InMemoryBackend,
) func(ctx context.Context, loop *adk.TurnLoop[*Task], consumed []*Task) (adk.Agent, error) {
	return func(ctx context.Context, loop *adk.TurnLoop[*Task], consumed []*Task) (adk.Agent, error) {
		fsMw, err := fsmiddleware.New(ctx, &fsmiddleware.MiddlewareConfig{
			Backend: backend,
		})
		if err != nil {
			return nil, fmt.Errorf("create filesystem middleware: %w", err)
		}

		return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
			Name: "ralph",
			Instruction: "You are an autonomous AI developer. " +
				"Use the filesystem tools to read and write code files. " +
				"Follow the task instructions precisely. " +
				"When all deliverables are complete, output exactly: " + completionPromise,
			Model:    chatModel,
			Handlers: []adk.ChatModelAgentMiddleware{fsMw},
			ModelRetryConfig: &adk.ModelRetryConfig{
				MaxRetries: 3,
				BackoffFunc: func(_ context.Context, attempt int) time.Duration {
					// Aggressive backoff for rate limits: 5s, 10s, 20s.
					return time.Duration(5<<(attempt-1)) * time.Second
				},
			},
			Middlewares: []adk.AgentMiddleware{
				{
					// Catch tool errors (e.g. "file not found") and return them as
					// string results so the model can see the error and adapt, instead
					// of treating them as fatal graph errors.
					WrapToolCall: compose.ToolMiddleware{
						Invokable: func(next compose.InvokableToolEndpoint) compose.InvokableToolEndpoint {
							return func(ctx context.Context, input *compose.ToolInput) (*compose.ToolOutput, error) {
								out, err := next(ctx, input)
								if err != nil {
									return &compose.ToolOutput{Result: fmt.Sprintf("Error: %v", err)}, nil
								}
								return out, nil
							}
						},
					},
				},
			},
		})
	}
}

// makeOnAgentEvents returns the OnAgentEvents callback — the "Stop Hook".
// It drains agent events, collects the text output, and checks for the
// completion promise. If found, it runs a verification gate (grep for
// remaining BUG: markers) before accepting. If not, re-pushes the task.
func makeOnAgentEvents(backend *filesystem.InMemoryBackend) func(ctx context.Context, tc *adk.TurnContext[*Task], events *adk.AsyncIterator[*adk.AgentEvent]) error {
	return func(ctx context.Context, tc *adk.TurnContext[*Task], events *adk.AsyncIterator[*adk.AgentEvent]) error {
		task := tc.Consumed[0]
		var response strings.Builder

		for {
			event, ok := events.Next()
			if !ok {
				break
			}
			if event.Err != nil {
				return event.Err
			}

			// Print the event for observability.
			prints.Event(event)

			// Collect text output from non-streaming message events.
			if event.Output != nil && event.Output.MessageOutput != nil {
				if m := event.Output.MessageOutput.Message; m != nil && m.Content != "" {
					response.WriteString(m.Content)
				}
			}
		}

		output := response.String()

		// === THE STOP HOOK ===
		// Check for the completion promise in the agent's output.
		if strings.Contains(output, completionPromise) {
			// === VERIFICATION GATE ===
			// Before accepting completion, grep for remaining BUG: markers
			// in the project files. This simulates an external test/lint check.
			bugs, _ := backend.GrepRaw(ctx, &filesystem.GrepRequest{
				Path:    "/project",
				Pattern: "BUG:",
			})
			if len(bugs) > 0 {
				fmt.Printf("\n✗ Completion promise rejected! %d BUG markers remaining:\n", len(bugs))
				for _, b := range bugs {
					fmt.Printf("    %s:%d: %s\n", b.Path, b.Line, strings.TrimSpace(b.Content))
				}
				fmt.Printf("  Re-pushing for another turn.\n")
				tc.Loop.Push(task)
				return nil
			}

			fmt.Printf("\n✓ Completion promise accepted — all BUG markers resolved!\n")
			task.Complete = true
			tc.Loop.Stop(adk.WithStopCause("completion promise detected"))
			return nil
		}

		// No completion promise — re-push the task for another turn.
		fmt.Printf("\n⟳ No completion promise found. Re-pushing for next turn.\n")
		tc.Loop.Push(task)
		return nil
	}
}
