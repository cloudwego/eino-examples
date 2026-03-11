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

package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	clc "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	localbk "github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/coze-dev/cozeloop-go"

	examplemodel "github.com/cloudwego/eino-examples/adk/common/model"
	adkstore "github.com/cloudwego/eino-examples/adk/common/store"
	commontool "github.com/cloudwego/eino-examples/adk/common/tool"
	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/mem"
)

func main() {
	var sessionID string
	var instruction string
	flag.StringVar(&sessionID, "session", "", "session ID (creates new if empty)")
	flag.StringVar(&instruction, "instruction", "", "custom instruction (empty for default)")
	flag.Parse()

	ctx := context.Background()

	// Setup CozeLoop tracing (optional)
	cozeloopApiToken := os.Getenv("COZELOOP_API_TOKEN")
	cozeloopWorkspaceID := os.Getenv("COZELOOP_WORKSPACE_ID")
	if cozeloopApiToken != "" && cozeloopWorkspaceID != "" {
		client, err := cozeloop.NewClient(
			cozeloop.WithAPIToken(cozeloopApiToken),
			cozeloop.WithWorkspaceID(cozeloopWorkspaceID),
		)
		if err != nil {
			log.Fatalf("cozeloop.NewClient failed: %v", err)
		}
		defer func() {
			time.Sleep(5 * time.Second)
			client.Close(ctx)
		}()
		callbacks.AppendGlobalHandlers(clc.NewLoopHandler(client))
		log.Println("CozeLoop tracing enabled")
	} else {
		log.Println("CozeLoop tracing disabled (set COZELOOP_API_TOKEN and COZELOOP_WORKSPACE_ID to enable)")
	}

	cm := examplemodel.NewChatModel()

	projectRoot := os.Getenv("PROJECT_ROOT")
	if projectRoot == "" {
		if cwd, err := os.Getwd(); err == nil {
			projectRoot = cwd
		}
	}
	if abs, err := filepath.Abs(projectRoot); err == nil {
		projectRoot = abs
	}

	defaultInstruction := fmt.Sprintf(`You are a helpful assistant that helps users learn the Eino framework.

IMPORTANT: When using filesystem tools (ls, read_file, glob, grep, etc.), you MUST use absolute paths.

The project root directory is: %s

- When the user asks to list files in "current directory", use path: %s
- When the user asks to read a file with a relative path, convert it to absolute path by prepending %s
- Example: if user says "read main.go", you should call read_file with file_path: "%s/main.go"

Always use absolute paths when calling filesystem tools.`, projectRoot, projectRoot, projectRoot, projectRoot)

	agentInstruction := defaultInstruction
	if instruction != "" {
		agentInstruction = instruction
	}

	backend, err := localbk.NewBackend(ctx, &localbk.Config{})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	agent, err := deep.New(ctx, &deep.Config{
		Name:           "Ch07InterruptAgent",
		Description:    "ChatWithDoc agent with interrupt/resume support.",
		ChatModel:      cm,
		Instruction:    agentInstruction,
		Backend:        backend,
		StreamingShell: backend,
		MaxIteration:   50,
		Handlers: []adk.ChatModelAgentMiddleware{
			&approvalMiddleware{},
			&safeToolMiddleware{},
		},
		ModelRetryConfig: &adk.ModelRetryConfig{
			MaxRetries: 5,
			IsRetryAble: func(_ context.Context, err error) bool {
				return strings.Contains(err.Error(), "429") ||
					strings.Contains(err.Error(), "Too Many Requests") ||
					strings.Contains(err.Error(), "qpm limit")
			},
		},
	})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
		CheckPointStore: adkstore.NewInMemoryStore(),
	})

	sessionDir := os.Getenv("SESSION_DIR")
	if sessionDir == "" {
		sessionDir = "./data/sessions"
	}

	store, err := mem.NewStore(sessionDir)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if sessionID == "" {
		sessionID = uuid.New().String()
		fmt.Printf("Created new session: %s\n", sessionID)
	} else {
		fmt.Printf("Resuming session: %s\n", sessionID)
	}

	session, err := store.GetOrCreate(sessionID)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("Session title: %s\n", session.Title())
	fmt.Printf("Project root: %s\n", projectRoot)
	fmt.Println("Enter your message (empty line to exit):")

	reader := bufio.NewReader(os.Stdin)
	checkPointID := sessionID
	for {
		_, _ = fmt.Fprint(os.Stdout, "you> ")
		line, err := reader.ReadString('\n')
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		line = strings.TrimSpace(line)
		if line == "" {
			break
		}

		userMsg := schema.UserMessage(line)
		if err := session.Append(userMsg); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		history := session.GetMessages()
		events := runner.Run(ctx, history, adk.WithCheckPointID(checkPointID))
		content, interruptInfo, err := printAndCollectAssistantFromEvents(events)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if interruptInfo != nil {
			content, err = handleInterrupt(ctx, runner, checkPointID, interruptInfo, reader)
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}

		assistantMsg := schema.AssistantMessage(content, nil)
		if err := session.Append(assistantMsg); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	fmt.Printf("\nSession saved: %s\n", sessionID)
	fmt.Printf("Resume with: go run ./cmd/ch07 --session %s\n", sessionID)
}

type approvalMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
}

func (m *approvalMiddleware) WrapInvokableToolCall(
	_ context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	if tCtx.Name != "execute" {
		return endpoint, nil
	}
	return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		wasInterrupted, _, storedArgs := tool.GetInterruptState[string](ctx)
		if !wasInterrupted {
			return "", tool.StatefulInterrupt(ctx, &commontool.ApprovalInfo{
				ToolName:        tCtx.Name,
				ArgumentsInJSON: args,
			}, args)
		}

		isTarget, hasData, data := tool.GetResumeContext[*commontool.ApprovalResult](ctx)
		if isTarget && hasData {
			if data.Approved {
				return endpoint(ctx, storedArgs, opts...)
			}
			if data.DisapproveReason != nil {
				return fmt.Sprintf("tool '%s' disapproved: %s", tCtx.Name, *data.DisapproveReason), nil
			}
			return fmt.Sprintf("tool '%s' disapproved", tCtx.Name), nil
		}

		isTarget, _, _ = tool.GetResumeContext[any](ctx)
		if !isTarget {
			return "", tool.StatefulInterrupt(ctx, &commontool.ApprovalInfo{
				ToolName:        tCtx.Name,
				ArgumentsInJSON: storedArgs,
			}, storedArgs)
		}

		return endpoint(ctx, storedArgs, opts...)
	}, nil
}

func (m *approvalMiddleware) WrapStreamableToolCall(
	_ context.Context,
	endpoint adk.StreamableToolCallEndpoint,
	tCtx *adk.ToolContext,
) (adk.StreamableToolCallEndpoint, error) {
	if tCtx.Name != "execute" {
		return endpoint, nil
	}
	return func(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
		wasInterrupted, _, storedArgs := tool.GetInterruptState[string](ctx)
		if !wasInterrupted {
			return nil, tool.StatefulInterrupt(ctx, &commontool.ApprovalInfo{
				ToolName:        tCtx.Name,
				ArgumentsInJSON: args,
			}, args)
		}

		isTarget, hasData, data := tool.GetResumeContext[*commontool.ApprovalResult](ctx)
		if isTarget && hasData {
			if data.Approved {
				return endpoint(ctx, storedArgs, opts...)
			}
			if data.DisapproveReason != nil {
				return singleChunkReader(fmt.Sprintf("tool '%s' disapproved: %s", tCtx.Name, *data.DisapproveReason)), nil
			}
			return singleChunkReader(fmt.Sprintf("tool '%s' disapproved", tCtx.Name)), nil
		}

		isTarget, _, _ = tool.GetResumeContext[any](ctx)
		if !isTarget {
			return nil, tool.StatefulInterrupt(ctx, &commontool.ApprovalInfo{
				ToolName:        tCtx.Name,
				ArgumentsInJSON: storedArgs,
			}, storedArgs)
		}

		return endpoint(ctx, storedArgs, opts...)
	}, nil
}

type safeToolMiddleware struct {
	*adk.BaseChatModelAgentMiddleware
}

func (m *safeToolMiddleware) WrapInvokableToolCall(
	_ context.Context,
	endpoint adk.InvokableToolCallEndpoint,
	_ *adk.ToolContext,
) (adk.InvokableToolCallEndpoint, error) {
	return func(ctx context.Context, args string, opts ...tool.Option) (string, error) {
		result, err := endpoint(ctx, args, opts...)
		if err != nil {
			if _, ok := compose.IsInterruptRerunError(err); ok {
				return "", err
			}
			return fmt.Sprintf("[tool error] %v", err), nil
		}
		return result, nil
	}, nil
}

func (m *safeToolMiddleware) WrapStreamableToolCall(
	_ context.Context,
	endpoint adk.StreamableToolCallEndpoint,
	_ *adk.ToolContext,
) (adk.StreamableToolCallEndpoint, error) {
	return func(ctx context.Context, args string, opts ...tool.Option) (*schema.StreamReader[string], error) {
		sr, err := endpoint(ctx, args, opts...)
		if err != nil {
			if _, ok := compose.IsInterruptRerunError(err); ok {
				return nil, err
			}
			return singleChunkReader(fmt.Sprintf("[tool error] %v", err)), nil
		}
		return safeWrapReader(sr), nil
	}, nil
}

func singleChunkReader(msg string) *schema.StreamReader[string] {
	r, w := schema.Pipe[string](1)
	_ = w.Send(msg, nil)
	w.Close()
	return r
}

