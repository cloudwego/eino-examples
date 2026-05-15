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
	"testing"

	"github.com/cloudwego/eino/schema"
)

func TestNormalizeForSessionAgenticMessage(t *testing.T) {
	msg := &schema.AgenticMessage{
		Role: schema.AgenticRoleTypeAssistant,
		ContentBlocks: []*schema.ContentBlock{
			{
				Type:          schema.ContentBlockTypeReasoning,
				Reasoning:     &schema.Reasoning{Text: "thinking"},
				Extra:         map[string]any{arkItemIDKey: "rs_123", arkItemStatusKey: "completed"},
				StreamingMeta: &schema.StreamingMeta{Index: 0},
			},
			{
				Type:             schema.ContentBlockTypeAssistantGenText,
				AssistantGenText: &schema.AssistantGenText{Text: "hello"},
				Extra: map[string]any{
					arkItemIDKey:    "msg_123",
					openAIItemIDKey: "item_123",
				},
				StreamingMeta: &schema.StreamingMeta{Index: 1},
			},
			{
				Type: schema.ContentBlockTypeFunctionToolCall,
				FunctionToolCall: &schema.FunctionToolCall{
					CallID:    "call_123",
					Name:      "read_file",
					Arguments: "{}",
				},
				Extra: map[string]any{
					openAIItemIDKey: "fc_123",
				},
			},
		},
		ResponseMeta: &schema.AgenticResponseMeta{TokenUsage: &schema.TokenUsage{PromptTokens: 1}},
	}

	got := NormalizeForSession[*schema.AgenticMessage](msg)

	if got == msg {
		t.Fatal("NormalizeForSession should return a copy")
	}
	if got.ResponseMeta != nil {
		t.Fatalf("ResponseMeta should not be persisted, got %#v", got.ResponseMeta)
	}
	if len(got.ContentBlocks) != 2 {
		t.Fatalf("reasoning block should be dropped, got %d blocks", len(got.ContentBlocks))
	}

	textBlock := got.ContentBlocks[0]
	if textBlock.StreamingMeta != nil {
		t.Fatalf("streaming meta should be dropped, got %#v", textBlock.StreamingMeta)
	}
	if _, ok := textBlock.Extra[arkItemIDKey]; ok {
		t.Fatalf("ark item id should be dropped: %#v", textBlock.Extra)
	}
	if _, ok := textBlock.Extra[openAIItemIDKey]; ok {
		t.Fatalf("openai item id should be dropped: %#v", textBlock.Extra)
	}
	if textBlock.Extra[arkItemStatusKey] != itemStatusCompleted {
		t.Fatalf("ark status = %v, want %q", textBlock.Extra[arkItemStatusKey], itemStatusCompleted)
	}
	if textBlock.Extra[openAIItemStatusKey] != itemStatusCompleted {
		t.Fatalf("openai status = %v, want %q", textBlock.Extra[openAIItemStatusKey], itemStatusCompleted)
	}

	callBlock := got.ContentBlocks[1]
	if callBlock.FunctionToolCall.CallID != "call_123" {
		t.Fatalf("tool call id = %q", callBlock.FunctionToolCall.CallID)
	}
	if _, ok := callBlock.Extra[openAIItemIDKey]; ok {
		t.Fatalf("function tool call item id should be dropped: %#v", callBlock.Extra)
	}
	if callBlock.Extra[arkItemStatusKey] != itemStatusCompleted {
		t.Fatalf("function tool call status = %v", callBlock.Extra[arkItemStatusKey])
	}
}

func TestNewAssistantAgenticSetsCompletedStatus(t *testing.T) {
	msg := NewAssistant[*schema.AgenticMessage]("hello", []ToolCall{{
		ID:   "call_123",
		Name: "read_file",
		Args: "{}",
	}})

	if len(msg.ContentBlocks) != 2 {
		t.Fatalf("got %d blocks, want 2", len(msg.ContentBlocks))
	}
	for _, block := range msg.ContentBlocks {
		if block.Extra[arkItemStatusKey] != itemStatusCompleted {
			t.Fatalf("ark status missing for %s: %#v", block.Type, block.Extra)
		}
		if block.Extra[openAIItemStatusKey] != itemStatusCompleted {
			t.Fatalf("openai status missing for %s: %#v", block.Type, block.Extra)
		}
	}
}
