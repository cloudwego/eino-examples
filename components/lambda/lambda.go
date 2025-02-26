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

package lambda

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// INFO: 参考文档 https://www.cloudwego.io/zh/docs/eino/core_modules/components/lambda_guide/

type Options struct {
	Field1 string
}

type MyOption func(*Options)

type MyStruct struct {
	ID int `json:"id"`
}

func ExampleOfCreateByAnyLambda() {
	// input 和 output 类型为自定义的任何类型
	lambda, _ := compose.AnyLambda(
		// Invoke 函数
		func(ctx context.Context, input string, opts ...MyOption) (output string, err error) {
			// some logic
			return "", nil
		},
		// Stream 函数
		func(ctx context.Context, input string, opts ...MyOption) (output *schema.StreamReader[string], err error) {
			// some logic
			return nil, nil
		},
		// Collect 函数
		func(ctx context.Context, input *schema.StreamReader[string], opts ...MyOption) (output string, err error) {
			// some logic
			return "", nil
		},
		// Transform 函数
		func(ctx context.Context, input *schema.StreamReader[string], opts ...MyOption) (output *schema.StreamReader[string], err error) {
			// some logic
			return nil, nil
		},
	)
	_ = lambda
}

func ExampleOfCreateByInvokableLambdaWithOptions() {
	lambda := compose.InvokableLambdaWithOption(
		func(ctx context.Context, input string, opts ...MyOption) (output string, err error) {
			// 处理 opts
			// some logic
			return "", nil
		},
	)
	_ = lambda
}

func ExampleOfCreateByInvokableLambda() {
	lambda := compose.InvokableLambda(
		func(ctx context.Context, input string) (output string, err error) {
			// some logic
			return "", nil
		},
	)
	_ = lambda
}

func ExampleOfLambdaInChain() {
	chain := compose.NewChain[string, string]()
	chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, input string) (string, error) {
		// some logic
		return "", nil
	}))

	_ = chain
}

func ExampleOfLambdaInGraph() {
	graph := compose.NewGraph[string, *MyStruct]()
	graph.AddLambdaNode(
		"node1",
		compose.InvokableLambda(func(ctx context.Context, input string) (*MyStruct, error) {
			// some logic
			return &MyStruct{ID: 1}, nil
		}),
	)

	_ = graph
}

func ExampleOfToListLambda() {
	chatModel, _ := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		Model:  "gpt-4o",
		APIKey: "",
	})

	// 创建一个 ToList Lambda
	lambda := compose.ToList[*schema.Message]()

	// 在 Chain 中使用
	chain := compose.NewChain[[]*schema.Message, []*schema.Message]()
	chain.AppendChatModel(chatModel) // chatModel 返回 *schema.Message
	chain.AppendLambda(lambda)       // 将 *schema.Message 转换为 []*schema.Message

	_ = chain
}

func ExampleOfMessageParserLambda() {
	// 创建解析器
	parser := schema.NewMessageJSONParser[*MyStruct](&schema.MessageJSONParseConfig{
		ParseFrom:    schema.MessageParseFromContent,
		ParseKeyPath: "", // 如果仅需要 parse 子字段，可用 "key.sub.grandsub"
	})

	// 创建解析 Lambda
	parserLambda := compose.MessageParser(parser)

	// 在 Chain 中使用
	chain := compose.NewChain[*schema.Message, *MyStruct]()
	chain.AppendLambda(parserLambda)

	// 使用示例
	runner, _ := chain.Compile(context.Background())
	parsed, _ := runner.Invoke(context.Background(), &schema.Message{
		Content: `{"id": 1}`,
	})
	// parsed.ID == 1

	_ = parsed
}
