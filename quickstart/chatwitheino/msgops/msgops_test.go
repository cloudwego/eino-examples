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

func TestKindFromEnvDefaultsToAgentic(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  Kind
	}{
		{name: "empty", value: "", want: KindAgentic},
		{name: "agentic", value: "agentic", want: KindAgentic},
		{name: "agentic message", value: "agenticmessage", want: KindAgentic},
		{name: "legacy message", value: "message", want: KindMessage},
		{name: "unknown", value: "unknown", want: KindAgentic},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("MESSAGE_KIND", tt.value)

			if got := KindFromEnv(); got != tt.want {
				t.Fatalf("KindFromEnv() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestNormalizeForSessionAgenticMessage(t *testing.T) {
	msg := &schema.AgenticMessage{
		Role: schema.AgenticRoleTypeAssistant,
		ContentBlocks: []*schema.ContentBlock{
			{
				Type:          schema.ContentBlockTypeReasoning,
				Reasoning:     &schema.Reasoning{Text: "thinking", Signature: "encrypted"},
				Extra:         map[string]any{arkItemIDKey: "rs_123"},
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
	if len(got.ContentBlocks) != 3 {
		t.Fatalf("got %d blocks, want 3", len(got.ContentBlocks))
	}

	reasoningBlock := got.ContentBlocks[0]
	if reasoningBlock.StreamingMeta != nil {
		t.Fatalf("reasoning streaming meta should be dropped, got %#v", reasoningBlock.StreamingMeta)
	}
	if reasoningBlock.Reasoning == nil || reasoningBlock.Reasoning.Signature != "encrypted" {
		t.Fatalf("reasoning signature not preserved: %#v", reasoningBlock.Reasoning)
	}
	if reasoningBlock.Extra[arkItemIDKey] != "rs_123" {
		t.Fatalf("reasoning item id should be preserved: %#v", reasoningBlock.Extra)
	}
	if reasoningBlock.Extra[arkItemStatusKey] != itemStatusCompleted {
		t.Fatalf("reasoning ark status = %v, want %q", reasoningBlock.Extra[arkItemStatusKey], itemStatusCompleted)
	}
	if reasoningBlock.Extra[openAIItemStatusKey] != itemStatusCompleted {
		t.Fatalf("reasoning openai status = %v, want %q", reasoningBlock.Extra[openAIItemStatusKey], itemStatusCompleted)
	}

	textBlock := got.ContentBlocks[1]
	if textBlock.StreamingMeta != nil {
		t.Fatalf("streaming meta should be dropped, got %#v", textBlock.StreamingMeta)
	}
	if textBlock.Extra[arkItemIDKey] != "msg_123" {
		t.Fatalf("ark item id should be preserved: %#v", textBlock.Extra)
	}
	if textBlock.Extra[openAIItemIDKey] != "item_123" {
		t.Fatalf("openai item id should be preserved: %#v", textBlock.Extra)
	}
	if textBlock.Extra[arkItemStatusKey] != itemStatusCompleted {
		t.Fatalf("ark status = %v, want %q", textBlock.Extra[arkItemStatusKey], itemStatusCompleted)
	}
	if textBlock.Extra[openAIItemStatusKey] != itemStatusCompleted {
		t.Fatalf("openai status = %v, want %q", textBlock.Extra[openAIItemStatusKey], itemStatusCompleted)
	}

	callBlock := got.ContentBlocks[2]
	if callBlock.FunctionToolCall.CallID != "call_123" {
		t.Fatalf("tool call id = %q", callBlock.FunctionToolCall.CallID)
	}
	if callBlock.Extra[openAIItemIDKey] != "fc_123" {
		t.Fatalf("function tool call item id should be preserved: %#v", callBlock.Extra)
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

func TestConcatChunksAgenticPreservesReasoning(t *testing.T) {
	chunks := []*schema.AgenticMessage{
		{
			Role: schema.AgenticRoleTypeAssistant,
			ContentBlocks: []*schema.ContentBlock{
				{
					Type:          schema.ContentBlockTypeReasoning,
					Reasoning:     &schema.Reasoning{Text: "think", Signature: "sig"},
					StreamingMeta: &schema.StreamingMeta{Index: 0},
				},
			},
		},
		{
			Role: schema.AgenticRoleTypeAssistant,
			ContentBlocks: []*schema.ContentBlock{
				{
					Type:             schema.ContentBlockTypeAssistantGenText,
					AssistantGenText: &schema.AssistantGenText{Text: "answer"},
					StreamingMeta:    &schema.StreamingMeta{Index: 1},
				},
			},
		},
	}

	got, err := ConcatChunks[*schema.AgenticMessage](chunks)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.ContentBlocks) != 2 {
		t.Fatalf("got %d blocks, want 2", len(got.ContentBlocks))
	}
	if got.ContentBlocks[0].Reasoning == nil || got.ContentBlocks[0].Reasoning.Signature != "sig" {
		t.Fatalf("reasoning not preserved: %#v", got.ContentBlocks[0])
	}
	if got.ContentBlocks[1].AssistantGenText == nil || got.ContentBlocks[1].AssistantGenText.Text != "answer" {
		t.Fatalf("assistant text not preserved: %#v", got.ContentBlocks[1])
	}
}