func safeWrapReader(sr *schema.StreamReader[string]) *schema.StreamReader[string] {
	r, w := schema.Pipe[string](64)
	go func() {
		defer w.Close()
		for {
			chunk, err := sr.Recv()
			if errors.Is(err, io.EOF) {
				return
			}
			if err != nil {
				_ = w.Send(fmt.Sprintf("\n[tool error] %v", err), nil)
				return
			}
			_ = w.Send(chunk, nil)
		}
	}()
	return r
}

func printAndCollectAssistantFromEvents(events *adk.AsyncIterator[*adk.AgentEvent]) (string, *adk.InterruptInfo, error) {
	var sb strings.Builder
	var interruptInfo *adk.InterruptInfo

	for {
		event, ok := events.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return "", nil, event.Err
		}

		if event.Action != nil && event.Action.Interrupted != nil {
			interruptInfo = event.Action.Interrupted
			continue
		}

		if event.Output != nil && event.Output.MessageOutput != nil {
			mv := event.Output.MessageOutput
			if mv.Role == schema.Tool {
				content := drainToolResult(mv)
				fmt.Printf("[tool result] %s\n", truncate(content, 200))
				continue
			}

			if mv.Role != schema.Assistant && mv.Role != "" {
				continue
			}

			if mv.IsStreaming {
				mv.MessageStream.SetAutomaticClose()
				var accumulatedToolCalls []schema.ToolCall
				for {
					frame, err := mv.MessageStream.Recv()
					if errors.Is(err, io.EOF) {
						break
					}
					if err != nil {
						return "", nil, err
					}
					if frame != nil {
						if frame.Content != "" {
							sb.WriteString(frame.Content)
							_, _ = fmt.Fprint(os.Stdout, frame.Content)
						}
						if len(frame.ToolCalls) > 0 {
							accumulatedToolCalls = append(accumulatedToolCalls, frame.ToolCalls...)
						}
					}
				}
				for _, tc := range accumulatedToolCalls {
					if tc.Function.Name != "" && tc.Function.Arguments != "" {
						fmt.Printf("\n[tool call] %s(%s)\n", tc.Function.Name, tc.Function.Arguments)
					}
				}
				_, _ = fmt.Fprintln(os.Stdout)
				continue
			}

			if mv.Message != nil {
				sb.WriteString(mv.Message.Content)
				_, _ = fmt.Fprintln(os.Stdout, mv.Message.Content)
				for _, tc := range mv.Message.ToolCalls {
					fmt.Printf("[tool call] %s(%s)\n", tc.Function.Name, tc.Function.Arguments)
				}
			}
		}
	}

	return sb.String(), interruptInfo, nil
}

func drainToolResult(mo *adk.MessageVariant) string {
	if mo.IsStreaming && mo.MessageStream != nil {
		var sb strings.Builder
		for {
			chunk, err := mo.MessageStream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				break
			}
			if chunk != nil && chunk.Content != "" {
				sb.WriteString(chunk.Content)
			}
		}
		return sb.String()
	}
	if mo.Message != nil {
		return mo.Message.Content
	}
	return ""
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	var result bytes.Buffer
	if err := json.Compact(&result, []byte(s)); err == nil {
		s = result.String()
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func handleInterrupt(ctx context.Context, runner *adk.Runner, checkPointID string, interruptInfo *adk.InterruptInfo, reader *bufio.Reader) (string, error) {
	for _, ic := range interruptInfo.InterruptContexts {
		if !ic.IsRootCause {
			continue
		}

		info, ok := ic.Info.(*commontool.ApprovalInfo)
		if !ok {
			continue
		}

		fmt.Printf("\n⚠️  Approval Required ⚠️\n")
		fmt.Printf("Tool: %s\n", info.ToolName)
		fmt.Printf("Arguments: %s\n", info.ArgumentsInJSON)
		fmt.Print("\nApprove this action? (y/n): ")

		response, err := reader.ReadString('\n')
		if err != nil {
			return "", fmt.Errorf("failed to read user input: %w", err)
		}
		response = strings.TrimSpace(strings.ToLower(response))

		var resumeData *commontool.ApprovalResult
		if response == "y" || response == "yes" {
			resumeData = &commontool.ApprovalResult{Approved: true}
			fmt.Println("✓ Approved, executing...")
		} else {
			resumeData = &commontool.ApprovalResult{Approved: false}
			fmt.Println("✗ Rejected")
		}

		events, err := runner.ResumeWithParams(ctx, checkPointID, &adk.ResumeParams{
			Targets: map[string]any{
				ic.ID: resumeData,
			},
		})
		if err != nil {
			return "", fmt.Errorf("failed to resume: %w", err)
		}

		content, newInterruptInfo, err := printAndCollectAssistantFromEvents(events)
		if err != nil {
			return "", err
		}

		if newInterruptInfo != nil {
			return handleInterrupt(ctx, runner, checkPointID, newInterruptInfo, reader)
		}

		return content, nil
	}

	return "", fmt.Errorf("no root cause interrupt context found")
}
