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

package jsonfix

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func echoEndpoint(_ context.Context, in *compose.ToolInput) (*compose.ToolOutput, error) {
	return &compose.ToolOutput{Result: in.Arguments}, nil
}

func TestInvokableMiddleware_RepairsJSON(t *testing.T) {
	mw := Invokable
	chained := mw(echoEndpoint)

	input := &compose.ToolInput{
		Name:      "test_tool",
		Arguments: "noise <|FunctionCallBegin|>{\"a\":1}<|FunctionCallEnd|> more",
		CallID:    "id1",
	}

	out, err := chained(context.Background(), input)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	var m map[string]int
	if e := json.Unmarshal([]byte(out.Result), &m); e != nil || m["a"] != 1 {
		t.Fatalf("repair failed: %v %v", out.Result, e)
	}
}

func TestStreamableMiddleware_RepairsJSON(t *testing.T) {
	mw := Streamable
	var captured string
	next := func(_ context.Context, in *compose.ToolInput) (*compose.StreamToolOutput, error) {
		captured = in.Arguments
		return &compose.StreamToolOutput{Result: schema.StreamReaderFromArray([]string{"ok"})}, nil
	}
	chained := mw(next)
	_, err := chained(context.Background(), &compose.ToolInput{Name: "t", Arguments: "{\"text\": \"He said \"hello\" to me\"}"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	var m map[string]string
	if e := json.Unmarshal([]byte(captured), &m); e != nil || m["text"] != "He said \"hello\" to me" {
		t.Fatalf("repair failed: %v %v", captured, e)
	}
}

func TestInvokableMiddleware_NoChangeForValidJSON(t *testing.T) {
	mw := Invokable
	chained := mw(echoEndpoint)

	original := "{\"a\":1}"
	out, err := chained(context.Background(), &compose.ToolInput{Name: "t", Arguments: original})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if out.Result != original {
		t.Fatalf("should not change valid json: got %s", out.Result)
	}
}

func TestInvokableMiddleware_MissingBracesAndUnicode(t *testing.T) {
	mw := Invokable
	chained := mw(echoEndpoint)

	inputs := []string{
		"{\"key\":\"value\"", // missing tail
		"\"key\":\"value\"}", // missing head
		"{\"emoji\": \"😀😎\"}",
		"{\"text\": \"line1\nline2\tTabbed\"}",
	}
	for _, in := range inputs {
		out, err := chained(context.Background(), &compose.ToolInput{Name: "t", Arguments: in})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		var any map[string]string
		if e := json.Unmarshal([]byte(out.Result), &any); e != nil {
			t.Fatalf("unmarshal failed: %v for %v", e, out.Result)
		}
	}
}

type spyInvokable struct{}

func (s *spyInvokable) Info(_ context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{Name: "spy", Desc: ""}, nil
}
func (s *spyInvokable) InvokableRun(_ context.Context, argumentsInJSON string, _ ...tool.Option) (string, error) {
	return argumentsInJSON, nil
}

func TestToolsNodeMiddleware_RepairsInvokable(t *testing.T) {
	tn, err := compose.NewToolNode(context.Background(), &compose.ToolsNodeConfig{
		Tools:               []tool.BaseTool{&spyInvokable{}},
		ToolCallMiddlewares: []compose.ToolMiddleware{Middleware()},
	})
	if err != nil {
		t.Fatalf("new tools node err: %v", err)
	}
	msg := schema.AssistantMessage("", nil)
	msg.ToolCalls = []schema.ToolCall{{
		ID:       "1",
		Function: schema.FunctionCall{Name: "spy", Arguments: "garbage {\"x\": \"a \"quote\" b\"} tail"},
	}}
	outs, err := tn.Invoke(context.Background(), msg)
	if err != nil {
		t.Fatalf("invoke err: %v", err)
	}
	if len(outs) != 1 {
		t.Fatalf("unexpected outs: %d", len(outs))
	}
	var m map[string]string
	if e := json.Unmarshal([]byte(outs[0].Content), &m); e != nil || m["x"] != "a \"quote\" b" {
		t.Fatalf("repair failed: %v %v", outs[0].Content, e)
	}
}

func TestInvokableMiddleware_UnescapedBackslashesAndSingleQuotesAndTrailingComma(t *testing.T) {
	mw := Invokable
	chained := mw(echoEndpoint)

	cases := []string{
		// unescaped backslashes in Windows path
		"{\"path\": \"C:\\Users\\name\\file.txt\"}",
		// single-quoted JSON
		"{'a': 'b'}",
		// trailing comma
		"{\"a\": 1, \"b\": 2, }",
	}

	for _, in := range cases {
		out, err := chained(context.Background(), &compose.ToolInput{Name: "t", Arguments: in})
		if err != nil {
			t.Fatalf("err: %v", err)
		}
		var any map[string]any
		if e := json.Unmarshal([]byte(out.Result), &any); e != nil {
			t.Fatalf("unmarshal failed: %v for %v", e, out.Result)
		}
	}
}

func TestInvokableMiddleware_RawQuotesInsideStringValue(t *testing.T) {
	mw := Invokable
	chained := mw(echoEndpoint)

	// invalid JSON: raw quotes inside value
	in := "{\"a\":\"b\"1\"\"}"
	out, err := chained(context.Background(), &compose.ToolInput{Name: "t", Arguments: in})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	var m map[string]string
	if e := json.Unmarshal([]byte(out.Result), &m); e != nil {
		t.Fatalf("unmarshal failed: %v for %v", e, out.Result)
	}
	v := m["a"]
	if v == "" || v[0] != 'b' {
		t.Fatalf("unexpected repaired value: %q", v)
	}
}
