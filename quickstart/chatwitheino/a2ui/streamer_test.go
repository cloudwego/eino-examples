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

package a2ui

import (
	"bytes"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

func TestStreamToWriterPersistsAgenticReasoning(t *testing.T) {
	iter, gen := adk.NewAsyncIteratorPair[*adk.TypedAgentEvent[*schema.AgenticMessage]]()
	go func() {
		defer gen.Close()
		gen.Send(&adk.TypedAgentEvent[*schema.AgenticMessage]{
			Output: &adk.TypedAgentOutput[*schema.AgenticMessage]{
				MessageOutput: &adk.TypedMessageVariant[*schema.AgenticMessage]{
					IsStreaming: true,
					AgenticRole: schema.AgenticRoleTypeAssistant,
					MessageStream: schema.StreamReaderFromArray([]*schema.AgenticMessage{
						{
							Role: schema.AgenticRoleTypeAssistant,
							ContentBlocks: []*schema.ContentBlock{
								{
									Type:          schema.ContentBlockTypeReasoning,
									Reasoning:     &schema.Reasoning{Text: "thinking", Signature: "encrypted"},
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
					}),
				},
			},
		})
		gen.Send(&adk.TypedAgentEvent[*schema.AgenticMessage]{Action: adk.NewExitAction()})
	}()

	var buf bytes.Buffer
	lastContent, intermediates, interruptID, _, err := StreamToWriter(&buf, "s", nil, iter)
	if err != nil {
		t.Fatal(err)
	}
	if interruptID != "" {
		t.Fatalf("unexpected interrupt: %s", interruptID)
	}
	if lastContent != "answer" {
		t.Fatalf("lastContent = %q, want answer", lastContent)
	}
	if len(intermediates) != 1 {
		t.Fatalf("got %d intermediates, want 1", len(intermediates))
	}

	got := intermediates[0]
	if len(got.ContentBlocks) != 2 {
		t.Fatalf("got %d content blocks, want 2", len(got.ContentBlocks))
	}
	if got.ContentBlocks[0].Reasoning == nil || got.ContentBlocks[0].Reasoning.Signature != "encrypted" {
		t.Fatalf("reasoning not preserved: %#v", got.ContentBlocks[0])
	}
	if got.ContentBlocks[1].AssistantGenText == nil || got.ContentBlocks[1].AssistantGenText.Text != "answer" {
		t.Fatalf("assistant text not preserved: %#v", got.ContentBlocks[1])
	}
}
