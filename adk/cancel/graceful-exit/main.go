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

// Package main demonstrates graceful agent cancellation on OS interrupt (Ctrl-C)
// and subsequent resumption from a saved checkpoint.
//
// # What This Example Shows
//
// The ADK Cancel mechanism (adk.WithCancel) lets you shut down a running agent
// cleanly, without losing progress. When a CheckPointStore is configured, the
// cancel automatically persists a checkpoint. A new Runner can later resume from
// that checkpoint, continuing exactly where the agent was interrupted.
//
// The agent is configured with tools AND a nested AgentTool so it makes
// multiple ChatModel → ToolCall rounds across agent layers. This is essential:
// CancelAfterChatModel fires BETWEEN rounds (after a ChatModel call finishes),
// and WithRecursive propagates the cancel to the nested agent inside the
// AgentTool, so whichever layer's ChatModel finishes first triggers the cancel.
// This speeds up the graceful exit because the inner agent can reach its own
// safe-point independently of the outer agent.
//
// This example runs in two phases:
//
//  1. Run + Cancel — A root agent starts a multi-step research task, delegating
//     analysis to a nested sub-agent via AgentTool. When the user presses
//     Ctrl-C (SIGINT/SIGTERM), the cancel function fires with CancelAfterChatModel
//     mode, WithRecursive, and a 30-second timeout. WithRecursive propagates the
//     cancel into the nested agent — whichever layer's ChatModel finishes first
//     triggers the safe-point, speeding up the graceful exit. The checkpoint is
//     saved automatically. If no safe-point is reached within 30 seconds, the
//     cancel escalates to CancelImmediate.
//
//  2. Resume — A new Runner resumes from the saved checkpoint. The agent
//     continues from whatever layer was interrupted and completes the task.
//
// # Key ADK APIs Demonstrated
//
//   - adk.WithCancel() — creates a cancel option + cancel function pair
//   - adk.WithCheckPointID(id) — associates a checkpoint ID with the run
//   - adk.WithAgentCancelMode(adk.CancelAfterChatModel) — waits for ChatModel safe-point
//   - adk.WithRecursive() — propagates cancel to nested AgentTools for faster safe-point
//   - adk.WithAgentCancelTimeout(d) — escalates to CancelImmediate after timeout
//   - adk.CancelError — the error surfaced via AgentEvent.Err on cancellation
//   - adk.NewAgentTool — wraps an Agent as a tool for nested agent topology
//   - Runner.Resume(ctx, checkpointID) — resumes from saved checkpoint
//
// # How to Run
//
// Set the model environment variables (see adk/common/model for details).
//
// Option A — OpenAI:
//
//	export OPENAI_API_KEY=sk-...
//	export OPENAI_MODEL=gpt-4o
//	export OPENAI_BASE_URL=https://api.openai.com/v1
//
// Option B — Ark (Volcengine):
//
//	export MODEL_TYPE=ark
//	export ARK_API_KEY=...
//	export ARK_MODEL=...
//	export ARK_BASE_URL=...
//
// Then run:
//
//	go run ./adk/cancel/graceful-exit/
//
// Press Ctrl-C while the agent is working to trigger the cancel. WithRecursive
// ensures the cancel reaches whichever agent layer is active. The program will
// print the cancel info, then automatically resume and complete the task.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloudwego/eino/adk"
	einotool "github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	commonmodel "github.com/cloudwego/eino-examples/adk/common/model"
	"github.com/cloudwego/eino-examples/adk/common/store"
)

// ANSI escape codes for colored terminal output.
// System messages (cancel info, phase headers) use these to stand out from model output.
const (
	colorReset  = "\033[0m"
	colorYellow = "\033[33m" // cancel/signal messages
	colorGreen  = "\033[32m" // phase headers / success
	colorRed    = "\033[31m" // errors / escalation
	colorCyan   = "\033[36m" // model output
	colorDim    = "\033[2m"  // dimmed separator lines
)

func sysMsg(color, msg string) {
	fmt.Printf("\n%s%s%s\n", color, msg, colorReset)
}

func sysMsgf(color, format string, args ...any) {
	fmt.Printf("\n%s"+format+"%s\n", append([]any{color}, append(args, colorReset)...)...)
}

const checkpointID = "graceful-exit-demo"

// --- Mock tools for the nested agent topology ---
// The root agent uses search_web, then delegates to the analyst AgentTool.
// The analyst sub-agent uses analyze_data and summarize_findings.
// This creates multiple ChatModel calls across two agent layers, giving
// WithRecursive a window to propagate CancelAfterChatModel inward.

type searchInput struct {
	Query string `json:"query" jsonschema:"description=The search query"`
}

type analyzeInput struct {
	Content string `json:"content" jsonschema:"description=The content to analyze"`
	Aspect  string `json:"aspect" jsonschema:"description=The aspect to focus on"`
}

type summarizeInput struct {
	Findings []string `json:"findings" jsonschema:"description=List of findings to summarize"`
}

