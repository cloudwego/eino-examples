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

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/chatmodel"
	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/helpers"
	"github.com/cloudwego/eino-examples/quickstart/chatwitheino/msgops"
)

func main() {
	var instruction string
	flag.StringVar(&instruction, "instruction", "You are a helpful assistant.", "")
	flag.Parse()

	ctx := context.Background()
	switch msgops.KindFromEnv() {
	case msgops.KindAgentic:
		runTyped[*schema.AgenticMessage](ctx, instruction)
	default:
		runTyped[*schema.Message](ctx, instruction)
	}
}

func runTyped[M adk.MessageType](ctx context.Context, instruction string) {
	cm, err := chatmodel.NewModel[M](ctx)
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	agent, err := adk.NewTypedChatModelAgent[M](ctx, &adk.TypedChatModelAgentConfig[M]{
		Name:        "Ch02ChatModelAgent",
		Description: "A minimal ChatModelAgent with in-memory multi-turn history.",
		Instruction: instruction,
		Model:       cm,
	})
	if err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	runner := adk.NewTypedRunner(adk.TypedRunnerConfig[M]{
		Agent:           agent,
		EnableStreaming: true,
	})

	history := make([]M, 0, 16)
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
		history = append(history, msgops.NewUser[M](line))

		events := runner.Run(ctx, msgops.NormalizeMessagesForModelInput(history))
		result, err := helpers.PrintAndCollect[M](events, helpers.PrintOptions{})
		if err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		history = append(history, msgops.NewAssistant[M](result.AssistantText, nil))
	}
	if err := scanner.Err(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
