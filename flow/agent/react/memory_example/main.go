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
	"os"
	"sync"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/components/tool/middlewares/errorremover"
	"github.com/cloudwego/eino-examples/flow/agent/react/memory_example/memory"
	"github.com/cloudwego/eino-examples/flow/agent/react/tools"
)

func main() {
	ctx := context.Background()

	apiKey := os.Getenv("OPENAI_API_KEY")
	modelName := os.Getenv("OPENAI_MODEL")
	baseURL := os.Getenv("OPENAI_BASE_URL")
	isAzure := os.Getenv("OPENAI_BY_AZURE") == "true"

	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{APIKey: apiKey, Model: modelName, BaseURL: baseURL, ByAzure: isAzure})
	if err != nil {
		panic(err)
	}

	sys := "你是一个简洁的助手。请在多轮对话中保持上下文。当用户询问餐厅或菜品时，请使用工具查询。"

	restaurantTool := tools.GetRestaurantTool()
	dishTool := tools.GetDishTool()

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		Model: model,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools:               []tool.BaseTool{restaurantTool, dishTool},
			ToolCallMiddlewares: []compose.ToolMiddleware{errorremover.Middleware()},
		},
		MessageModifier: func(_ context.Context, input []*schema.Message) []*schema.Message {
			return append([]*schema.Message{schema.SystemMessage(sys)}, input...)
		},
	})
	if err != nil {
		panic(err)
	}

	store := memory.NewInMemoryStore()
	sessionID := "session:demo"

	verifyGobRoundTrip()

	run := func(turn string) {
		fmt.Println("\n========== Turn Start ==========")
		fmt.Printf("[User Input] %s\n", turn)

		prev, _ := store.Read(ctx, sessionID)
		fmt.Printf("[Restored %d messages]\n", len(prev))
		for i, m := range prev {
			if len(m.ToolCalls) > 0 {
				for _, tc := range m.ToolCalls {
					fmt.Printf("  [%d] role=%s tool_call=%s args=%s\n", i, m.Role, tc.Function.Name, truncateRunes(tc.Function.Arguments, 60))
				}
			} else if m.Role == schema.Tool {
				fmt.Printf("  [%d] role=%s tool=%s result=%s\n", i, m.Role, m.ToolName, truncateRunes(m.Content, 60))
			} else {
				fmt.Printf("  [%d] role=%s content=%s\n", i, m.Role, truncateRunes(m.Content, 60))
			}
		}

		eff := append(prev, schema.UserMessage(turn))

		msgFutureOpt, msgFuture := react.WithMessageFuture()

		var wg sync.WaitGroup
		wg.Add(1)
		go func() {
			defer wg.Done()
			sr, err := agent.Stream(ctx, eff, msgFutureOpt)
			if err != nil {
				panic(err)
			}
			sr.Close()
		}()

		produced := make([]*schema.Message, 0, 4)
		iter := msgFuture.GetMessageStreams()
		idx := 0
		for {
			sr, ok, e := iter.Next()
			if e != nil {
				panic(e)
			}
			if !ok {
				break
			}
			var chunks []*schema.Message
			for {
				m, er := sr.Recv()
				if errors.Is(er, io.EOF) {
					break
				}
				if er != nil {
					panic(er)
				}
				chunks = append(chunks, m)
			}
			full, er := schema.ConcatMessages(chunks)
			if er == nil && full != nil {
				printMessage(idx, full)
				produced = append(produced, full)
			}
			idx++
		}

		wg.Wait()

		fmt.Printf("[Produced %d messages this turn]\n", len(produced))
		_ = store.Write(ctx, sessionID, append(eff, produced...))

		hits, _ := store.Query(ctx, sessionID, "restaurant", 3)
		fmt.Printf("[Query 'restaurant' hits=%d]\n", len(hits))
		for i, h := range hits {
			fmt.Printf("  hit[%d] role=%s content=%s\n", i, h.Role, truncate(h.Content, 60))
		}
		fmt.Println("========== Turn End ==========")
	}

	run("帮我找北京排名前2的餐厅。")
	run("第一家餐厅有什么菜？")
}

func printMessage(idx int, m *schema.Message) {
	switch m.Role {
	case schema.Assistant:
		if len(m.ToolCalls) > 0 {
			for _, tc := range m.ToolCalls {
				fmt.Printf("[Stream %d] role=%s tool_call=%s args=%s\n", idx, m.Role, tc.Function.Name, truncate(tc.Function.Arguments, 60))
			}
		} else {
			fmt.Printf("[Stream %d] role=%s content=%s\n", idx, m.Role, truncate(m.Content, 80))
		}
	case schema.Tool:
		fmt.Printf("[Stream %d] role=%s tool=%s result=%s\n", idx, m.Role, m.ToolName, truncate(m.Content, 80))
	default:
		fmt.Printf("[Stream %d] role=%s content=%s\n", idx, m.Role, truncate(m.Content, 80))
	}
}

func verifyGobRoundTrip() {
	msgs := []*schema.Message{
		schema.UserMessage("a"),
		schema.AssistantMessage("b", nil),
	}
	b, err := memory.EncodeMessages(msgs)
	if err != nil {
		panic(err)
	}
	out, err := memory.DecodeMessages(b)
	if err != nil {
		panic(err)
	}
	fmt.Printf("gob_round_trip=%d\n", len(out))
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

func truncateRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n]) + "..."
}
