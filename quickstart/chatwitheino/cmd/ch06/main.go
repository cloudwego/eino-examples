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
	"github.com/cloudwego/eino/schema"
	"github.com/coze-dev/cozeloop-go"

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
	// Set COZELOOP_API_TOKEN and COZELOOP_WORKSPACE_ID to enable
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
		Name:           "Ch06CallbackAgent",
		Description:    "ChatWithDoc agent with CozeLoop tracing.",
		ChatModel:      cm,
		Instruction:    agentInstruction,
		Backend:        backend,
		StreamingShell: backend,
		MaxIteration:   50,
		Handlers: []adk.TypedChatModelAgentMiddleware[M]{
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

	scanner := bufio.NewScanner(os.Stdin)
	for {
		_, _ = fmt.Fprint(os.Stdout, "you> ")
		if !scanner.Scan() {
			break
		}
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			break
		}

		userMsg := msgops.NewUser[M](line)
		if err := session.Append(userMsg); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		history := session.GetMessages()
		events := runner.Run(ctx, msgops.NormalizeMessagesForModelInput(history))
		content, err := printAndCollectAssistantFromEvents[M](events)
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		assistantMsg := msgops.NewAssistant[M](content, nil)
		if err := session.Append(assistantMsg); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}

	if err := scanner.Err(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("\nSession saved: %s\n", sessionID)
	fmt.Printf("Resume with: go run ./cmd/ch06 --session %s\n", sessionID)
}

func printAndCollectAssistantFromEvents[M adk.MessageType](events *adk.AsyncIterator[*adk.TypedAgentEvent[M]]) (string, error) {
	var sb strings.Builder

	for {
		event, ok := events.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return "", event.Err
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
				for {
					frame, err := mv.MessageStream.Recv()
					if errors.Is(err, io.EOF) {
						break
					}
					if err != nil {
						return "", err
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

	return sb.String(), nil
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
