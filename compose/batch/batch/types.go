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

// Package batch provides a BatchNode implementation for processing multiple inputs
// through a Graph or Workflow with configurable concurrency and interrupt/resume support.
//
// Key features:
//   - Generic batch processing: Accept []I, return []O
//   - Configurable concurrency: Sequential (0) or concurrent with limit (>0)
//   - Interrupt handling: Collects interrupts from sub-tasks using CompositeInterrupt
//   - Resume support: Restores state and only re-runs interrupted tasks
//   - Callbacks: Implements Typer and Checker interfaces for callback support
package batch

import (
	"context"

	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func init() {
	// Register NodeInterruptState for serialization during checkpoint save/restore.
	// This is required for the interrupt state to be properly persisted.
	schema.RegisterName[*NodeInterruptState]("batch.NodeInterruptState")
}

// ComponentOfBatchNode is the component type identifier for callbacks.
// Used by callbacks.EnsureRunInfo to identify this component in the callback chain.
const ComponentOfBatchNode components.Component = "Batch"

// AddressSegmentBatchProcess is the address segment type for batch processing.
// Used by compose.AppendAddressSegment to create unique addresses for each sub-task,
// enabling proper interrupt ID generation (e.g., "batch_process:0", "batch_process:1").
const AddressSegmentBatchProcess compose.AddressSegmentType = "batch_process"

// Compilable represents a Graph or Workflow that can be compiled into a Runnable.
// Both compose.Graph and compose.Workflow implement this interface.
type Compilable[I, O any] interface {
	Compile(ctx context.Context, opts ...compose.GraphCompileOption) (compose.Runnable[I, O], error)
}

// NodeConfig contains configuration for creating a BatchNode.
type NodeConfig[I, O any] struct {
	// Name is the node name used for callbacks and logging. Defaults to "Node" if empty.
	Name string

	// InnerTask is the Graph or Workflow to run for each input item.
	// Must implement Compilable[I, O] interface.
	InnerTask Compilable[I, O]

	// MaxConcurrency controls parallel execution:
	//   - 0: Sequential processing (one task at a time)
	//   - >0: Concurrent processing with this many parallel tasks
	//         First task runs on main goroutine, rest run in goroutines
	MaxConcurrency int

	// InnerCompileOptions are passed to InnerTask.Compile() for each invocation.
	// Use this for compile-time options like WithGraphName.
	InnerCompileOptions []compose.GraphCompileOption
}

// NodeInterruptState stores the batch node's state when an interrupt occurs.
// This state is persisted via CompositeInterrupt and restored on resume.
type NodeInterruptState struct {
	// OriginalInputs stores all input items (as []any for serialization).
	// Required because inputs are not passed during resume invocation.
	OriginalInputs []any

	// CompletedResults maps index -> result for tasks that completed before interrupt.
	// These results are restored directly without re-running the tasks.
	CompletedResults map[int]any

	// InterruptedIndices lists which task indices were interrupted.
	// Only these tasks will be re-run on resume.
	InterruptedIndices []int

	// TotalCount is the total number of input items.
	// Used to allocate the correct output slice size on resume.
	TotalCount int
}

// CallbackInput is passed to callbacks.OnStart when batch processing begins.
type CallbackInput[I any] struct {
	Inputs         []I // All input items to be processed
	MaxConcurrency int // Configured concurrency limit
}

// CallbackOutput is passed to callbacks.OnEnd when batch processing completes.
type CallbackOutput[O any] struct {
	Outputs []O // All output results
}
