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

package abtest

import (
	"context"
	"errors"

	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

type ModelRouter func(ctx context.Context, input []*schema.Message, opts ...model.Option) (string, model.BaseChatModel, error)

// ABRouterChatModel is a dynamic router over chat models that implements ToolCallingChatModel.
//
// Behavior:
//   - Routing: delegates the choice to a user-provided ModelRouter which returns (modelName, BaseChatModel).
//   - RunInfo naming: uses the returned modelName when calling callbacks.EnsureRunInfo so callbacks can log the chosen model.
//   - Tools: stores tool infos via WithTools and applies them lazily if the chosen model supports ToolCallingChatModel.
//   - Callbacks: if the chosen model exposes components.Checker and IsCallbacksEnabled()==true, delegates directly;
//     otherwise injects OnStart/OnEnd/OnError around Generate/Stream.
//   - IsCallbacksEnabled: returns true to indicate this wrapper already coordinates callback triggering.
//
// Typical usage:
//
//	router := NewABRouterChatModel(func(ctx, msgs, opts...) (string, model.BaseChatModel, error) {
//	    return "openai", openaiModel, nil
//	})
//	router = router.WithTools(toolInfos) // optional
//	msg, _ := router.Generate(ctx, input)
type ABRouterChatModel struct {
	router ModelRouter
	tools  []*schema.ToolInfo
}

func NewABRouterChatModel(router ModelRouter) *ABRouterChatModel {
	return &ABRouterChatModel{router: router}
}

func (a *ABRouterChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	return &ABRouterChatModel{router: a.router, tools: tools}, nil
}

func (a *ABRouterChatModel) pickModel(ctx context.Context, input []*schema.Message, opts ...model.Option) (string, model.BaseChatModel, error) {
	if a.router == nil {
		return "", nil, errors.New("no router")
	}
	name, base, err := a.router(ctx, input, opts...)
	if err != nil || base == nil {
		return "", nil, err
	}
	if tcm, ok := base.(model.ToolCallingChatModel); ok && len(a.tools) > 0 {
		nTcm, wErr := tcm.WithTools(a.tools)
		if wErr != nil {
			return "", nil, wErr
		}
		base = nTcm
	}
	return name, base, nil
}

func (a *ABRouterChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	name, base, err := a.pickModel(ctx, input, opts...)
	if err != nil || base == nil {
		if err == nil {
			err = errors.New("router returned nil model")
		}
		callbacks.OnError(ctx, err)
		return nil, err
	}
	ctx = callbacks.ReuseHandlers(ctx, &callbacks.RunInfo{Name: name, Component: components.ComponentOfChatModel})
	if ch, ok := base.(components.Checker); ok && ch.IsCallbacksEnabled() {
		return base.Generate(ctx, input, opts...)
	}
	nCtx := callbacks.OnStart(ctx, &model.CallbackInput{Messages: input})
	out, err := base.Generate(nCtx, input, opts...)
	if err != nil {
		callbacks.OnError(nCtx, err)
		return nil, err
	}
	callbacks.OnEnd(nCtx, &model.CallbackOutput{Message: out})
	return out, nil
}

func (a *ABRouterChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	name, base, err := a.pickModel(ctx, input, opts...)
	if err != nil || base == nil {
		if err == nil {
			err = errors.New("router returned nil model")
		}
		callbacks.OnError(ctx, err)
		return nil, err
	}
	ctx = callbacks.ReuseHandlers(ctx, &callbacks.RunInfo{Name: name, Component: components.ComponentOfChatModel})
	if ch, ok := base.(components.Checker); ok && ch.IsCallbacksEnabled() {
		return base.Stream(ctx, input, opts...)
	}
	nCtx := callbacks.OnStart(ctx, &model.CallbackInput{Messages: input})
	sr, err := base.Stream(nCtx, input, opts...)
	if err != nil {
		callbacks.OnError(nCtx, err)
		return nil, err
	}
	out := schema.StreamReaderWithConvert(sr, func(m *schema.Message) (*model.CallbackOutput, error) {
		return &model.CallbackOutput{Message: m}, nil
	})
	_, out = callbacks.OnEndWithStreamOutput(nCtx, out)
	back := schema.StreamReaderWithConvert(out, func(o *model.CallbackOutput) (*schema.Message, error) {
		return o.Message, nil
	})
	return back, nil
}

func (a *ABRouterChatModel) IsCallbacksEnabled() bool { return true }
