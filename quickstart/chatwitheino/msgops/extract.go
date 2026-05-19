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

package msgops

import (
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// Text returns the user-facing text found in a message.
func Text[M adk.MessageType](msg M) string {
	switch m := any(msg).(type) {
	case *schema.Message:
		return messageText(m)
	case *schema.AgenticMessage:
		return agenticText(m)
	default:
		return ""
	}
}

// AssistantText returns generated assistant text from a message.
func AssistantText[M adk.MessageType](msg M) string {
	switch m := any(msg).(type) {
	case *schema.Message:
		if m == nil {
			return ""
		}
		return messageAssistantText(m)
	case *schema.AgenticMessage:
		if m == nil {
			return ""
		}
		var parts []string
		for _, block := range m.ContentBlocks {
			if block != nil && block.Type == schema.ContentBlockTypeAssistantGenText && block.AssistantGenText != nil {
				parts = append(parts, block.AssistantGenText.Text)
			}
		}
		return strings.Join(parts, "")
	default:
		return ""
	}
}

// AssistantDeltaText returns assistant text from one streaming chunk. It
// intentionally delegates to AssistantText because schema.Message and
// AgenticMessage currently expose streamed assistant deltas through the same
// text-bearing fields; keeping this helper separate makes streaming call sites
// explicit and leaves room for chunk-specific handling later.
func AssistantDeltaText[M adk.MessageType](msg M) string {
	return AssistantText(msg)
}

// HasContent reports whether a message carries any content worth persisting.
func HasContent[M adk.MessageType](msg M) bool {
	switch m := any(msg).(type) {
	case *schema.Message:
		if m == nil {
			return false
		}
		return m.Content != "" ||
			m.ReasoningContent != "" ||
			len(m.ToolCalls) > 0 ||
			len(m.MultiContent) > 0 ||
			len(m.AssistantGenMultiContent) > 0
	case *schema.AgenticMessage:
		return m != nil && len(m.ContentBlocks) > 0
	default:
		return false
	}
}

// UserText returns text only for user-role messages.
func UserText[M adk.MessageType](msg M) string {
	switch m := any(msg).(type) {
	case *schema.Message:
		if m == nil || m.Role != schema.User {
			return ""
		}
		return messageText(m)
	case *schema.AgenticMessage:
		if m == nil || m.Role != schema.AgenticRoleTypeUser {
			return ""
		}
		var parts []string
		for _, block := range m.ContentBlocks {
			if block != nil && block.Type == schema.ContentBlockTypeUserInputText && block.UserInputText != nil {
				parts = append(parts, block.UserInputText.Text)
			}
		}
		return strings.Join(parts, "\n")
	default:
		return ""
	}
}

// ToolCalls extracts function tool calls from a message or streaming chunk.
func ToolCalls[M adk.MessageType](msg M) []ToolCall {
	switch m := any(msg).(type) {
	case *schema.Message:
		if m == nil {
			return nil
		}
		out := make([]ToolCall, 0, len(m.ToolCalls))
		for _, tc := range m.ToolCalls {
			idx := 0
			if tc.Index != nil {
				idx = *tc.Index
			}
			out = append(out, ToolCall{
				ID:    tc.ID,
				Name:  tc.Function.Name,
				Args:  tc.Function.Arguments,
				Index: idx,
			})
		}
		return out
	case *schema.AgenticMessage:
		if m == nil {
			return nil
		}
		var out []ToolCall
		for _, block := range m.ContentBlocks {
			if block == nil || block.Type != schema.ContentBlockTypeFunctionToolCall || block.FunctionToolCall == nil {
				continue
			}
			idx := 0
			if block.StreamingMeta != nil {
				idx = block.StreamingMeta.Index
			}
			out = append(out, ToolCall{
				ID:    block.FunctionToolCall.CallID,
				Name:  block.FunctionToolCall.Name,
				Args:  block.FunctionToolCall.Arguments,
				Index: idx,
			})
		}
		return out
	default:
		return nil
	}
}

// ToolResults extracts function tool results from a message or streaming chunk.
func ToolResults[M adk.MessageType](msg M) []ToolResult {
	switch m := any(msg).(type) {
	case *schema.Message:
		if m == nil || (m.Role != schema.Tool && m.ToolCallID == "") {
			return nil
		}
		return []ToolResult{{
			ID:      m.ToolCallID,
			Name:    m.ToolName,
			Content: messageText(m),
		}}
	case *schema.AgenticMessage:
		if m == nil {
			return nil
		}
		var out []ToolResult
		for _, block := range m.ContentBlocks {
			if block == nil || block.Type != schema.ContentBlockTypeFunctionToolResult || block.FunctionToolResult == nil {
				continue
			}
			out = append(out, ToolResult{
				ID:      block.FunctionToolResult.CallID,
				Name:    block.FunctionToolResult.Name,
				Content: functionToolResultText(block.FunctionToolResult),
			})
		}
		return out
	default:
		return nil
	}
}

// RoleLabel returns the human label used by chatwitheino UIs.
func RoleLabel[M adk.MessageType](msg M) string {
	switch m := any(msg).(type) {
	case *schema.Message:
		if m == nil {
			return "Agent"
		}
		switch m.Role {
		case schema.User:
			return "You"
		case schema.Assistant:
			return "Agent"
		case schema.Tool:
			return "Tool"
		case schema.System:
			return "System"
		default:
			if m.Role != "" {
				return string(m.Role)
			}
			return "Agent"
		}
	case *schema.AgenticMessage:
		if m == nil {
			return "Agent"
		}
		switch m.Role {
		case schema.AgenticRoleTypeUser:
			if len(ToolResults(msg)) > 0 {
				return "Tool"
			}
			return "You"
		case schema.AgenticRoleTypeAssistant:
			return "Agent"
		case schema.AgenticRoleTypeSystem:
			return "System"
		default:
			if m.Role != "" {
				return string(m.Role)
			}
			return "Agent"
		}
	default:
		return "Agent"
	}
}
func messageText(msg *schema.Message) string {
	if msg == nil {
		return ""
	}
	if msg.Content != "" {
		return msg.Content
	}
	var parts []string
	for _, part := range msg.UserInputMultiContent {
		if part.Type == schema.ChatMessagePartTypeText && part.Text != "" {
			parts = append(parts, part.Text)
		}
	}
	for _, part := range msg.AssistantGenMultiContent {
		if part.Type == schema.ChatMessagePartTypeText && part.Text != "" {
			parts = append(parts, part.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func messageAssistantText(msg *schema.Message) string {
	if msg == nil {
		return ""
	}
	if msg.Content != "" {
		return msg.Content
	}
	var parts []string
	for _, part := range msg.AssistantGenMultiContent {
		if part.Type == schema.ChatMessagePartTypeText && part.Text != "" {
			parts = append(parts, part.Text)
		}
	}
	return strings.Join(parts, "")
}

func agenticText(msg *schema.AgenticMessage) string {
	if msg == nil {
		return ""
	}
	var parts []string
	for _, block := range msg.ContentBlocks {
		if block == nil {
			continue
		}
		switch block.Type {
		case schema.ContentBlockTypeUserInputText:
			if block.UserInputText != nil {
				parts = append(parts, block.UserInputText.Text)
			}
		case schema.ContentBlockTypeAssistantGenText:
			if block.AssistantGenText != nil {
				parts = append(parts, block.AssistantGenText.Text)
			}
		case schema.ContentBlockTypeFunctionToolResult:
			if block.FunctionToolResult != nil {
				parts = append(parts, functionToolResultText(block.FunctionToolResult))
			}
		}
	}
	return strings.Join(parts, "\n")
}

func functionToolResultText(result *schema.FunctionToolResult) string {
	if result == nil {
		return ""
	}
	var parts []string
	for _, block := range result.Content {
		if block == nil {
			continue
		}
		switch block.Type {
		case schema.FunctionToolResultContentBlockTypeText:
			if block.Text != nil {
				parts = append(parts, block.Text.Text)
			}
		default:
			parts = append(parts, strings.TrimSpace(block.String()))
		}
	}
	return strings.Join(parts, "\n")
}
