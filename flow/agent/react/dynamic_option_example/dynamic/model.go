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

package dynamic

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// ChatModel wraps a BaseChatModel and enables dynamic option modification.
// Before each Generate() or Stream() call, it:
// 1. Reads the current iteration state from the parent graph via compose.ProcessState
// 2. Calls GetOptionFunc to get dynamic options based on the current state
// 3. Increments the iteration counter
// 4. Merges dynamic options with any static options and calls the inner model
type ChatModel struct {
	// Model is the underlying ChatModel to wrap
	Model model.BaseChatModel

	// GetOptionFunc is called before each Generate()/Stream() call to get dynamic options
	GetOptionFunc OptionFunc
}

// Generate implements model.BaseChatModel.
// It reads state, calls GetOptionFunc, increments iteration, and delegates to the inner model.
func (d *ChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	var dynamicOpts []model.Option

	// Access the parent graph's state via compose.ProcessState.
	// This is the key mechanism that allows state to persist across ReAct loop iterations.
	// We are accessing parent graph's state here. Require eino version v0.7.11+
	err := compose.ProcessState[*State](ctx, func(_ context.Context, state *State) error {
		// Small delay to ensure log ordering (optional, for demo purposes)
		time.Sleep(100 * time.Millisecond)
		fmt.Printf("\n==================== Iteration %d ====================\n", state.Iteration)

		// Get dynamic options based on current state
		dynamicOpts = d.GetOptionFunc(ctx, input, state)

		// Increment iteration for next call
		state.Iteration++
		return nil
	})
	if err != nil {
		// If state access fails (e.g., not running in a graph), use no dynamic options
		dynamicOpts = nil
	}

	// Merge dynamic options with static options (dynamic options take precedence)
	mergedOpts := append(dynamicOpts, opts...)
	resp, err := d.Model.Generate(ctx, input, mergedOpts...)

	// Store tool calls in state for potential use in next iteration's decision
	if err == nil && resp != nil && len(resp.ToolCalls) > 0 {
		_ = compose.ProcessState[*State](ctx, func(_ context.Context, state *State) error {
			toolCalls := make([]*schema.ToolCall, len(resp.ToolCalls))
			for i := range resp.ToolCalls {
				toolCalls[i] = &resp.ToolCalls[i]
			}
			state.LastToolCalls = toolCalls
			return nil
		})
	}

	return resp, err
}

// Stream implements model.BaseChatModel.
// Same logic as Generate but returns a stream reader.
func (d *ChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	var dynamicOpts []model.Option

	// We are accessing parent graph's state here. Require eino version v0.7.11+
	err := compose.ProcessState[*State](ctx, func(_ context.Context, state *State) error {
		time.Sleep(100 * time.Millisecond)
		fmt.Printf("\n==================== Iteration %d ====================\n", state.Iteration)
		dynamicOpts = d.GetOptionFunc(ctx, input, state)
		state.Iteration++
		return nil
	})
	if err != nil {
		dynamicOpts = nil
	}

	mergedOpts := append(dynamicOpts, opts...)
	return d.Model.Stream(ctx, input, mergedOpts...)
}

// WithTools implements model.ToolCallingChatModel.
// It creates a new ChatModel wrapping the result of the inner model's WithTools.
func (d *ChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	tcm, ok := d.Model.(model.ToolCallingChatModel)
	if !ok {
		return nil, nil
	}
	newModel, err := tcm.WithTools(tools)
	if err != nil {
		return nil, err
	}
	return &ChatModel{
		Model:         newModel,
		GetOptionFunc: d.GetOptionFunc,
	}, nil
}

// IsCallbacksEnabled implements components.Checker.
// Delegates to the inner model if it implements Checker.
func (d *ChatModel) IsCallbacksEnabled() bool {
	checker, ok := d.Model.(components.Checker)
	if ok {
		return checker.IsCallbacksEnabled()
	}
	return false
}

// GetType returns the type name for this component.
func (d *ChatModel) GetType() string {
	return "DynamicChatModel"
}

// Compile-time interface checks
var (
	_ model.BaseChatModel        = (*ChatModel)(nil)
	_ model.ToolCallingChatModel = (*ChatModel)(nil)
	_ components.Checker         = (*ChatModel)(nil)
)
