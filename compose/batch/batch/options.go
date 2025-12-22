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

import "github.com/cloudwego/eino/compose"

// options holds runtime configuration for a batch invocation.
type options struct {
	// innerOptions are compose.Option values passed to each inner task invocation.
	// These are request-time options (vs compile-time options in NodeConfig).
	innerOptions []compose.Option
}

// Option is a function that configures batch invocation options.
type Option func(*options)

// WithInnerOptions passes compose.Option values to each inner task invocation.
// Use this for request-time options like:
//   - compose.WithCallbacks: Add callbacks for progress tracking
//   - compose.WithMaxRunSteps: Limit execution steps per task
//
// Example:
//
//	batchNode.Invoke(ctx, inputs,
//	    batch.WithInnerOptions(
//	        compose.WithCallbacks(progressHandler),
//	    ),
//	)
func WithInnerOptions(opts ...compose.Option) Option {
	return func(o *options) {
		o.innerOptions = append(o.innerOptions, opts...)
	}
}

// applyBatchOptions creates an options struct from the given Option functions.
func applyBatchOptions(opts ...Option) *options {
	o := &options{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}
