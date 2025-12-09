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

// Package jsonfix provides a ToolMiddleware for Eino's ToolsNode that
// repairs malformed JSON arguments produced by LLMs before tool execution.
//
// Usage:
//
//	conf := &compose.ToolsNodeConfig{
//	  Tools: []tool.BaseTool{yourTool},
//	  ToolCallMiddlewares: []compose.ToolMiddleware{jsonfix.Middleware()},
//	}
//
// Behavior:
//   - Fast-path returns original arguments when already valid JSON.
//   - Strips common LLM artifacts and isolates the first {...} region.
//   - Applies robust fix using jsonrepair only when input is invalid.
//   - Safe for both invokable and streamable tools.
package jsonfix

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/cloudwego/eino/compose"
	"github.com/kaptinlin/jsonrepair"
)

// FixJSON is a helper endpoint that returns repaired JSON for diagnostics
// or standalone use. ToolsNode should prefer Invokable/Streamable middleware.
func FixJSON(ctx context.Context, in *compose.ToolInput) (*compose.ToolOutput, error) {
	fixed := repair(in.Arguments)
	return &compose.ToolOutput{Result: fixed}, nil
}

// Invokable wraps a non-stream tool endpoint to sanitize JSON arguments.
// Register via ToolCallMiddlewares to apply automatically to invokable tools.
func Invokable(next compose.InvokableToolEndpoint) compose.InvokableToolEndpoint {
	return func(ctx context.Context, in *compose.ToolInput) (*compose.ToolOutput, error) {
		in.Arguments = repair(in.Arguments)
		return next(ctx, in)
	}
}

// Streamable wraps a stream tool endpoint to sanitize JSON arguments.
// Register via ToolCallMiddlewares to apply automatically to streamable tools.
func Streamable(next compose.StreamableToolEndpoint) compose.StreamableToolEndpoint {
	return func(ctx context.Context, in *compose.ToolInput) (*compose.StreamToolOutput, error) {
		in.Arguments = repair(in.Arguments)
		return next(ctx, in)
	}
}

// Middleware bundles both invokable and streamable wrappers for convenience.
func Middleware() compose.ToolMiddleware {
	return compose.ToolMiddleware{Invokable: Invokable, Streamable: Streamable}
}

// repair attempts minimal work first (validity check, region isolation) and
// only uses jsonrepair when necessary. It trims common LLM artifacts.
func repair(input string) string {
	s := strings.TrimSpace(input)
	// Fast-path: valid JSON as-is
	if strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") && json.Valid([]byte(s)) {
		return s
	}

	// Isolate JSON object region if present; strip noise if object-only is valid
	i := strings.IndexByte(s, '{')
	j := strings.LastIndexByte(s, '}')
	if i >= 0 && j >= i {
		sub := s[i : j+1]
		if json.Valid([]byte(sub)) {
			return sub
		}
		s = sub
	}

	// Remove common LLM artifacts
	s = strings.TrimPrefix(s, "<|FunctionCallBegin|>")
	s = strings.TrimSuffix(s, "<|FunctionCallEnd|>")
	s = strings.TrimPrefix(s, "<think>")

	// Attempt robust repair only when invalid
	if json.Valid([]byte(s)) {
		return s
	}
	// Heuristic: add missing leading/trailing brace if one side exists
	if !strings.HasPrefix(s, "{") && strings.HasSuffix(s, "}") {
		s = "{" + s
	} else if strings.HasPrefix(s, "{") && !strings.HasSuffix(s, "}") {
		s = s + "}"
	}
	out, err := jsonrepair.JSONRepair(s)
	if err != nil {
		return s
	}
	return out
}
