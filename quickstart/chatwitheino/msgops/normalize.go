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
	"maps"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

const (
	arkItemIDKey        = "ark-item-id"
	arkItemStatusKey    = "ark-item-status"
	openAIItemIDKey     = "openai-item-id"
	openAIItemStatusKey = "openai-item-status"
	itemStatusCompleted = "completed"
)

// NormalizeForSession returns a provider-safe message for chatwitheino's JSONL
// session store. Agentic model implementations attach Responses API item IDs to
// content blocks, and those IDs are required when replaying provider-native
// output items such as assistant messages, reasoning blocks, and tool calls.
// The examples persist those IDs while dropping streaming-only metadata.
func NormalizeForSession[M adk.MessageType](msg M) M {
	if KindOf[M]() != KindAgentic {
		return msg
	}
	agenticMsg, ok := any(msg).(*schema.AgenticMessage)
	if !ok || agenticMsg == nil {
		return msg
	}
	return any(normalizeAgenticMessage(agenticMsg)).(M)
}

// NormalizeMessagesForModelInput prepares stored messages before passing them
// back to Runner/ChatModel. Today this matches NormalizeForSession; keeping the
// boundary explicit makes provider-specific input rules easy to evolve without
// changing the session file format.
func NormalizeMessagesForModelInput[M adk.MessageType](messages []M) []M {
	out := make([]M, 0, len(messages))
	for _, msg := range messages {
		out = append(out, NormalizeForSession(msg))
	}
	return out
}

func normalizeAgenticMessage(msg *schema.AgenticMessage) *schema.AgenticMessage {
	if msg == nil {
		return nil
	}

	out := &schema.AgenticMessage{
		Role:          msg.Role,
		ContentBlocks: make([]*schema.ContentBlock, 0, len(msg.ContentBlocks)),
		Extra:         cloneMap(msg.Extra),
	}

	for _, block := range msg.ContentBlocks {
		normalized := normalizeAgenticContentBlock(block)
		if normalized != nil {
			out.ContentBlocks = append(out.ContentBlocks, normalized)
		}
	}

	return out
}

func normalizeAgenticContentBlock(block *schema.ContentBlock) *schema.ContentBlock {
	if block == nil {
		return nil
	}

	out := *block
	out.Extra = normalizeAgenticBlockExtra(block.Type, block.Extra)
	out.StreamingMeta = nil
	return &out
}

func normalizeAgenticBlockExtra(blockType schema.ContentBlockType, extra map[string]any) map[string]any {
	out := make(map[string]any, len(extra)+2)
	maps.Copy(out, extra)

	if needsCompletedStatus(blockType) {
		out[arkItemStatusKey] = itemStatusCompleted
		out[openAIItemStatusKey] = itemStatusCompleted
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func needsCompletedStatus(blockType schema.ContentBlockType) bool {
	switch blockType {
	case schema.ContentBlockTypeReasoning,
		schema.ContentBlockTypeAssistantGenText,
		schema.ContentBlockTypeFunctionToolCall:
		return true
	default:
		return false
	}
}

func cloneMap(in map[string]any) map[string]any {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]any, len(in))
	maps.Copy(out, in)
	return out
}
