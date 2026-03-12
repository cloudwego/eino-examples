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

package batch

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/compose"
)

// Node is a batch processor that runs a Graph/Workflow for each input item.
// It supports configurable concurrency, interrupt/resume, and callbacks.
//
// Type parameters:
//   - I: Input type for each item
//   - O: Output type for each item
type Node[I, O any] struct {
	name                string
	innerTask           Compilable[I, O]
	maxConcurrency      int
	innerCompileOptions []compose.GraphCompileOption
}

// NewBatchNode creates a new batch processing node.
//
// Example:
//
//	batchNode := batch.NewBatchNode(&batch.NodeConfig[Request, Response]{
//	    Name:           "MyBatchProcessor",
//	    InnerTask:      myWorkflow,
//	    MaxConcurrency: 5,
//	})
func NewBatchNode[I, O any](config *NodeConfig[I, O]) *Node[I, O] {
	name := config.Name
	if name == "" {
		name = "Node"
	}
	return &Node[I, O]{
		name:                name,
		innerTask:           config.InnerTask,
		maxConcurrency:      config.MaxConcurrency,
		innerCompileOptions: config.InnerCompileOptions,
	}
}

// GetType returns the node name for callback identification.
// Implements components.Typer interface.
func (b *Node[I, O]) GetType() string {
	return b.name
}

// IsCallbacksEnabled returns true to enable callback support.
// Implements components.Checker interface.
func (b *Node[I, O]) IsCallbacksEnabled() bool {
	return true
}

// Invoke processes all inputs through the inner task and returns results.
// It handles concurrency, errors, and interrupts according to configuration.
//
// Parameters:
//   - ctx: Context for cancellation and deadline
//   - inputs: Slice of input items to process
//   - opts: Optional batch options (e.g., WithInnerOptions)
//
// Returns:
//   - []O: Results in the same order as inputs
//   - error: First normal error, or CompositeInterrupt if any task interrupted
func (b *Node[I, O]) Invoke(ctx context.Context, inputs []I, opts ...Option) ([]O, error) {
	batchOpts := applyBatchOptions(opts...)

	// Setup callbacks for batch-level monitoring
	ctx = callbacks.EnsureRunInfo(ctx, b.name, ComponentOfBatchNode)
	ctx = callbacks.OnStart(ctx, &CallbackInput[I]{
		Inputs:         inputs,
		MaxConcurrency: b.maxConcurrency,
	})

	outputs, err := b.invoke(ctx, inputs, batchOpts)
	if err != nil {
		callbacks.OnError(ctx, err)
		return nil, err
	}

	callbacks.OnEnd(ctx, &CallbackOutput[O]{Outputs: outputs})
	return outputs, nil
}

