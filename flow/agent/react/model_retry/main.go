/*
 * Copyright 2026 CloudWeGo Authors
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
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx := context.Background()

	chatModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey: os.Getenv("ARK_API_KEY"),
		Model:  os.Getenv("ARK_MODEL_NAME"),
	})
	if err != nil {
		panic(err)
	}

	retryModel := &retryWrapper{
		model:      chatModel,
		maxRetries: 2,
	}

	rAgent, err := react.NewAgent(ctx, &react.AgentConfig{
		ToolCallingModel: retryModel,
	})
	if err != nil {
		panic(err)
	}

	msgFutureOpt, msgFuture := react.WithMessageFuture()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		processMessageFuture(msgFuture)
	}()

	sr, err := rAgent.Stream(ctx, []*schema.Message{
		{Role: schema.User, Content: "你好，请介绍一下你自己"},
	}, msgFutureOpt)
	if err != nil {
		fmt.Printf("stream error: %v\n", err)
	} else {
		defer sr.Close()
		for {
			_, err := sr.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				fmt.Printf("sr recv error: %v\n", err)
				break
			}
		}
	}

	wg.Wait()
	fmt.Println("\n========== Finished ==========")
}

func processMessageFuture(msgFuture react.MessageFuture) {
	iter := msgFuture.GetMessageStreams()
	for {
		sr, ok, err := iter.Next()
		if err != nil {
			fmt.Printf("error: %v\n", err)
			return
		}
		if !ok {
			break
		}

		for {
			msg, err := sr.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				fmt.Printf("recv error: %v\n", err)
				return
			}
			if msg.Content != "" {
				fmt.Print(msg.Content)
			}
		}
	}
}

type retryWrapper struct {
	model      model.ToolCallingChatModel
	maxRetries int
}

func (r *retryWrapper) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	var lastErr error
	for i := 0; i <= r.maxRetries; i++ {
		if i > 0 {
			fmt.Printf("[Retry] attempt %d/%d\n", i, r.maxRetries)
		}
		msg, err := r.model.Generate(ctx, input, opts...)
		if err == nil {
			return msg, nil
		}
		lastErr = err
		fmt.Printf("[Error] %v\n", err)
	}
	return nil, fmt.Errorf("all retries failed: %w", lastErr)
}

func (r *retryWrapper) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	var lastErr error
	for i := 0; i <= r.maxRetries; i++ {
		if i > 0 {
			fmt.Printf("[Retry] attempt %d/%d\n", i, r.maxRetries)
		}
		sr, err := r.model.Stream(ctx, input, opts...)
		if err == nil {
			return sr, nil
		}
		lastErr = err
		fmt.Printf("[Error] %v\n", err)
	}
	return nil, fmt.Errorf("all retries failed: %w", lastErr)
}

func (r *retryWrapper) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	wrapped, err := r.model.WithTools(tools)
	if err != nil {
		return nil, err
	}
	return &retryWrapper{
		model:      wrapped,
		maxRetries: r.maxRetries,
	}, nil
}
