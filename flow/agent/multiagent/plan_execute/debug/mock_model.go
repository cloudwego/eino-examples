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

package debug

import (
	"context"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type options struct {
	debugMode   bool
	debugOutput *schema.Message
}

func WithDebugOutput(output *schema.Message) model.Option {
	return model.WrapImplSpecificOptFn(func(opts *options) {
		opts.debugMode = true
		opts.debugOutput = output
	})
}

// ChatModelDebugDecorator 给内部的 ChatModel 提供单次 Mock 输出的能力.
type ChatModelDebugDecorator struct {
	Model model.ChatModel
}

func (c *ChatModelDebugDecorator) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	option := model.GetImplSpecificOptions(&options{}, opts...)
	if option.debugMode {
		if c.IsCallbacksEnabled() {
			ctx = callbacks.OnStart(ctx, &model.CallbackInput{
				Messages: input,
			})
			callbacks.OnEnd(ctx, &model.CallbackOutput{
				Message: option.debugOutput,
			})
		}
		return option.debugOutput, nil
	}

	return c.Model.Generate(ctx, input, opts...)
}

func (c *ChatModelDebugDecorator) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	option := model.GetImplSpecificOptions(&options{}, opts...)
	if option.debugMode {
		callbackEnabled := c.IsCallbacksEnabled()
		if callbackEnabled {
			ctx = callbacks.OnStart(ctx, &model.CallbackInput{
				Messages: input,
			})
		}
		sr, sw := schema.Pipe[*schema.Message](0)
		go func() {
			defer sw.Close()
			sw.Send(option.debugOutput, nil)
		}()

		if callbackEnabled {
			outStream := schema.StreamReaderWithConvert(sr, func(m *schema.Message) (*model.CallbackOutput, error) {
				return &model.CallbackOutput{
					Message: m,
				}, nil
			})
			_, outStream = callbacks.OnEndWithStreamOutput(ctx, outStream)
			sr = schema.StreamReaderWithConvert(outStream, func(o *model.CallbackOutput) (*schema.Message, error) {
				return o.Message, nil
			})
		}

		return sr, nil
	}
	return c.Model.Stream(ctx, input, opts...)
}

func (c *ChatModelDebugDecorator) BindTools(tools []*schema.ToolInfo) error {
	return c.Model.BindTools(tools)
}

// IsCallbacksEnabled 透出内部的 ChatModel 是否已埋入了回调切面.
func (c *ChatModelDebugDecorator) IsCallbacksEnabled() bool {
	checker, ok := c.Model.(components.Checker)
	if ok {
		return checker.IsCallbacksEnabled()
	}

	return false
}
