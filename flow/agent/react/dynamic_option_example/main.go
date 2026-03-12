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
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	arkModel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"

	"github.com/cloudwego/eino-examples/components/model/httptransport"
	"github.com/cloudwego/eino-examples/flow/agent/react/dynamic_option_example/dynamic"
	"github.com/cloudwego/eino-examples/flow/agent/react/tools"
	"github.com/cloudwego/eino-examples/internal/logs"
)

func main() {
	arkApiKey := os.Getenv("ARK_API_KEY")
	arkModelName := os.Getenv("ARK_MODEL_NAME")

	ctx := context.Background()

	// Create HTTP client with curl-style logging for debugging HTTP requests
	client := &http.Client{Transport: httptransport.NewCurlRT(
		http.DefaultTransport,
		httptransport.WithLogger(log.Default()),
		httptransport.WithCtxLogger(httptransport.IDCtxLogger{L: log.Default()}),
		httptransport.WithPrintAuth(false),
		httptransport.WithMaskHeaders([]string{"X-API-KEY", "API-KEY"}),
		httptransport.WithStreamLogging(true),
		httptransport.WithMaxStreamLogBytes(8192),
	)}

	// Create Ark ChatModel with custom HTTP client
	config := &ark.ChatModelConfig{
		APIKey:     arkApiKey,
		Model:      arkModelName,
		HTTPClient: client,
	}
	arkChatModel, err := ark.NewChatModel(ctx, config)
	if err != nil {
		logs.Errorf("failed to create chat model: %v", err)
		return
	}

	restaurantTool := tools.GetRestaurantTool()
	dishTool := tools.GetDishTool()

	persona := `# Character:
你是一个帮助用户推荐餐厅和菜品的助手，根据用户的需要，查询餐厅信息并推荐，查询餐厅的菜品并推荐。
`

	// Wrap the ChatModel with dynamic.ChatModel to enable dynamic option modification.
	// The GetOptionFunc will be called before each ChatModel.Generate() call,
	// allowing us to modify options based on the current iteration state.
	dynamicModel := &dynamic.ChatModel{
		Model:         arkChatModel,
		GetOptionFunc: getDynamicOptions,
	}

	// Create ReAct agent with the dynamic model
	rAgent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: dynamicModel,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: []tool.BaseTool{restaurantTool, dishTool},
		},
	})
	if err != nil {
		logs.Errorf("failed to create agent: %v", err)
		return
	}

	// Create a parent graph that wraps the ReAct agent.
	// This parent graph provides the local state (dynamic.State) that persists
	// across ReAct loop iterations. The state is accessed via compose.ProcessState
	// inside the dynamic.ChatModel wrapper.
	parentGraph := compose.NewGraph[[]*schema.Message, *schema.Message](
		compose.WithGenLocalState(func(ctx context.Context) *dynamic.State {
			return dynamic.NewState()
		}),
	)

	// Export the ReAct agent as a sub-graph and add it to the parent graph
	agentGraph, agentOpts := rAgent.ExportGraph()
	err = parentGraph.AddGraphNode("react_agent", agentGraph, agentOpts...)
	if err != nil {
		logs.Errorf("failed to add graph node: %v", err)
		return
	}
	_ = parentGraph.AddEdge(compose.START, "react_agent")
	_ = parentGraph.AddEdge("react_agent", compose.END)

	runnable, err := parentGraph.Compile(ctx, compose.WithGraphName("DynamicOptionReactAgent"))
	if err != nil {
		logs.Errorf("failed to compile graph: %v", err)
		return
	}

	messages := []*schema.Message{
		{
			Role:    schema.System,
			Content: persona,
		},
		{
			Role:    schema.User,
			Content: "我在北京，给我推荐一些菜，需要有口味辣一点的菜，至少推荐有 2 家餐厅",
		},
	}

	// Create MessageFuture to observe intermediate results (reasoning, tool calls, tool results).
	// This allows us to print the agent's thought process in real-time.
	msgFutureOpt, msgFuture := react.WithMessageFuture()

	// Process MessageFuture in a separate goroutine to print intermediate results
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		processMessageFuture(msgFuture)
	}()

	// Use Invoke instead of Stream. The MessageFuture still provides streaming
	// access to intermediate messages even when using Invoke.
	// Note: DesignateNode is used to pass the option to the specific sub-graph node.
	_, err = runnable.Invoke(ctx, messages, agent.GetComposeOptions(msgFutureOpt)[0].DesignateNode("react_agent"))
	if err != nil {
		logs.Errorf("failed to invoke: %v", err)
		return
	}

	wg.Wait()
	fmt.Printf("\n==================== Finished ====================\n")
}

