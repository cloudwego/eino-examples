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
//   - RalphLoop wrapping TurnLoop to drive the push-based event loop
//   - InMemoryBackend + filesystem middleware for persistent file tools
//   - VerifyCompletion as the verification gate
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
	"github.com/cloudwego/eino/compose"

	commonmodel "github.com/cloudwego/eino-examples/adk/common/model"
	"github.com/cloudwego/eino-examples/adk/common/prints"
)

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
	seedBuggyProject(ctx, backend)

	// Build the agent with filesystem tools and error-tolerant tool middleware.
	fsMw, err := fsmiddleware.New(ctx, &fsmiddleware.MiddlewareConfig{
		Backend: backend,
	})
	if err != nil {
		log.Fatalf("create filesystem middleware: %v", err)
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
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
				return time.Duration(5<<(attempt-1)) * time.Second
			},
		},
		Middlewares: []adk.AgentMiddleware{
			{
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
	if err != nil {
		log.Fatalf("create agent: %v", err)
	}

	// Run the Ralph Loop.
	rl := NewRalphLoop(RalphLoopConfig{
		Agent:             agent,
		Prompt:            taskPrompt,
		MaxTurns:          10,
		CompletionPromise: completionPromise,
		VerifyCompletion: func(ctx context.Context) error {
			bugs, _ := backend.GrepRaw(ctx, &filesystem.GrepRequest{
				Path:    "/project",
				Pattern: "BUG:",
			})
			if len(bugs) > 0 {
				msg := fmt.Sprintf("%d BUG markers remaining", len(bugs))
				for _, b := range bugs {
					msg += fmt.Sprintf("\n    %s:%d: %s", b.Path, b.Line, strings.TrimSpace(b.Content))
				}
				return fmt.Errorf("%s", msg)
			}
			return nil
		},
		OnEvent: func(event *adk.AgentEvent) {
			prints.Event(event)
		},
	})

	result := rl.Run(ctx)

	// --- Summary ---
	fmt.Println()
	fmt.Println("=== Ralph Loop Complete ===")
	fmt.Printf("Stop cause:  %s\n", result.StopCause)
	fmt.Printf("Turns:       %d/%d\n", result.Turns, result.MaxTurns)
	fmt.Printf("Complete:    %v\n", result.Complete)
	if result.Err != nil {
		fmt.Printf("Exit error:  %v\n", result.Err)
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
