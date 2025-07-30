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

package withoption

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/internal/gptr"
)

// <<<<<< ImplSpecificOption >>>>>>>

type ImplOption struct {
	UsingProxy      bool
	SomeOptionField string
	// ...

}

func WithUsingProxy(b bool) tool.Option {
	// wrap to tool.Option.
	return tool.WrapImplSpecificOptFn(func(t *ImplOption) {
		t.UsingProxy = b
	})
}

func WithSomeOptionField(s string) tool.Option {
	return tool.WrapImplSpecificOptFn(func(t *ImplOption) {
		t.SomeOptionField = s
	})
}

// <<<<<< Tool implementation >>>>>>>

type FakeOptionTool struct{}

type ToolInput struct {
	SomeInField string `json:"some_in_field" jsonschema:"description=some desc of this field"`
}

type ToolOutput struct {
	SomeOutField string `json:"some_out_field"`
}

// Info impl tool.BaseTool.
func (t *FakeOptionTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return utils.GoStruct2ToolInfo[ToolInput]("fake_tool", "fake tool to show how to use tool option")
}

// InvokableRun impl tool.InvokableTool.
func (t *FakeOptionTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	in := ToolInput{}

	err := json.Unmarshal([]byte(argumentsInJSON), &in)
	if err != nil {
		return "", err
	}

	op := &ImplOption{
		UsingProxy:      false,
		SomeOptionField: "some default value",
	}

	// Get specific option struct.
	op = tool.GetImplSpecificOptions(op, opts...)

	res, err := t.invoke(ctx, in, op)
	if err != nil {
		return "", err
	}

	b, err := json.Marshal(res)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func (t *FakeOptionTool) invoke(ctx context.Context, in ToolInput, option *ImplOption) (*ToolOutput, error) {
	fmt.Println("=== receve tool invoke ===")
	fmt.Printf("Input: %+v\n", in)
	fmt.Printf("Option: %+v\n", option)
	fmt.Println("===========")
	// do some logic here
	return &ToolOutput{
		SomeOutField: "fake out value",
	}, nil
}

func HowToUse() {
	ctx := context.Background()

	tn, err := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
		Tools: []tool.BaseTool{&FakeOptionTool{}},
	})
	if err != nil {
		panic(err)
	}

	// Mostly, ToolsNode is after a ChatModel node, just a demo to show how to use tool option below.
	chain := compose.NewChain[*schema.Message, []*schema.Message]()
	chain.AppendToolsNode(tn)

	r, err := chain.Compile(ctx)
	if err != nil {
		panic(err)
	}

	toolCallMsg := schema.AssistantMessage("", []schema.ToolCall{{
		Index: gptr.Of(0),
		ID:    "fc-xxxx",
		Function: schema.FunctionCall{
			Name:      "fake_tool",
			Arguments: `{"some_in_field":"input value of some in field"}`,
		},
	}})

	// must wrap tool option by compose.WithToolsNodeOption() and compose.WithToolOption().
	res, err := r.Invoke(ctx, toolCallMsg, compose.WithToolsNodeOption(compose.WithToolOption(
		WithUsingProxy(true),
		WithSomeOptionField("some changed field value"),
	)))

	if err != nil {
		panic(err)
	}

	for _, r := range res {
		fmt.Printf("%+v\n", r)
	}
}