// processMessageFuture reads from the MessageFuture and prints intermediate results.
// Each iteration of the ReAct loop produces multiple message streams:
// - Assistant message with reasoning and tool calls
// - Tool result messages
// - Final assistant message with the answer
func processMessageFuture(msgFuture react.MessageFuture) {
	iter := msgFuture.GetMessageStreams()
	for {
		sr, ok, err := iter.Next()
		if err != nil {
			logs.Errorf("failed to get next message stream: %v", err)
			return
		}
		if !ok {
			break
		}

		// Accumulate streaming chunks into complete content
		var reasoningBuilder strings.Builder
		var contentBuilder strings.Builder
		var toolCallsMap = make(map[int]*strings.Builder)
		var toolCallNames = make(map[int]string)
		var toolResult *struct {
			name    string
			content string
		}

		// Read all chunks from the stream
		for {
			msg, err := sr.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				logs.Errorf("failed to recv from message stream: %v", err)
				return
			}

			// Accumulate reasoning content (thinking process)
			if msg.ReasoningContent != "" {
				reasoningBuilder.WriteString(msg.ReasoningContent)
			}

			// Accumulate tool calls (function name and arguments come in separate chunks)
			if len(msg.ToolCalls) > 0 {
				for _, tc := range msg.ToolCalls {
					idx := 0
					if tc.Index != nil {
						idx = *tc.Index
					}
					if _, exists := toolCallsMap[idx]; !exists {
						toolCallsMap[idx] = &strings.Builder{}
					}
					if tc.Function.Name != "" {
						toolCallNames[idx] = tc.Function.Name
					}
					toolCallsMap[idx].WriteString(tc.Function.Arguments)
				}
			}

			// Capture tool result
			if msg.Role == schema.Tool && msg.Content != "" {
				toolResult = &struct {
					name    string
					content string
				}{
					name:    msg.ToolName,
					content: msg.Content,
				}
			}

			// Accumulate assistant content (final answer)
			if msg.Role == schema.Assistant && msg.Content != "" {
				contentBuilder.WriteString(msg.Content)
			}
		}

		// Print accumulated content
		if reasoningBuilder.Len() > 0 {
			fmt.Printf("\n[Reasoning]\n%s\n", reasoningBuilder.String())
		}

		if len(toolCallsMap) > 0 {
			for idx := 0; idx < len(toolCallsMap); idx++ {
				if builder, exists := toolCallsMap[idx]; exists {
					name := toolCallNames[idx]
					fmt.Printf("\n[ToolCall] %s(%s)\n", name, builder.String())
				}
			}
		}

		if toolResult != nil {
			fmt.Printf("\n[ToolResult] %s:\n%s\n", toolResult.name, truncateString(toolResult.content, 300))
		}

		if contentBuilder.Len() > 0 && len(toolCallsMap) == 0 {
			fmt.Printf("\n[FinalAnswer]\n%s\n", contentBuilder.String())
		}
	}
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// getDynamicOptions is called before each ChatModel.Generate() call.
// It demonstrates how to dynamically modify options based on the current iteration:
// - Iteration 0: Enable thinking mode, allow tool calls
// - Iteration 1+: Disable thinking mode, forbid tool calls to force final answer
func getDynamicOptions(ctx context.Context, input []*schema.Message, state *dynamic.State) []model.Option {
	var opts []model.Option

	fmt.Printf("\n--- [DynamicOption] Preparing options for iteration %d ---\n", state.Iteration)

	// Control thinking mode based on iteration
	if state.Iteration >= 1 {
		fmt.Printf("  -> Disabling thinking mode\n")
		opts = append(opts, ark.WithThinking(&arkModel.Thinking{
			Type: arkModel.ThinkingTypeDisabled,
		}))
	} else {
		fmt.Printf("  -> Thinking mode enabled (first iteration)\n")
	}

	// Control tool choice based on iteration
	// After the first iteration, forbid tool calls to force the model to give a final answer
	if state.Iteration >= 1 {
		fmt.Printf("  -> Forcing final answer (tool_choice=forbidden)\n")
		opts = append(opts, model.WithToolChoice(schema.ToolChoiceForbidden))
		opts = append(opts, model.WithTools([]*schema.ToolInfo{}))
	} else {
		fmt.Printf("  -> Tool choice: auto\n")
		opts = append(opts, model.WithToolChoice(schema.ToolChoiceAllowed))
		// Re-bind tools for the first iteration
		restaurantTool := tools.GetRestaurantTool()
		dishTool := tools.GetDishTool()
		info1, _ := restaurantTool.Info(ctx)
		info2, _ := dishTool.Info(ctx)
		opts = append(opts, model.WithTools([]*schema.ToolInfo{info1, info2}))
	}

	return opts
}
