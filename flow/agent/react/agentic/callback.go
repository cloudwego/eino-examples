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
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino-examples/internal/logs"
	"github.com/cloudwego/eino-ext/components/model/agenticark"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model/responses"
)

func newAgenticModelCallback() *agenticModelCallback {
	return &agenticModelCallback{}
}

type agenticModelCallback struct{}

func (a *agenticModelCallback) OnEndWithStreamOutput(ctx context.Context, runInfo *callbacks.RunInfo,
	output *schema.StreamReader[*model.AgenticCallbackOutput]) context.Context {

	printHeader(fmt.Sprintf("AgenticModel Name: %s", runInfo.Name), "\033[36m")

	go func() {
		var (
			lastBlockType schema.ContentBlockType
			lineLength    int
		)

		for {
			chunk, err := output.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				logs.Errorf("read streaming output failed, err: %v", err)
				return
			}

			if chunk.Message == nil {
				continue
			}

			for _, block := range chunk.Message.ContentBlocks {
				switch block.Type {
				case schema.ContentBlockTypeReasoning:
					if lastBlockType != block.Type {
						printHeader("reasoning", "\033[33m")
						lineLength = 0
					}

					lineLength = printWrapped(block.Reasoning.Text, lineLength)

					lastBlockType = block.Type

				case schema.ContentBlockTypeFunctionToolCall:
					if lastBlockType != block.Type {
						printHeader(fmt.Sprintf("call function tool: %s", block.FunctionToolCall.Name), "\033[35m")
						lineLength = 0
					}

					lineLength = printWrapped(block.FunctionToolCall.Arguments, lineLength)

					lastBlockType = block.Type

				case schema.ContentBlockTypeServerToolCall:
					if lastBlockType != block.Type {
						printHeader(fmt.Sprintf("call server tool: %s", block.ServerToolCall.Name), "\033[35m")
						lineLength = 0
					}

					status, ok := agenticark.GetItemStatus(block)
					if ok && status != responses.ItemStatus_completed.String() {
						fmt.Print(status + "...\n\n")
					}
					if block.ServerToolCall.Arguments != nil {
						fmt.Print(responses.ItemStatus_completed.String() + "!\n\n")
						args, _ := sonic.MarshalIndent(block.ServerToolCall.Arguments, "", "  ")
						fmt.Println(string(args))
						lineLength = 0
					}

					lastBlockType = block.Type

				case schema.ContentBlockTypeServerToolResult:
					if lastBlockType != block.Type {
						printHeader(fmt.Sprintf("server tool result: %s", block.ServerToolResult.Name), "\033[32m")
						lineLength = 0
					}
					if block.ServerToolResult.Result != nil {
						res, _ := sonic.MarshalIndent(block.ServerToolResult.Result, "", "  ")
						fmt.Println(string(res))
						lineLength = 0
					}

					lastBlockType = block.Type

				case schema.ContentBlockTypeAssistantGenText:
					if lastBlockType != block.Type {
						printHeader("assistant generated text", "\033[36m")
						lineLength = 0
					}

					lineLength = printWrapped(block.AssistantGenText.Text, lineLength)

					lastBlockType = block.Type
				}
			}
		}
	}()

	return ctx
}

func printWrapped(content string, currentLength int) int {
	for _, r := range content {
		if r == '\n' {
			currentLength = 0
			fmt.Print(string(r))
			continue
		}

		w := 1
		if r > 127 {
			w = 2
		}

		if currentLength+w > 120 {
			fmt.Print("\n")
			currentLength = 0
		}

		fmt.Print(string(r))
		currentLength += w
	}
	return currentLength
}

func printHeader(content string, color string) {
	const lineLength = 60
	separator := strings.Repeat("=", lineLength)

	contentLength := len(content)
	if contentLength >= lineLength {
		// Content too long, just print it within separators
		fmt.Printf("\n\n%s%s\n%s\n%s\033[0m\n\n", color, separator, content, separator)
		return
	}

	padding := lineLength - contentLength
	leftPad := padding / 2

	// Create padding string with spaces
	padStr := strings.Repeat(" ", leftPad)

	fmt.Printf("\n\n%s%s\n%s%s\n%s\033[0m\n\n", color, separator, padStr, content, separator)
}