func searchWeb(_ context.Context, input *searchInput) (string, error) {
	// Simulate a slow network call — gives the user time to press Ctrl-C.
	time.Sleep(500 * time.Millisecond)
	return fmt.Sprintf("[Search results for %q]: "+
		"1. Ancient radio signals detected from Proxima Centauri system. "+
		"2. New study suggests periodic patterns in deep-space signals. "+
		"3. SETI confirms non-natural origin hypothesis for Signal X-47.", input.Query), nil
}

func analyzeData(_ context.Context, input *analyzeInput) (string, error) {
	time.Sleep(500 * time.Millisecond)
	return fmt.Sprintf("[Analysis of %q focusing on %q]: "+
		"The pattern exhibits a 73-second periodicity with mathematical structure "+
		"consistent with an artificial origin. Confidence level: 94%%.", input.Content, input.Aspect), nil
}

func summarizeFindings(_ context.Context, input *summarizeInput) (string, error) {
	time.Sleep(300 * time.Millisecond)
	return fmt.Sprintf("[Summary of %d findings]: "+
		"Multiple independent analyses confirm the signal's artificial nature. "+
		"Recommended action: escalate to priority observation queue.", len(input.Findings)), nil
}

func main() {
	ctx := context.Background()

	chatModel := commonmodel.NewChatModel()
	cpStore := store.NewInMemoryStore()

	// Create tools using utils.InferTool — these give the agent multi-round behavior.
	searchTool, err := utils.InferTool("search_web",
		"Search the web for information on a given query. Returns relevant articles.", searchWeb)
	if err != nil {
		log.Fatalf("create search tool: %v", err)
	}
	analyzeTool, err := utils.InferTool("analyze_data",
		"Analyze a piece of content focusing on a specific aspect.", analyzeData)
	if err != nil {
		log.Fatalf("create analyze tool: %v", err)
	}
	summarizeTool, err := utils.InferTool("summarize_findings",
		"Summarize a list of findings into a concise report.", summarizeFindings)
	if err != nil {
		log.Fatalf("create summarize tool: %v", err)
	}

	// --- Sub-agent: "analyst" ---
	// This agent owns the analyze + summarize tools. It is wrapped as an AgentTool
	// so the outer agent delegates analysis work to it. This creates a nested agent
	// topology where WithRecursive can propagate the cancel inward.
	analyst, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "analyst",
		Description: "Analyzes search results and produces a summary report",
		Instruction: "You are a data analyst. When given search results:\n" +
			"1. Use analyze_data to analyze the content.\n" +
			"2. Use summarize_findings to produce a final summary.\n" +
			"Always use BOTH tools before giving your answer.",
		Model: chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []einotool.BaseTool{analyzeTool, summarizeTool},
			},
		},
	})
	if err != nil {
		log.Fatalf("create analyst agent: %v", err)
	}

	// Wrap the analyst as an AgentTool. The outer agent can call it like any tool.
	analystTool := adk.NewAgentTool(ctx, analyst)

	// --- Root agent: "researcher" ---
	// Uses search_web directly and delegates analysis to the analyst AgentTool.
	// This creates the topology:
	//   researcher (ChatModel → search_web / analyst_tool)
	//     └── analyst (ChatModel → analyze_data / summarize_findings)
	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "researcher",
		Description: "A research assistant that searches and delegates analysis",
		Instruction: "You are a research assistant. When given a research topic, " +
			"you MUST follow this workflow:\n" +
			"1. Use search_web to find information.\n" +
			"2. Use the analyst tool to analyze and summarize the results.\n" +
			"3. Present the final summary to the user with your commentary.\n" +
			"Always use search_web first, then delegate to the analyst.",
		Model: chatModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []einotool.BaseTool{searchTool, analystTool},
			},
			// EmitInternalEvents surfaces the nested analyst agent's events
			// in the top-level AsyncIterator, so we can see it working in real-time.
			EmitInternalEvents: true,
		},
	})
	if err != nil {
		log.Fatalf("create agent: %v", err)
	}

	input := []adk.Message{
		schema.UserMessage("Research the topic: mysterious deep-space radio signals and their potential artificial origins."),
	}

	// =====================================================================
	// Phase 1: Run the agent with cancel support
	// =====================================================================
	sysMsg(colorGreen, "╔══════════════════════════════════════════════════════════════╗")
	sysMsg(colorGreen, "║  Phase 1: Running agent (press Ctrl-C to cancel)            ║")
	sysMsg(colorGreen, "╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
		CheckPointStore: cpStore,
	})

	// WithCancel returns:
	//   cancelOpt  — pass to Runner.Run to enable cancellation for this execution
	//   cancelFn   — call this function to request cancellation
	cancelOpt, cancelFn := adk.WithCancel()

	// WithCheckPointID associates a checkpoint ID so the cancel can persist state.
	iter := runner.Run(ctx, input, cancelOpt, adk.WithCheckPointID(checkpointID))

	// Listen for OS interrupt signals in a separate goroutine.
	// When Ctrl-C is pressed, we request a graceful cancel.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		sysMsgf(colorYellow, "⚡ Received signal: %v — requesting cancel...", sig)

		// CancelAfterChatModel: wait for the current ChatModel call to finish
		// before canceling. WithRecursive propagates the cancel to the nested
		// analyst AgentTool — whichever layer's ChatModel finishes first triggers
		// the cancel, speeding up the graceful exit.
		// WithAgentCancelTimeout: if no safe-point is reached within 30s,
		// escalate to CancelImmediate as a last resort.
		handle, contributed := cancelFn(
			adk.WithAgentCancelMode(adk.CancelAfterChatModel),
			adk.WithRecursive(),
			adk.WithAgentCancelTimeout(30*time.Second),
		)
		sysMsgf(colorYellow, "⚡ Cancel contributed: %v", contributed)

		// Wait blocks until the cancel reaches a terminal state.
		if waitErr := handle.Wait(); waitErr != nil {
			sysMsgf(colorRed, "⚡ Cancel wait result: %v", waitErr)
		} else {
			sysMsg(colorGreen, "⚡ Cancel completed successfully — checkpoint saved")
		}
	}()

	// Consume the event stream.
	canceled := drainEvents(iter)

	// Stop listening for signals — Phase 2 should not be interrupted.
	signal.Stop(sigCh)

	if !canceled {
		sysMsg(colorGreen, "Agent completed without cancellation.")
		return
	}

	// =====================================================================
	// Phase 2: Resume from the saved checkpoint
	// =====================================================================
	fmt.Println()
	sysMsg(colorGreen, "╔══════════════════════════════════════════════════════════════╗")
	sysMsg(colorGreen, "║  Phase 2: Resuming from checkpoint                          ║")
	sysMsg(colorGreen, "╚══════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// Create a new Runner with the same agent and the same CheckPointStore.
	// In production, the store would be a persistent backend (Redis, DB, etc.)
	// and the resume could happen in a different process or after a restart.
	resumeRunner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
		CheckPointStore: cpStore,
	})

	// Resume picks up from the checkpoint saved during cancellation.
	// The agent continues from whatever layer (root or nested) was interrupted.
	resumeIter, err := resumeRunner.Resume(ctx, checkpointID)
	if err != nil {
		log.Fatalf("resume failed: %v", err)
	}

	drainEvents(resumeIter)

	fmt.Println()
	sysMsg(colorGreen, "✓ Done — agent resumed and completed successfully.")
}

