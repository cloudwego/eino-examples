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
	"log"
	"os"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/dynamictool/toolsearch"

	"github.com/cloudwego/eino-examples/adk/common/prints"
)

func main() {
	ctx := context.Background()

	weatherTools := createWeatherTools()
	financeTools := createFinanceTools()
	allDynamicTools := append(weatherTools, financeTools...)

	toolSearchMiddleware, err := toolsearch.New(ctx, &toolsearch.Config{
		DynamicTools: allDynamicTools,
	})
	if err != nil {
		fmt.Printf("failed to create tool search middleware: %v\n", err)
		return
	}

	chatModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey: os.Getenv("ARK_API_KEY"),
		Model:  os.Getenv("ARK_MODEL"),
	})
	if err != nil {
		log.Fatal(err)
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "tool_search_agent",
		Description: "An agent that can dynamically search and use tools from a large tool library",
		Instruction: `You are a helpful assistant.`,
		Model:       chatModel,
		Handlers: []adk.ChatModelAgentMiddleware{
			toolSearchMiddleware,
		},
	})
	if err != nil {
		fmt.Printf("failed to create agent: %v\n", err)
		return
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})
	iter := runner.Query(ctx, "What's the weather in Beijing?")
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		prints.Event(event)
	}
}
