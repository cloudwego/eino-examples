/*
 * Copyright 2026 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/agenticark"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

type runResult struct {
	finalMessage *schema.AgenticMessage
	responseID   string
	text         string
}

func printStepHeader(title, subtitle string) {
	fmt.Printf("\n========== %s ==========\n", title)
	if subtitle != "" {
		fmt.Println(subtitle)
	}
}

func runAgenticAgent(ctx context.Context, runner *adk.TypedRunner[*schema.AgenticMessage], messages []*schema.AgenticMessage) (*runResult, error) {
	iter := runner.Run(ctx, messages)
	result := &runResult{}

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			return result, event.Err
		}
		if event.Output == nil || event.Output.MessageOutput == nil {
			continue
		}

		msg, err := consumeMessageVariant(event.Output.MessageOutput)
		if err != nil {
			return result, err
		}
		if msg == nil {
			continue
		}

		printAgenticMessage(msg)
		if id := responseID(msg); id != "" {
			result.finalMessage = msg
			result.responseID = id
		} else if msg.Role == schema.AgenticRoleTypeAssistant {
			result.finalMessage = msg
		}
		if text := assistantText(msg); text != "" {
			result.text = strings.TrimSpace(result.text + "\n" + text)
		}
	}

	return result, nil
}

func consumeMessageVariant(mv *adk.TypedMessageVariant[*schema.AgenticMessage]) (*schema.AgenticMessage, error) {
	if mv == nil {
		return nil, nil
	}
	if !mv.IsStreaming {
		return mv.Message, nil
	}
	if mv.MessageStream == nil {
		return nil, nil
	}

	var chunks []*schema.AgenticMessage
	for {
		chunk, err := mv.MessageStream.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if chunk != nil {
			chunks = append(chunks, chunk)
		}
	}
	if len(chunks) == 0 {
		return nil, nil
	}
	return schema.ConcatAgenticMessages(chunks)
}

func printAgenticMessage(msg *schema.AgenticMessage) {
	if msg == nil {
		return
	}
	if id := responseID(msg); id != "" {
		fmt.Printf("[response] id=%s", id)
		if meta := responseMeta(msg); meta != nil {
			if meta.Status != "" {
				fmt.Printf(" status=%s", meta.Status)
			}
			if meta.PreviousResponseID != "" {
				fmt.Printf(" previous=%s", meta.PreviousResponseID)
			}
			if meta.Thinking != nil && meta.Thinking.Type != "" {
				fmt.Printf(" thinking=%s", meta.Thinking.Type)
			}
			if meta.Error != nil {
				fmt.Printf(" error=%s", meta.Error.Message)
			}
			if meta.IncompleteDetails != nil {
				fmt.Printf(" incomplete=%s", meta.IncompleteDetails.Reason)
			}
			if meta.StreamingError != nil {
				fmt.Printf(" stream_error=%s", meta.StreamingError.Message)
			}
		}
		fmt.Println()
	}

	for _, block := range msg.ContentBlocks {
		printContentBlock(block)
	}

	printUsage(msg)
}

func printContentBlock(block *schema.ContentBlock) {
	if block == nil {
		return
	}
	switch block.Type {
	case schema.ContentBlockTypeReasoning:
		if block.Reasoning != nil {
			printPayload("reasoning", block.Reasoning.Text)
		}
	case schema.ContentBlockTypeAssistantGenText:
		if block.AssistantGenText != nil && strings.TrimSpace(block.AssistantGenText.Text) != "" {
			printPayload("assistant", block.AssistantGenText.Text)
			printTextAnnotations(block.AssistantGenText)
		}
	case schema.ContentBlockTypeFunctionToolCall:
		if block.FunctionToolCall != nil {
			printNamedPayload("tool.call", block.FunctionToolCall.Name, block.FunctionToolCall.Arguments)
		}
	case schema.ContentBlockTypeFunctionToolResult:
		if block.FunctionToolResult != nil {
			printNamedPayload("tool.result", block.FunctionToolResult.Name, functionToolResultText(block.FunctionToolResult))
		}
	case schema.ContentBlockTypeServerToolCall:
		if block.ServerToolCall != nil {
			printNamedPayload("server_tool.call", block.ServerToolCall.Name, stringify(block.ServerToolCall.Arguments))
		}
	case schema.ContentBlockTypeServerToolResult:
		if block.ServerToolResult != nil {
			printNamedPayload("server_tool.result", block.ServerToolResult.Name, formatServerToolResult(block.ServerToolResult))
		}
	default:
		printNamedPayload("block", string(block.Type), stringify(block))
	}
}

func printTextAnnotations(text *schema.AssistantGenText) {
	ext, ok := text.Extension.(*agenticark.AssistantGenTextExtension)
	if !ok || ext == nil {
		return
	}
	for i, ann := range ext.Annotations {
		if ann == nil {
			continue
		}
		switch ann.Type {
		case agenticark.TextAnnotationTypeURLCitation:
			if ann.URLCitation != nil {
				fmt.Printf("[citation.%d] %s %s\n", i+1, ann.URLCitation.Title, ann.URLCitation.URL)
			}
		case agenticark.TextAnnotationTypeDocCitation:
			if ann.DocCitation != nil {
				fmt.Printf("[citation.%d] %s\n", i+1, ann.DocCitation.DocName)
			}
		}
	}
}

func printUsage(msg *schema.AgenticMessage) {
	if msg == nil || msg.ResponseMeta == nil || msg.ResponseMeta.TokenUsage == nil {
		return
	}
	usage := msg.ResponseMeta.TokenUsage
	fmt.Printf("[usage] input=%d output=%d total=%d", usage.PromptTokens, usage.CompletionTokens, usage.TotalTokens)
	if usage.PromptTokenDetails.CachedTokens > 0 {
		fmt.Printf(" cached=%d", usage.PromptTokenDetails.CachedTokens)
	}
	if usage.CompletionTokensDetails.ReasoningTokens > 0 {
		fmt.Printf(" reasoning=%d", usage.CompletionTokensDetails.ReasoningTokens)
	}
	fmt.Println()
}

func responseID(msg *schema.AgenticMessage) string {
	if meta := responseMeta(msg); meta != nil {
		return meta.ID
	}
	return ""
}

func responseMeta(msg *schema.AgenticMessage) *agenticark.ResponseMetaExtension {
	if msg == nil || msg.ResponseMeta == nil || msg.ResponseMeta.Extension == nil {
		return nil
	}
	if ext, ok := msg.ResponseMeta.Extension.(*agenticark.ResponseMetaExtension); ok {
		return ext
	}
	return nil
}

func assistantText(msg *schema.AgenticMessage) string {
	var b strings.Builder
	for _, block := range msg.ContentBlocks {
		if block != nil && block.AssistantGenText != nil {
			b.WriteString(block.AssistantGenText.Text)
		}
	}
	return strings.TrimSpace(b.String())
}

func functionToolResultText(result *schema.FunctionToolResult) string {
	var parts []string
	for _, block := range result.Content {
		if block != nil && block.Text != nil {
			parts = append(parts, block.Text.Text)
		}
	}
	return strings.Join(parts, " ")
}

func formatServerToolResult(result *schema.ServerToolResult) string {
	ext, ok := result.Content.(*agenticark.ServerToolResult)
	if !ok || ext == nil {
		return fmt.Sprintf("%v", result.Content)
	}
	switch {
	case ext.ImageProcess != nil:
		return formatImageProcessResult(ext.ImageProcess)
	case ext.DoubaoApp != nil:
		return stringify(ext.DoubaoApp)
	default:
		return stringify(result.Content)
	}
}

func formatImageProcessResult(result *agenticark.ImageProcessResult) string {
	if result == nil {
		return ""
	}
	if result.Error != nil {
		return fmt.Sprintf("error=%s", result.Error.Message)
	}
	if result.Action != nil {
		if result.Action.ResultImageURL != "" {
			return fmt.Sprintf("action=%s image_url=%s", result.Action.Type, result.Action.ResultImageURL)
		}
		return fmt.Sprintf("action=%s", result.Action.Type)
	}
	return stringify(result)
}

func printPayload(label, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}
	if strings.Contains(value, "\n") {
		fmt.Printf("[%s]\n%s\n", label, value)
		return
	}
	fmt.Printf("[%s] %s\n", label, value)
}

func printNamedPayload(label, name, value string) {
	value = strings.TrimSpace(value)
	if value == "" {
		fmt.Printf("[%s] %s\n", label, name)
		return
	}
	fmt.Printf("[%s] %s\n%s\n", label, name, value)
}

func stringify(v any) string {
	if v == nil {
		return ""
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err == nil {
		return string(b)
	}
	return fmt.Sprintf("%+v", v)
}
