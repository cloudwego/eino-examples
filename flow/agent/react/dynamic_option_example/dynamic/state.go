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

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// State holds the iteration state that persists across ReAct loop iterations.
// It is stored in the parent graph's local state and accessed via compose.ProcessState.
type State struct {
	// Iteration is the current iteration number (0-indexed).
	// It is incremented after each ChatModel.Generate() call.
	Iteration int

	// LastToolCalls stores the tool calls from the previous iteration.
	// This can be used to make decisions based on what tools were called.
	LastToolCalls []*schema.ToolCall

	// CustomData allows storing arbitrary data for custom decision logic.
	CustomData map[string]any
}

func init() {
	// Register the State type for serialization support in compose graphs.
	schema.RegisterName[State]("DynamicOptionState")
}

// NewState creates a new State with default values.
func NewState() *State {
	return &State{
		Iteration:  0,
		CustomData: make(map[string]any),
	}
}

// OptionFunc is the function signature for dynamic option generation.
// It is called before each ChatModel.Generate() call and returns options
// to be merged with any static options passed to the agent.
//
// Parameters:
//   - ctx: The context from the current request
//   - input: The input messages being sent to the ChatModel
//   - state: The current iteration state (can be modified)
//
// Returns:
//   - A slice of model.Option to be applied to this ChatModel call
type OptionFunc func(ctx context.Context, input []*schema.Message, state *State) []model.Option
