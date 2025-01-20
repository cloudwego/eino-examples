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
	"io"
	"testing"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
)

func TestTypeMatch(t *testing.T) {
	ctx := context.Background()

	g1 := compose.NewGraph[[]*schema.Message, string]()
	_ = g1.AddChatModelNode("model", &mockChatModel{})
	_ = g1.AddLambdaNode("lambda", compose.InvokableLambda(func(_ context.Context, msg *schema.Message) (string, error) {
		return msg.Content, nil
	}))
	_ = g1.AddEdge(compose.START, "model")
	_ = g1.AddEdge("model", "lambda")
	_ = g1.AddEdge("lambda", compose.END)

	runner, err := g1.Compile(ctx)
	assert.NoError(t, err)

	c, err := runner.Invoke(ctx, []*schema.Message{
		schema.UserMessage("what's the weather in beijing?"),
	})
	assert.NoError(t, err)
	assert.Equal(t, "the weather is good", c)

	s, err := runner.Stream(ctx, []*schema.Message{
		schema.UserMessage("what's the weather in beijing?"),
	})
	assert.NoError(t, err)

	var fullStr string
	for {
		chunk, err := s.Recv()
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}

		fullStr += chunk
	}
	assert.Equal(t, c, fullStr)
}
