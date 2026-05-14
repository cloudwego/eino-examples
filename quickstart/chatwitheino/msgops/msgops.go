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
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// Kind identifies the message representation used by a chatwitheino run.
type Kind string

const (
	KindMessage Kind = "message"
	KindAgentic Kind = "agentic"
)

// ToolCall contains the common function-tool-call fields used by the examples.
type ToolCall struct {
	ID    string
	Name  string
	Args  string
	Index int
}

// ToolResult contains the common function-tool-result fields used by the examples.
type ToolResult struct {
	ID      string
	Name    string
	Content string
}

// KindFromEnv reads MESSAGE_KIND. Unknown values fall back to message.
func KindFromEnv() Kind {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("MESSAGE_KIND"))) {
	case string(KindAgentic), "agenticmessage", "agentic_message":
		return KindAgentic
	default:
		return KindMessage
	}
}

// KindOf returns the message kind represented by M.
func KindOf[M adk.MessageType]() Kind {
	var zero M
	switch any(zero).(type) {
	case *schema.AgenticMessage:
		return KindAgentic
	default:
		return KindMessage
	}
}

// IsNil reports whether a generic message value is nil.
func IsNil[M adk.MessageType](msg M) bool {
	var zero M
	return any(msg) == any(zero)
}

// DefaultSessionDir returns the default session directory for the current kind.
func DefaultSessionDir(kind Kind) string {
	if kind == KindAgentic {
		if dir := strings.TrimSpace(os.Getenv("SESSION_DIR_AGENTIC")); dir != "" {
			return dir
		}
		return "./data/sessions_agentic"
	}
	if dir := strings.TrimSpace(os.Getenv("SESSION_DIR")); dir != "" {
		return dir
	}
	return "./data/sessions"
}

// NewUser constructs a user message for M.
func NewUser[M adk.MessageType](text string) M {
	if KindOf[M]() == KindAgentic {
		return any(schema.UserAgenticMessage(text)).(M)
	}
	return any(schema.UserMessage(text)).(M)
}

// NewSystem constructs a system message for M.
func NewSystem[M adk.MessageType](text string) M {
	if KindOf[M]() == KindAgentic {
		return any(schema.SystemAgenticMessage(text)).(M)
	}
	return any(schema.SystemMessage(text)).(M)
}

// NewAssistant constructs an assistant message with optional function tool calls.
func NewAssistant[M adk.MessageType](text string, calls []ToolCall) M {
	if KindOf[M]() == KindAgentic {
		blocks := make([]*schema.ContentBlock, 0, len(calls)+1)
		if text != "" {
			blocks = append(blocks, schema.NewContentBlock(&schema.AssistantGenText{Text: text}))
		}
		for _, call := range calls {
			blocks = append(blocks, schema.NewContentBlock(&schema.FunctionToolCall{
				CallID:    call.ID,
				Name:      call.Name,
				Arguments: call.Args,
			}))
		}
		return any(&schema.AgenticMessage{
			Role:          schema.AgenticRoleTypeAssistant,
			ContentBlocks: blocks,
		}).(M)
	}

	schemaCalls := make([]schema.ToolCall, 0, len(calls))
	for _, call := range calls {
		idx := call.Index
		schemaCalls = append(schemaCalls, schema.ToolCall{
			ID:       call.ID,
			Index:    &idx,
			Function: schema.FunctionCall{Name: call.Name, Arguments: call.Args},
		})
	}
	return any(schema.AssistantMessage(text, schemaCalls)).(M)
}

// NewToolResult constructs a tool-result message for M.
func NewToolResult[M adk.MessageType](id, name, content string) M {
	if KindOf[M]() == KindAgentic {
		var blocks []*schema.FunctionToolResultContentBlock
		if content != "" {
			blocks = []*schema.FunctionToolResultContentBlock{{
				Type: schema.FunctionToolResultContentBlockTypeText,
				Text: &schema.UserInputText{Text: content},
			}}
		}
		return any(&schema.AgenticMessage{
			Role: schema.AgenticRoleTypeUser,
			ContentBlocks: []*schema.ContentBlock{
				schema.NewContentBlock(&schema.FunctionToolResult{
					CallID:  id,
					Name:    name,
					Content: blocks,
				}),
			},
		}).(M)
	}
	return any(schema.ToolMessage(content, id, schema.WithToolName(name))).(M)
}

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

