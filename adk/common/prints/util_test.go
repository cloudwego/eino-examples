/*
 * Copyright 2025 CloudWeGo Authors
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

package prints

import (
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

func TestEventHandlesStreamingToolCallWithoutIndex(t *testing.T) {
	if os.Getenv("EINO_PRINTS_NIL_INDEX_SUBPROCESS") == "1" {
		Event(eventWithNilIndexToolCallStream())
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestEventHandlesStreamingToolCallWithoutIndex")
	cmd.Env = append(os.Environ(), "EINO_PRINTS_NIL_INDEX_SUBPROCESS=1")

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("Event should not exit when a streaming tool call has nil index, err=%v, output=%s", err, output)
	}
	if !strings.Contains(string(output), "tool name: lookup_contract") {
		t.Fatalf("Event should print the streaming tool call name, output=%s", output)
	}
	if !strings.Contains(string(output), `arguments: {"supplier":"acme"}`) {
		t.Fatalf("Event should print the streaming tool call arguments, output=%s", output)
	}
}

func TestEventPrintsMultipleStreamingToolCallsWithoutIndex(t *testing.T) {
	output := captureStdout(t, func() {
		Event(eventWithMultipleNilIndexToolCallStream())
	})

	if !strings.Contains(output, "tool name: lookup_contract") {
		t.Fatalf("Event should print the first streaming tool call name, output=%s", output)
	}
	if !strings.Contains(output, `arguments: {"supplier":"acme"}`) {
		t.Fatalf("Event should print the first streaming tool call arguments, output=%s", output)
	}
	if !strings.Contains(output, "tool name: notify_owner") {
		t.Fatalf("Event should print the second streaming tool call name, output=%s", output)
	}
	if !strings.Contains(output, `arguments: {"owner":"legal"}`) {
		t.Fatalf("Event should print the second streaming tool call arguments, output=%s", output)
	}
}

func eventWithNilIndexToolCallStream() *adk.AgentEvent {
	chunk := schema.AssistantMessage("", []schema.ToolCall{
		{
			ID: "call_1",
			Function: schema.FunctionCall{
				Name:      "lookup_contract",
				Arguments: `{"supplier":"acme"}`,
			},
		},
	})

	return adk.EventFromMessage(
		nil,
		schema.StreamReaderFromArray([]*schema.Message{chunk}),
		schema.Assistant,
		"",
	)
}

func eventWithMultipleNilIndexToolCallStream() *adk.AgentEvent {
	chunk := schema.AssistantMessage("", []schema.ToolCall{
		{
			ID: "call_1",
			Function: schema.FunctionCall{
				Name:      "lookup_contract",
				Arguments: `{"supplier":"acme"}`,
			},
		},
		{
			ID: "call_2",
			Function: schema.FunctionCall{
				Name:      "notify_owner",
				Arguments: `{"owner":"legal"}`,
			},
		},
	})

	return adk.EventFromMessage(
		nil,
		schema.StreamReaderFromArray([]*schema.Message{chunk}),
		schema.Assistant,
		"",
	)
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create stdout pipe failed: %v", err)
	}
	os.Stdout = w

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close stdout pipe writer failed: %v", err)
	}
	os.Stdout = originalStdout

	output, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stdout pipe failed: %v", err)
	}
	return string(output)
}
