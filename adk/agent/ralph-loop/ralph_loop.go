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
	// across turns; each turn gets a fresh prompt so the agent sees a clean
	// context window.
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
	return &RalphLoop{config: config}
}

// runOneTurn executes a single agent turn via Runner and returns the
// concatenated text output from the agent.
func (rl *RalphLoop) runOneTurn(ctx context.Context) (output string, err error) {
	cfg := rl.config

	runner := adk.NewTypedRunner(adk.RunnerConfig{
		Agent: cfg.Agent,
	})

	iter := runner.Run(ctx, []*schema.Message{schema.UserMessage(cfg.Prompt)})

	var response strings.Builder
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return "", event.Err
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

	return response.String(), nil
}

// Run executes the Ralph Loop synchronously, returning when the loop completes.
// The outer loop drives turn iteration and validates completion externally.
func (rl *RalphLoop) Run(ctx context.Context) *RalphLoopResult {
	cfg := rl.config

	for turn := 1; turn <= cfg.MaxTurns; turn++ {
		if ctx.Err() != nil {
			return &RalphLoopResult{Turns: turn - 1, MaxTurns: cfg.MaxTurns, StopCause: "context cancelled", Err: ctx.Err()}
		}

		fmt.Printf("\n=== Turn %d/%d ===\n", turn, cfg.MaxTurns)

		output, err := rl.runOneTurn(ctx)
		if err != nil {
			return &RalphLoopResult{Turns: turn, MaxTurns: cfg.MaxTurns, Err: err}
		}

		// Validate result externally.
		if strings.Contains(output, cfg.CompletionPromise) {
			if cfg.VerifyCompletion != nil {
				if verifyErr := cfg.VerifyCompletion(ctx); verifyErr != nil {
					fmt.Printf("\n✗ Completion rejected: %v. Continuing.\n", verifyErr)
					continue
				}
			}
			fmt.Printf("\n✓ Completion accepted!\n")
			return &RalphLoopResult{Complete: true, Turns: turn, MaxTurns: cfg.MaxTurns, StopCause: "completion promise accepted"}
		}

		fmt.Printf("\n⟳ No completion promise. Continuing.\n")
	}

	return &RalphLoopResult{Turns: cfg.MaxTurns, MaxTurns: cfg.MaxTurns, StopCause: "max turns reached"}
}