// AssistantDeltaText returns assistant text from one streaming chunk.
func AssistantDeltaText[M adk.MessageType](msg M) string {
	return AssistantText(msg)
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

// VariantRoleLabel returns a display label without consuming a stream.
func VariantRoleLabel[M adk.MessageType](mv *adk.TypedMessageVariant[M]) string {
	if mv == nil {
		return "Agent"
	}
	if KindOf[M]() == KindAgentic {
		switch mv.AgenticRole {
		case schema.AgenticRoleTypeUser:
			return "You"
		case schema.AgenticRoleTypeSystem:
			return "System"
		case schema.AgenticRoleTypeAssistant, "":
			return "Agent"
		default:
			return string(mv.AgenticRole)
		}
	}
	switch mv.Role {
	case schema.User:
		return "You"
	case schema.Assistant, "":
		return "Agent"
	case schema.Tool:
		return "Tool"
	case schema.System:
		return "System"
	default:
		return string(mv.Role)
	}
}

// VariantIsToolResult reports whether a message variant carries tool output.
func VariantIsToolResult[M adk.MessageType](mv *adk.TypedMessageVariant[M]) bool {
	if mv == nil {
		return false
	}
	if KindOf[M]() == KindMessage {
		if mv.Role == schema.Tool {
			return true
		}
		if !IsNil(mv.Message) {
			return len(ToolResults(mv.Message)) > 0
		}
		return false
	}
	if !IsNil(mv.Message) {
		return len(ToolResults(mv.Message)) > 0
	}
	return mv.AgenticRole == schema.AgenticRoleTypeUser
}

// DrainToolResult consumes a tool-result variant and returns its text and call ID.
func DrainToolResult[M adk.MessageType](mv *adk.TypedMessageVariant[M]) (content, id, name string) {
	if mv == nil {
		return "", "", ""
	}
	if mv.IsStreaming && mv.MessageStream != nil {
		var buf strings.Builder
		for {
			chunk, err := mv.MessageStream.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				break
			}
			for _, result := range ToolResults(chunk) {
				if id == "" {
					id = result.ID
				}
				if name == "" {
					name = result.Name
				}
				buf.WriteString(result.Content)
			}
			if text := Text(chunk); text != "" && len(ToolResults(chunk)) == 0 {
				buf.WriteString(text)
			}
		}
		return buf.String(), id, name
	}
	if IsNil(mv.Message) {
		return "", "", ""
	}
	results := ToolResults(mv.Message)
	if len(results) == 0 {
		return Text(mv.Message), "", ""
	}
	for _, result := range results {
		if id == "" {
			id = result.ID
		}
		if name == "" {
			name = result.Name
		}
		content += result.Content
	}
	return content, id, name
}

// UnmarshalMessage unmarshals one JSONL message line into M.
func UnmarshalMessage[M adk.MessageType](data []byte) (M, error) {
	if KindOf[M]() == KindAgentic {
		var msg schema.AgenticMessage
		err := json.Unmarshal(data, &msg)
		return any(&msg).(M), err
	}
	var msg schema.Message
	err := json.Unmarshal(data, &msg)
	return any(&msg).(M), err
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

// ValidateKind rejects files written for a different message representation.
func ValidateKind(stored, target Kind, legacyMessageOK bool) error {
	if stored == "" && target == KindMessage && legacyMessageOK {
		return nil
	}
	if stored == "" {
		return fmt.Errorf("session file has no message_kind; current MESSAGE_KIND=%s", target)
	}
	if stored != target {
		return fmt.Errorf("session file uses message_kind=%s; current MESSAGE_KIND=%s", stored, target)
	}
	return nil
}
