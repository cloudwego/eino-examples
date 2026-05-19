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
	"context"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/chatmodel"
	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/helpers"
	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/mem"
	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/msgops"
)

func main() {
	var sessionID string
	var instruction string
	flag.StringVar(&sessionID, "session", "", "session ID (creates new if empty)")
	flag.StringVar(&instruction, "instruction", "You are a helpful assistant.", "")
	flag.Parse()

	ctx := context.Background()
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

	agent, err := adk.NewTypedChatModelAgent[M](ctx, &adk.TypedChatModelAgentConfig[M]{
		Name:        "Ch03MemoryAgent",
		Description: "ChatModelAgent with JSONL-based persistent session.",
		Instruction: instruction,
		Model:       cm,
	})
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
		result, err := helpers.PrintAndCollect[M](events, helpers.PrintOptions{})
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		assistantMsg := msgops.NewAssistant[M](result.AssistantText, nil)
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
	fmt.Printf("Resume with: go run ./cmd/ch03 --session %s\n", sessionID)
}
