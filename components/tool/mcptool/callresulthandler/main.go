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

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino-ext/components/tool/mcp/officialmcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// main function demonstrates how to use the tool call result handler.
func main() {
	// 1. Initialize context and get tools.
	// The GetTools function is configured to use our custom toolCallResultHandler.
	ctx := context.Background()
	tools, err := GetTools(ctx)
	if err != nil {
		fmt.Println(err)
		return
	}

	// 2. Create a new ToolNode.
	// A ToolNode is a component that can execute tool calls.
	tn, _ := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
		Tools: tools,
	})

	// 3. Simulate a tool call message from an assistant.
	// This message represents a request to call the 'web_search' tool.
	msg := schema.AssistantMessage("call web_search tool", []schema.ToolCall{
		{
			ID: "1",
			Function: schema.FunctionCall{
				Name:      "web_search",
				Arguments: `{"url":"web_url"}`,
			},
		},
	})

	// 4. Invoke the ToolNode.
	// When tn.Invoke is called, it will execute the 'web_search' tool.
	// After the tool returns a result, the toolCallResultHandler will be triggered
	// to process the result before it is returned here.
	result, err := tn.Invoke(ctx, msg)
	if err != nil {
		fmt.Println(err)
		return
	}
	_ = result

}

type detailContent struct {
	Summary string
	Details string
}

const webSearchTool = "web_search"

// toolCallResultHandler is a callback function that gets executed after a tool call.
// It allows for the modification of the tool call's result before it's returned.
// This can be useful for tailoring the output, or in this case,
// condensing the result to save on token usage.
func toolCallResultHandler(ctx context.Context, name string, result *mcp.CallToolResult) (*mcp.CallToolResult, error) {
	// First, check if the tool call resulted in an error.
	if result.IsError {
		marshaledResult, err := sonic.MarshalString(result)
		if err != nil {
			return nil, err
		}
		// If there was an error, return it to be handled upstream.
		return nil, fmt.Errorf("failed to call official mcp tool, mcp server return error: %s", marshaledResult)
	}

	// We're specifically interested in post-processing the 'web_search' tool's output.
	if name == webSearchTool && len(result.Content) > 0 {
		// The output format of the 'web_search' tool is known and consistent.
		// It is expected to return a single content block, which is why we can safely access the first element.
		content := result.Content[0]
		// We also know that the content will be of type TextContent.
		if textContent, ok := content.(*mcp.TextContent); ok {
			detailCt := detailContent{}
			// The Text field contains a JSON string with 'Summary' and 'Details'. We unmarshal it.
			err := sonic.UnmarshalString(textContent.Text, &detailCt)
			if err != nil {
				return nil, err
			}

			// To reduce token consumption for the language model, if the 'Details' are too long (over 1000 chars),
			// we replace the content with the shorter 'Summary'.
			if len(detailCt.Details) > 1000 {
				textContent.Text = detailCt.Summary
			} else {
				textContent.Text = detailCt.Details
			}

			// Update the result content with the potentially modified text.
			result.Content[0] = textContent
		}
	}

	// Return the (possibly modified) result.
	return result, nil
}

// GetTools initializes and returns a list of tools.
// It hooks in the toolCallResultHandler to process the results of any tool calls.
func GetTools(ctx context.Context) ([]tool.BaseTool, error) {
	// officialmcp.GetTools is used to get the official MCP tools.
	// We provide a custom configuration to it.
	tools, err := officialmcp.GetTools(ctx, &officialmcp.Config{
		// ToolCallResultHandler is a field in the config that takes a function.
		// This function will be called with the result of every tool call.
		ToolCallResultHandler: toolCallResultHandler,
	})
	if err != nil {
		return nil, err
	}
	return tools, nil
}
