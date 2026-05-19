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
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

// IsNil reports whether a generic message value is nil.
func IsNil[M adk.MessageType](msg M) bool {
	var zero M
	return any(msg) == any(zero)
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

// ConcatChunks merges streaming message chunks using the official Eino schema
// helpers. For AgenticMessage this preserves reasoning blocks and reasoning
// signatures emitted by the model convertors.
func ConcatChunks[M adk.MessageType](chunks []M) (M, error) {
	var zero M
	if len(chunks) == 0 {
		return zero, fmt.Errorf("no chunks to concat")
	}
	if len(chunks) == 1 {
		return chunks[0], nil
	}

	if KindOf[M]() == KindAgentic {
		msgs := make([]*schema.AgenticMessage, 0, len(chunks))
		for _, chunk := range chunks {
			msg, ok := any(chunk).(*schema.AgenticMessage)
			if !ok || msg == nil {
				return zero, fmt.Errorf("unexpected agentic chunk type %T", chunk)
			}
			msgs = append(msgs, msg)
		}
		merged, err := schema.ConcatAgenticMessages(msgs)
		if err != nil {
			return zero, err
		}
		return any(merged).(M), nil
	}

	msgs := make([]*schema.Message, 0, len(chunks))
	for _, chunk := range chunks {
		msg, ok := any(chunk).(*schema.Message)
		if !ok || msg == nil {
			return zero, fmt.Errorf("unexpected message chunk type %T", chunk)
		}
		msgs = append(msgs, msg)
	}
	merged, err := schema.ConcatMessages(msgs)
	if err != nil {
		return zero, err
	}
	return any(merged).(M), nil
}

// UnmarshalMessage unmarshals one JSONL message line into M.
func UnmarshalMessage[M adk.MessageType](data []byte) (M, error) {
	if KindOf[M]() == KindAgentic {
		var msg schema.AgenticMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			var zero M
			return zero, err
		}
		return NormalizeForSession(any(&msg).(M)), nil
	}
	var msg schema.Message
	err := json.Unmarshal(data, &msg)
	return any(&msg).(M), err
}
