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

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	extools "github.com/cloudwego/eino-examples/flow/agent/react/unknown_tool_handler_example/tools"
)

func main() {
	ctx := context.Background()

	unknown := func(ctx context.Context, name, input string) (string, error) {
		return fmt.Sprintf("unknown tool: %s; you made it up, try again with the correct tool name", name), nil
	}

	rAgent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: &mockToolCallingModel{},
		ToolsConfig: compose.ToolsNodeConfig{
			Tools:               []tool.BaseTool{extools.SumToolFn()},
			UnknownToolsHandler: unknown,
		},
	})
	if err != nil {
		panic(err)
	}

	msg, err := rAgent.Generate(ctx, []*schema.Message{{Role: schema.User, Content: "Add 1 and 2"}}, agent.WithComposeOptions(compose.WithCallbacks(&simpleLogger{})))
	if err != nil {
		panic(err)
	}
	fmt.Println(msg.String())
}

type mockToolCallingModel struct{ step int }

func (m *mockToolCallingModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	switch m.step {
	case 0:
		m.step++
		return &schema.Message{Role: schema.Assistant, ToolCalls: []schema.ToolCall{{ID: "1", Function: schema.FunctionCall{Name: "sumx", Arguments: "{\"a\":1,\"b\":2}"}}}}, nil
	case 1:
		m.step++
		return &schema.Message{Role: schema.Assistant, ToolCalls: []schema.ToolCall{{ID: "2", Function: schema.FunctionCall{Name: "sum", Arguments: "{\"a\":1,\"b\":2}"}}}}, nil
	default:
		return &schema.Message{Role: schema.Assistant, Content: "3"}, nil
	}
}

func (m *mockToolCallingModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	return nil, fmt.Errorf("not supported")
}

func (m *mockToolCallingModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return m, nil
}

type simpleLogger struct{ callbacks.HandlerBuilder }

func (l *simpleLogger) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	return ctx
}

func (l *simpleLogger) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	fmt.Println(output)
	return ctx
}

func (l *simpleLogger) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
	output.Close()
	return ctx
}

func (l *simpleLogger) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo, input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	input.Close()
	return ctx
}

func (l *simpleLogger) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	fmt.Println(err)
	return ctx
}