// invoke is the internal implementation of batch processing.
func (b *Node[I, O]) invoke(ctx context.Context, inputs []I, batchOpts *options) ([]O, error) {
	// Check if this is a resume from a previous interrupt
	wasInterrupted, hasState, prevState := compose.GetInterruptState[*NodeInterruptState](ctx)

	var store *batchBridgeStore
	var indicesToProcess []int
	var effectiveInputs []I

	if wasInterrupted && hasState && prevState != nil {
		// RESUME PATH: Restore state from previous interrupt
		// Use fresh store (don't restore checkpoint data - it causes input issues)
		store = newBatchBridgeStore()
		indicesToProcess = prevState.InterruptedIndices

		// Restore original inputs from interrupt state
		// (inputs parameter is nil during resume)
		effectiveInputs = make([]I, prevState.TotalCount)
		for i, v := range prevState.OriginalInputs {
			if typedInput, ok := v.(I); ok {
				effectiveInputs[i] = typedInput
			}
		}
	} else {
		// FIRST RUN PATH: Process all inputs
		store = newBatchBridgeStore()
		effectiveInputs = inputs
		indicesToProcess = make([]int, len(inputs))
		for i := range inputs {
			indicesToProcess[i] = i
		}
	}

	// Allocate output slice
	outputs := make([]O, len(effectiveInputs))

	// Restore completed results from previous run (if resuming)
	if wasInterrupted && hasState && prevState != nil {
		for idx, result := range prevState.CompletedResults {
			if idx < len(outputs) {
				if typedResult, ok := result.(O); ok {
					outputs[idx] = typedResult
				}
			}
		}
	}

	// Compile inner task with checkpoint store
	compileOpts := append([]compose.GraphCompileOption{
		compose.WithCheckPointStore(store),
	}, b.innerCompileOptions...)

	runner, err := b.innerTask.Compile(ctx, compileOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to compile inner task: %w", err)
	}

	// Nothing to process (all completed in previous run)
	if len(indicesToProcess) == 0 {
		return outputs, nil
	}

	// Task result for collecting outputs from goroutines
	type taskResult struct {
		index  int
		output O
		err    error
	}

	resultCh := make(chan taskResult, len(indicesToProcess))
	var wg sync.WaitGroup

	// runTask executes a single inner task
	runTask := func(index int, input I) {
		defer wg.Done()

		// Create sub-context with unique address segment for this task
		// This enables proper interrupt ID generation (e.g., "batch_process:0")
		subCtx := compose.AppendAddressSegment(ctx, AddressSegmentBatchProcess, strconv.Itoa(index))

		// Combine checkpoint ID with user-provided inner options
		invokeOpts := append([]compose.Option{
			compose.WithCheckPointID(makeBatchCheckpointID(index)),
		}, batchOpts.innerOptions...)

		output, taskErr := runner.Invoke(subCtx, input, invokeOpts...)
		resultCh <- taskResult{index: index, output: output, err: taskErr}
	}

	// Execute tasks based on concurrency setting
	if b.maxConcurrency == 0 {
		// Sequential: Run one task at a time
		for _, idx := range indicesToProcess {
			wg.Add(1)
			runTask(idx, effectiveInputs[idx])
		}
	} else {
		// Concurrent: Use semaphore to limit parallelism
		sem := make(chan struct{}, b.maxConcurrency)

		for i, idx := range indicesToProcess {
			wg.Add(1)
			if i == 0 {
				// First task runs on main goroutine (optimization)
				runTask(idx, effectiveInputs[idx])
			} else {
				// Subsequent tasks run in goroutines with semaphore
				go func(index int, input I) {
					sem <- struct{}{}
					defer func() { <-sem }()
					runTask(index, input)
				}(idx, effectiveInputs[idx])
			}
		}
	}

	// Close result channel when all tasks complete
	go func() {
		wg.Wait()
		close(resultCh)
	}()

	// Collect results and categorize errors
	var normalErr error
	var interruptErrs []error
	completedResults := make(map[int]any)
	interruptedIndices := make([]int, 0)

	for result := range resultCh {
		if result.err != nil {
			if _, ok := compose.ExtractInterruptInfo(result.err); ok {
				// Interrupt error: collect for CompositeInterrupt
				interruptErrs = append(interruptErrs, result.err)
				interruptedIndices = append(interruptedIndices, result.index)
			} else if normalErr == nil {
				// Normal error: keep first one
				normalErr = fmt.Errorf("task %d failed: %w", result.index, result.err)
			}
		} else {
			// Success: store result
			outputs[result.index] = result.output
			completedResults[result.index] = result.output
		}
	}

	// Return first normal error (if any)
	if normalErr != nil {
		return nil, normalErr
	}

	// Return composite interrupt (if any tasks interrupted)
	if len(interruptErrs) > 0 {
		// Store original inputs for resume (inputs will be nil on resume call)
		originalInputs := make([]any, len(effectiveInputs))
		for i, v := range effectiveInputs {
			originalInputs[i] = v
		}
		state := &NodeInterruptState{
			OriginalInputs:     originalInputs,
			CompletedResults:   completedResults,
			InterruptedIndices: interruptedIndices,
			TotalCount:         len(effectiveInputs),
		}
		// CompositeInterrupt bundles all interrupt errors with state for resume
		return nil, compose.CompositeInterrupt(ctx, nil, state, interruptErrs...)
	}

	return outputs, nil
}
