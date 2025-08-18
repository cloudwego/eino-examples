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

package main

import (
	"context"
	"slices"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

func NewCachedARKChatModel(ctx context.Context, cfg *ark.ChatModelConfig) (*CachedARKChatModel, error) {
	cm, err := ark.NewChatModel(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return &CachedARKChatModel{
		ChatModel: cm,
	}, nil
}

type CachedARKChatModel struct {
	ChatModel model.ToolCallingChatModel
}

const (
	cacheOptionCtxKey = "ark-cache-option"
)

func WithCacheCtx(ctx context.Context, cache *ark.CacheOption) context.Context {
	return context.WithValue(ctx, cacheOptionCtxKey, cache)
}

func GetCacheCtx(ctx context.Context) *ark.CacheOption {
	cache, ok := ctx.Value(cacheOptionCtxKey).(*ark.CacheOption)
	if !ok {
		return nil
	}
	return cache
}

func (cm *CachedARKChatModel) Generate(ctx context.Context, in []*schema.Message,
	opts ...model.Option) (outMsg *schema.Message, err error) {

	opts_ := opts
	in_ := in

	cacheOption := GetCacheCtx(ctx)
	if cacheOption != nil && cacheOption.ContextID != nil {
		opts_ = slices.Clone(opts)
		opts_ = append(opts_, ark.WithCache(cacheOption), model.WithTools([]*schema.ToolInfo{}))
		in_ = in[len(in)-1:]
	}

	outMsg, err = cm.ChatModel.Generate(ctx, in_, opts_...)
	if err != nil {
		return nil, err
	}

	contextID, ok := ark.GetContextID(outMsg)
	if ok {
		cacheOption.ContextID = &contextID
	}

	return outMsg, nil
}

func (cm *CachedARKChatModel) Stream(ctx context.Context, in []*schema.Message, opts ...model.Option) (
	outStream *schema.StreamReader[*schema.Message], err error) {

	opts_ := opts
	in_ := in

	cacheOption, ok := ctx.Value(cacheOptionCtxKey).(*ark.CacheOption)
	if ok && cacheOption != nil && cacheOption.ContextID != nil {
		opts_ = slices.Clone(opts)
		opts_ = append(opts_, ark.WithCache(cacheOption), model.WithTools([]*schema.ToolInfo{}))
		in_ = in[len(in)-1:]
	}

	outStream, err = cm.ChatModel.Stream(ctx, in_, opts_...)
	if err != nil {
		return nil, err
	}

	outStream_ := schema.StreamReaderWithConvert(outStream, func(msg *schema.Message) (*schema.Message, error) {
		contextID, ok := ark.GetContextID(msg)
		if ok {
			cacheOption.ContextID = &contextID
		}
		return msg, nil
	})

	return outStream_, nil
}

func (cm *CachedARKChatModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	m, err := cm.ChatModel.WithTools(tools)
	if err != nil {
		return nil, err
	}

	ncm := *cm
	ncm.ChatModel = m

	return &ncm, nil
}
