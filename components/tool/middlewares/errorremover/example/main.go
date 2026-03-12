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

	"github.com/cloudwego/eino-examples/components/tool/middlewares/errorremover"
)

type WebSearch struct {
	URL string `json:"url"`
}

func main() {
	ctx := context.Background()
	// 1. Create a mock "web_search" tool.
	// This tool is designed to always return an error to demonstrate the middleware's functionality.
	searcher, _ := utils.InferTool("web_search", "search content for web url", func(ctx context.Context, in *WebSearch) (string, error) {
		// The tool call always fails.
		return "", fmt.Errorf("not found web url")
	})

	// 2. Create a compose.ToolNode and inject the remove_error middleware.
	// This middleware will intercept the tool execution lifecycle.
	//
	// IMPORTANT: Middleware order is critical. To catch errors from any subsequent
	// middleware or from the tool itself, `remove_error.Middleware()` must be placed
	// at the beginning of the `ToolCallMiddlewares` slice. Any middleware placed
	// before it will not have its errors handled by this mechanism due to the
	// sequential nature of middleware execution.
	tn, _ := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
		Tools:               []tool.BaseTool{searcher},
		ToolCallMiddlewares: []compose.ToolMiddleware{errorremover.Middleware()}, // Inject the remove_error middleware.
	})

	msg := schema.AssistantMessage("", []schema.ToolCall{
		{
			ID: "1",
			Function: schema.FunctionCall{
				Name:      "web_search",
				Arguments: `{"url":"web_url"}`,
			},
		},
	})

	// 4. Simulate a tool call.
	// Although the underlying 'web_search' tool fails, the 'Invoke' call will succeed.
	// This is because the middleware catches the error and replaces the output with the result from the registered handler.
	outs, err := tn.Invoke(ctx, msg)
	if err != nil {
		fmt.Println("error:", err)
		return
	}

	// 5. Print the result.
	// The output content will be the string returned by the error handler, not the original error message.
	for _, o := range outs {
		fmt.Println("tool:", o.ToolName, "id:", o.ToolCallID, "content:", o.Content)
	}

}