// drainEvents consumes all events from the iterator, printing output and
// detecting CancelError. Returns true if a CancelError was encountered.
// Events are prefixed with the agent name to distinguish top-level vs nested output.
func drainEvents(iter *adk.AsyncIterator[*adk.AgentEvent]) bool {
	for {
		event, ok := iter.Next()
		if !ok {
			return false
		}

		if event.Err != nil {
			var cancelErr *adk.CancelError
			if errors.As(event.Err, &cancelErr) {
				sysMsg(colorDim, "────────────────────────────────────────────────────────────")
				sysMsg(colorYellow, "⚡ CancelError received:")
				sysMsgf(colorYellow, "    Mode:      %v", cancelErr.Info.Mode)
				sysMsgf(colorYellow, "    Escalated: %v", cancelErr.Info.Escalated)
				sysMsg(colorDim, "────────────────────────────────────────────────────────────")
				return true
			}
			log.Printf("unexpected error: %v", event.Err)
			return false
		}

		// Determine display prefix based on which agent emitted the event.
		prefix := fmt.Sprintf("[%s] ", event.AgentName)

		// Print streamed/non-streamed message content.
		if event.Output != nil && event.Output.MessageOutput != nil {
			if s := event.Output.MessageOutput.MessageStream; s != nil {
				first := true
				for {
					chunk, recvErr := s.Recv()
					if recvErr != nil {
						if recvErr == io.EOF {
							break
						}
						// StreamCanceledError is expected when CancelImmediate fires.
						break
					}
					if first {
						fmt.Print(colorDim + prefix + colorReset + colorCyan)
						first = false
					}
					fmt.Print(chunk.Content)
				}
				if !first {
					fmt.Print(colorReset + "\n")
				}
			} else if m := event.Output.MessageOutput.Message; m != nil {
				if m.Content != "" {
					fmt.Printf("%s%s%s%s%s%s\n", colorDim, prefix, colorReset, colorCyan, m.Content, colorReset)
				}
				// Show tool call invocations for visibility.
				for _, tc := range m.ToolCalls {
					sysMsgf(colorDim, "%s  → tool call: %s(%s)", prefix, tc.Function.Name, tc.Function.Arguments)
				}
				// Show tool results.
				if m.Role == schema.Tool {
					sysMsgf(colorDim, "%s  ← tool result: %.100s...", prefix, m.Content)
				}
			}
		}
	}
}
