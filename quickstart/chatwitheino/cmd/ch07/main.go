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

	localbk "github.com/cloudwego/eino-ext/adk/backend/local"
	clc "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	"github.com/coze-dev/cozeloop-go"

	adkstore "github.com/cloudwego/eino-examples/adk/common/store"
	commontool "github.com/cloudwego/eino-examples/adk/common/tool"
	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/chatmodel"
	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/helpers"
	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/mem"
	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/msgops"
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

	switch msgops.KindFromEnv() {
	case msgops.KindAgentic:
		runTyped[*schema.AgenticMessage](ctx, sessionID, instruction)
	default:
		runTyped[*schema.Message](ctx, sessionID, instruction)
	}
}

func runTyped[M adk.MessageType](ctx context.Context, sessionID, instruction string) {
	cm, err := chatmodel.NewModel[M](ctx)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

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

	cfg := &deep.TypedConfig[M]{
		Name:           "Ch07InterruptAgent",
		Description:    "ChatWithDoc agent with interrupt/resume support.",
		ChatModel:      cm,
		Instruction:    agentInstruction,
		Backend:        backend,
		StreamingShell: backend,
		MaxIteration:   50,
		Handlers: []adk.TypedChatModelAgentMiddleware[M]{
			newApprovalMiddleware[M](),
			helpers.NewSafeToolMiddleware[M](),
		},
	}
	helpers.ApplyMessageModelRetry(cfg)
	agent, err := deep.NewTyped[M](ctx, cfg)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	runner := adk.NewTypedRunner[M](adk.TypedRunnerConfig[M]{
		Agent:           agent,
		EnableStreaming: true,
		CheckPointStore: adkstore.NewInMemoryStore(),
	})

	sessionDir := msgops.DefaultSessionDir(msgops.KindOf[M]())

	store, err := mem.NewStore[M](sessionDir)
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

		userMsg := msgops.NewUser[M](line)
		if err := session.Append(userMsg); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		history := session.GetMessages()
		events := runner.Run(ctx, msgops.NormalizeMessagesForModelInput(history), adk.WithCheckPointID(checkPointID))
		content, interruptInfo, err := printAndCollectAssistantFromEvents[M](events)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		if interruptInfo != nil {
			content, err = handleInterrupt[M](ctx, runner, checkPointID, interruptInfo, reader)
			if err != nil {
				_, _ = fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}

		assistantMsg := msgops.NewAssistant[M](content, nil)
		if err := session.Append(assistantMsg); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	fmt.Printf("\nSession saved: %s\n", sessionID)
	fmt.Printf("Resume with: go run ./cmd/ch07 --session %s\n", sessionID)
}

type approvalMiddleware[M adk.MessageType] struct {
	*adk.TypedBaseChatModelAgentMiddleware[M]
}

func newApprovalMiddleware[M adk.MessageType]() adk.TypedChatModelAgentMiddleware[M] {
	return &approvalMiddleware[M]{
		TypedBaseChatModelAgentMiddleware: &adk.TypedBaseChatModelAgentMiddleware[M]{},
	}
}

func (m *approvalMiddleware[M]) WrapInvokableToolCall(
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

func (m *approvalMiddleware[M]) WrapStreamableToolCall(
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
				return helpers.SingleChunkReader(fmt.Sprintf("tool '%s' disapproved: %s", tCtx.Name, *data.DisapproveReason)), nil
			}
			return helpers.SingleChunkReader(fmt.Sprintf("tool '%s' disapproved", tCtx.Name)), nil
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

func printAndCollectAssistantFromEvents[M adk.MessageType](events *adk.AsyncIterator[*adk.TypedAgentEvent[M]]) (string, *adk.InterruptInfo, error) {
	var sb strings.Builder
	var interruptInfo *adk.InterruptInfo

	for {
		event, ok := events.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			if helpers.LogModelRetry(os.Stderr, event.Err) {
				continue
			}
			return "", nil, event.Err
		}

		if event.Action != nil && event.Action.Interrupted != nil {
			interruptInfo = event.Action.Interrupted
			continue
		}

		if event.Output != nil && event.Output.MessageOutput != nil {
			mv := event.Output.MessageOutput
			if msgops.VariantIsToolResult(mv) {
				content, _, _ := msgops.DrainToolResult(mv)
				fmt.Printf("[tool result] %s\n", truncate(content, 200))
				continue
			}

			if mv.IsStreaming {
				mv.MessageStream.SetAutomaticClose()
				var accumulatedToolCalls []msgops.ToolCall
				streamPrefix := sb.String()
				streamWillRetry := false
				for {
					frame, err := mv.MessageStream.Recv()
					if errors.Is(err, io.EOF) {
						break
					}
					if err != nil {
						if helpers.LogModelRetry(os.Stderr, err) {
							sb.Reset()
							sb.WriteString(streamPrefix)
							accumulatedToolCalls = nil
							streamWillRetry = true
							break
						}
						return "", nil, err
					}
					if !msgops.IsNil(frame) {
						if text := msgops.AssistantDeltaText(frame); text != "" {
							sb.WriteString(text)
							_, _ = fmt.Fprint(os.Stdout, text)
						}
						if calls := msgops.ToolCalls(frame); len(calls) > 0 {
							accumulatedToolCalls = append(accumulatedToolCalls, calls...)
						}
					}
				}
				if streamWillRetry {
					continue
				}
				for _, tc := range accumulatedToolCalls {
					if tc.Name != "" && tc.Args != "" {
						fmt.Printf("\n[tool call] %s(%s)\n", tc.Name, tc.Args)
					}
				}
				_, _ = fmt.Fprintln(os.Stdout)
				continue
			}

			if !msgops.IsNil(mv.Message) {
				content := msgops.AssistantText(mv.Message)
				sb.WriteString(content)
				_, _ = fmt.Fprintln(os.Stdout, content)
				for _, tc := range msgops.ToolCalls(mv.Message) {
					fmt.Printf("[tool call] %s(%s)\n", tc.Name, tc.Args)
				}
			}
		}
	}

	return sb.String(), interruptInfo, nil
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

func handleInterrupt[M adk.MessageType](ctx context.Context, runner *adk.TypedRunner[M], checkPointID string, interruptInfo *adk.InterruptInfo, reader *bufio.Reader) (string, error) {
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

		content, newInterruptInfo, err := printAndCollectAssistantFromEvents[M](events)
		if err != nil {
			return "", err
		}

		if newInterruptInfo != nil {
			return handleInterrupt[M](ctx, runner, checkPointID, newInterruptInfo, reader)
		}

		return content, nil
	}

	return "", fmt.Errorf("no root cause interrupt context found")
}
