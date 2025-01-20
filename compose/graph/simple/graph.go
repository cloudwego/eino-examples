/*
 * Copyright 2024 CloudWeGo Authors
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
	"fmt"
	"io"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

const (
	nodeOfModel  = "model"
	nodeOfPrompt = "prompt"
)

func main() {
	ctx := context.Background()
	g := compose.NewGraph[map[string]any, *schema.Message]()

	pt := prompt.FromMessages(
		schema.FString,
		schema.UserMessage("what's the weather in {location}?"),
	)

	_ = g.AddChatTemplateNode(nodeOfPrompt, pt) // error check can be skipped here because in Compile, we will check the error happened here too
	_ = g.AddChatModelNode(nodeOfModel, &mockChatModel{}, compose.WithNodeName("ChatModel"))
	_ = g.AddEdge(compose.START, nodeOfPrompt)
	_ = g.AddEdge(nodeOfPrompt, nodeOfModel)
	_ = g.AddEdge(nodeOfModel, compose.END)

	r, err := g.Compile(ctx, compose.WithMaxRunSteps(10))
	if err != nil {
		panic(err)
	}

	in := map[string]any{"location": "beijing"}
	ret, err := r.Invoke(ctx, in)
	if err != nil {
		panic(err)
	}
	fmt.Println("invoke result: ", ret)

	// stream
	s, err := r.Stream(ctx, in)
	if err != nil {
		panic(err)
	}

	defer s.Close()
	for {
		chunk, err := s.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}

		fmt.Println("stream chunk: ", chunk)
	}
}

type mockChatModel struct{}

func (m *mockChatModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	return schema.AssistantMessage("the weather is good", nil), nil
}

func (m *mockChatModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	sr, sw := schema.Pipe[*schema.Message](0)
	go func() {
		defer sw.Close()
		sw.Send(schema.AssistantMessage("the weather is", nil), nil)
		sw.Send(schema.AssistantMessage(" good", nil), nil)
	}()
	return sr, nil
}

func (m *mockChatModel) BindTools(tools []*schema.ToolInfo) error {
	panic("implement me")
}
