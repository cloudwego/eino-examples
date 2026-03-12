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

// This example shows how to configure the jsonfix middleware on a ToolsNode
// to repair invalid JSON arguments before invoking a local tool.
// Run: go run ./components/tool/middlewares/jsonfix/example
package main

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	jsonfix "github.com/cloudwego/eino-examples/components/tool/middlewares/jsonfix"
)

type greetReq struct {
	Name string `json:"name"`
}

type greetResp struct {
	Greeting string `json:"greeting"`
}

func main() {
	ctx := context.Background()

	// Define a simple local tool with JSON input/output using InferTool.
	greeter, _ := utils.InferTool("greeter", "greet by name", func(ctx context.Context, in *greetReq) (*greetResp, error) {
		return &greetResp{Greeting: "Hello, " + in.Name}, nil
	})

	// Create ToolsNode and register the jsonfix middleware.
	tn, _ := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
		Tools:               []tool.BaseTool{greeter},
		ToolCallMiddlewares: []compose.ToolMiddleware{jsonfix.Middleware()},
	})

	// Craft an Assistant message with an invalid JSON argument to simulate LLM output.
	msg := schema.AssistantMessage("", nil)
	msg.ToolCalls = []schema.ToolCall{{
		ID: "1",
		Function: schema.FunctionCall{
			Name:      "greeter",
			Arguments: "noise <|FunctionCallBegin|>{\"name\":\"Alice\"1\"\"}<|FunctionCallEnd|>",
		},
	}}

	// ToolsNode invokes the tool. Middleware repairs the argument first.
	outs, err := tn.Invoke(ctx, msg)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	for _, o := range outs {
		fmt.Println("tool:", o.ToolName, "id:", o.ToolCallID, "content:", o.Content)
	}
}
