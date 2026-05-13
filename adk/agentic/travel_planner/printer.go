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
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino-ext/components/model/agenticark"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

func printEvents(iter *adk.AsyncIterator[*adk.TypedAgentEvent[*schema.AgenticMessage]]) error {
	eventIndex := 0
	for {
		event, ok := iter.Next()
		if !ok {
			return nil
		}
		eventIndex++
		if event.Err != nil {
			return event.Err
		}

		fmt.Printf("\n========== event %d ==========\n", eventIndex)
		fmt.Printf("agent: %s\n", event.AgentName)
		if len(event.RunPath) > 0 {
			fmt.Printf("path: %v\n", event.RunPath)
		}

		msg, _, err := adk.TypedGetMessage(event)
		if err != nil {
			return err
		}
		if msg == nil {
			continue
		}
		printAgenticMessage(msg)
	}
}

func printAgenticMessage(msg *schema.AgenticMessage) {
	fmt.Printf("role: %s\n", msg.Role)
	for i, block := range msg.ContentBlocks {
		if block == nil {
			continue
		}
		fmt.Printf("\n[%02d] %s\n", i+1, block.Type)
		switch block.Type {
		case schema.ContentBlockTypeReasoning:
			if block.Reasoning != nil {
				fmt.Println(block.Reasoning.Text)
			}
		case schema.ContentBlockTypeAssistantGenText:
			printAssistantText(block)
		case schema.ContentBlockTypeFunctionToolCall:
			if block.FunctionToolCall != nil {
				fmt.Printf("tool: %s\nargs: %s\n", block.FunctionToolCall.Name, block.FunctionToolCall.Arguments)
			}
		case schema.ContentBlockTypeFunctionToolResult:
			if block.FunctionToolResult != nil {
				fmt.Printf("tool: %s\n", block.FunctionToolResult.Name)
				for _, part := range block.FunctionToolResult.Content {
					if part != nil && part.Text != nil {
						fmt.Println(part.Text.Text)
					}
				}
			}
		case schema.ContentBlockTypeServerToolCall:
			if block.ServerToolCall != nil {
				fmt.Printf("server tool: %s\n", block.ServerToolCall.Name)
				printJSON("args", block.ServerToolCall.Arguments)
			}
		case schema.ContentBlockTypeServerToolResult:
			if block.ServerToolResult != nil {
				fmt.Printf("server tool: %s\n", block.ServerToolResult.Name)
				printJSON("result", block.ServerToolResult.Content)
			}
		default:
			printJSON("block", block)
		}
	}

	if msg.ResponseMeta != nil {
		fmt.Println("\n[response_meta]")
		if ext, ok := msg.ResponseMeta.Extension.(*agenticark.ResponseMetaExtension); ok {
			fmt.Printf("id: %s\nstatus: %s\nprevious_response_id: %s\n", ext.ID, ext.Status, ext.PreviousResponseID)
			if ext.Thinking != nil {
				fmt.Printf("thinking: %s\n", ext.Thinking.Type)
			}
		}
		if usage := msg.ResponseMeta.TokenUsage; usage != nil {
			fmt.Printf("tokens: prompt=%d completion=%d total=%d reasoning=%d\n",
				usage.PromptTokens,
				usage.CompletionTokens,
				usage.TotalTokens,
				usage.CompletionTokensDetails.ReasoningTokens,
			)
		}
	}
}

func printAssistantText(block *schema.ContentBlock) {
	if block.AssistantGenText == nil {
		return
	}
	fmt.Println(block.AssistantGenText.Text)
	ext, ok := block.AssistantGenText.Extension.(*agenticark.AssistantGenTextExtension)
	if !ok || ext == nil || len(ext.Annotations) == 0 {
		return
	}
	fmt.Println("\nannotations:")
	for _, anno := range ext.Annotations {
		if anno == nil || anno.URLCitation == nil {
			continue
		}
		fmt.Printf("- %s: %s\n", anno.URLCitation.Title, anno.URLCitation.URL)
	}
}

func printJSON(label string, v any) {
	if v == nil {
		return
	}
	bs, err := sonic.MarshalIndent(v, "", "  ")
	if err != nil {
		fmt.Printf("%s: %#v\n", label, v)
		return
	}
	fmt.Printf("%s: %s\n", label, string(bs))
}
