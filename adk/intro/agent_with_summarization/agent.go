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
	"log"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/summarization"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"

	"github.com/cloudwego/eino-examples/adk/common/model"
	"github.com/cloudwego/eino-examples/adk/common/prints"
	"github.com/cloudwego/eino-examples/adk/common/trace"
	"github.com/cloudwego/eino-examples/internal/logs"
)

const (
	summaryMaxTokensBefore = 10 * 1024
	agentMaxIterations     = 30
	summaryUserInstruction = `Summarize the conversation history so the long-form report writer can continue without losing context.
Preserve the user's original request, the report structure, key facts already written, tool calls and tool results, and any remaining work.
Respond with plain text only. Do not call tools.`
)

func main() {
	ctx := context.Background()

	a, err := newAgent(ctx)
	if err != nil {
		logs.Fatalf("create agent failed, err=%v", err)
	}

	traceCloseFn, startSpanFn := trace.AppendCozeLoopCallbackIfConfigured(ctx)
	defer traceCloseFn(ctx)

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true, // you can disable streaming here
		Agent:           a,
	})

	query := `Write a very long report on the history of artificial intelligence.`
	ctx, endSpanFn := startSpanFn(ctx, "Agent", query)

	iter := runner.Query(ctx, query)

	var lastMessage adk.Message
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			log.Fatal(event.Err)
		}

		prints.Event(event)

		if event.Output != nil {
			if msg, _, getErr := adk.GetMessage(event); getErr == nil {
				lastMessage = msg
			}
		}

	}

	endSpanFn(ctx, lastMessage)

	// wait for all span to be ended
	time.Sleep(10 * time.Second)
}

func newAgent(ctx context.Context) (adk.Agent, error) {
	sumMW, err := summarization.New(ctx, &summarization.Config{
		Model:           model.NewChatModel(),
		UserInstruction: summaryUserInstruction,
		Trigger: &summarization.TriggerCondition{
			ContextTokens: summaryMaxTokensBefore,
		},
		// Callback receives the full ChatModelAgentState, including Messages, for observability only.
		// Mutating before or after here does not update the agent state.
		Callback: func(ctx context.Context, before, after adk.ChatModelAgentState) error {
			log.Printf("summarization compressed messages: before=%d after=%d", len(before.Messages), len(after.Messages))
			return nil
		},
	})
	if err != nil {
		return nil, err
	}

	a, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "main_agent",
		Description: "A long-form report assistant",
		Instruction: "You are a long-form report writer working in ReAct mode.\n" +
			"Think step by step, call tools to expand content by repeating paragraphs, then synthesize a cohesive response.\n" +
			"One time call one tool, do not call multiple tools in one turn.\n" +
			"Each tool call should indicate the call number. After 20 tool calls, produce a final summary.",
		Model:    model.NewChatModel(),
		Handlers: []adk.ChatModelAgentMiddleware{sumMW},
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					NewRepeatSectionsTool(),
				},
			},
		},
		MaxIterations: agentMaxIterations,
	})
	if err != nil {
		return nil, err
	}
	return a, nil
}
