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

// This example shows how to configure the errorremover middleware on a ToolsNode
// to catch errors during local tool invocation and return custom information.
// Run: go run ./components/tool/middlewares/errorremover/example

package errorremover

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// removeErrorHandler is an error handler function that generates a string
// describing the error that occurred during a tool call.
func removeErrorHandler(ctx context.Context, in *compose.ToolInput, err error) string {
	// Formats the error message to include the tool name and the specific error content.
	return fmt.Sprintf("Failed to call tool '%s', error message: '%s'", in.Name, err.Error())
}

// Invokable creates a middleware endpoint for non-streaming (invokable) tools.
// It intercepts the tool's execution. If the tool returns an error, it calls the
// error handler and returns its result as a successful ToolOutput,
// effectively suppressing the original error.
func Invokable(next compose.InvokableToolEndpoint) compose.InvokableToolEndpoint {
	return func(ctx context.Context, in *compose.ToolInput) (*compose.ToolOutput, error) {
		// Proceed with the next middleware or the actual tool execution.
		output, err := next(ctx, in)
		// If an error occurs during execution.
		if err != nil {
			if _, ok := compose.IsInterruptRerunError(err); ok {
				return nil, err
			}
			// Generate a custom error message using removeErrorHandler.
			result := removeErrorHandler(ctx, in, err)
			// Wrap the custom message in a successful ToolOutput and return it,
			// with the error itself set to nil.
			return &compose.ToolOutput{Result: result}, nil
		}
		// If there was no error, return the original output.
		return output, nil
	}
}

// Streamable creates a middleware endpoint for streaming tools.
// It intercepts the tool's execution. If the tool returns an error, it calls the
// error handler and returns its result as a new stream containing a single successful item.
// This effectively replaces the error with a successful stream output.
func Streamable(next compose.StreamableToolEndpoint) compose.StreamableToolEndpoint {
	return func(ctx context.Context, in *compose.ToolInput) (*compose.StreamToolOutput, error) {
		// Proceed with the next middleware or the actual tool execution.
		streamOutput, err := next(ctx, in)
		// If an error occurs during execution.
		if err != nil {
			if _, ok := compose.IsInterruptRerunError(err); ok {
				return nil, err
			}
			// Generate a custom error message using removeErrorHandler.
			result := removeErrorHandler(ctx, in, err)
			// Return the new stream as a successful output.
			return &compose.StreamToolOutput{Result: schema.StreamReaderFromArray([]string{result})}, nil

		}
		// If there was no error, return the original stream output.
		return streamOutput, nil

	}
}

// Middleware constructs and returns a compose.ToolMiddleware.
// This middleware is designed to catch errors from tool executions and replace them
// with a custom, successful output.
func Middleware() compose.ToolMiddleware {
	return compose.ToolMiddleware{Invokable: Invokable, Streamable: Streamable}
}
