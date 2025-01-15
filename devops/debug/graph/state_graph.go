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

package graph

import (
	"context"

	"github.com/cloudwego/eino/compose"

	"github.com/cloudwego/eino-examples/internal/logs"
)

type nodeState struct {
	Messages []string
}

func RegisterSimpleStateGraph(ctx context.Context) {
	stateFunction := func(ctx context.Context) *nodeState {
		s := &nodeState{
			Messages: make([]string, 0, 3),
		}
		return s
	}

	sg := compose.NewGraph[string, string](compose.WithGenLocalState(stateFunction))

	_ = sg.AddLambdaNode("node_1", compose.InvokableLambda(func(ctx context.Context, input string) (output string, err error) {
		return input + " process by node_1,", nil
	}), compose.WithStatePreHandler(func(ctx context.Context, input string, state *nodeState) (string, error) {
		state.Messages = append(state.Messages, input)
		return input, nil
	}))

	_ = sg.AddLambdaNode("node_2", compose.InvokableLambda(func(ctx context.Context, input string) (output string, err error) {
		return input + " process by node_2,", nil
	}), compose.WithStatePreHandler(func(ctx context.Context, input string, state *nodeState) (string, error) {
		state.Messages = append(state.Messages, input)
		return input, nil
	}))

	_ = sg.AddLambdaNode("node_3", compose.InvokableLambda(func(ctx context.Context, input string) (output string, err error) {
		return input + " process by node_3,", nil
	}), compose.WithStatePreHandler(func(ctx context.Context, input string, state *nodeState) (string, error) {
		state.Messages = append(state.Messages, input)
		return input, nil
	}))

	_ = sg.AddEdge(compose.START, "node_1")

	_ = sg.AddEdge("node_1", "node_2")

	_ = sg.AddEdge("node_2", "node_3")

	_ = sg.AddEdge("node_3", compose.END)

	r, err := sg.Compile(ctx)
	if err != nil {
		logs.Errorf("compile state graph failed, err=%v", err)
		return
	}

	message, err := r.Invoke(ctx, "eino state graph test")
	if err != nil {
		logs.Errorf("invoke state graph failed, err=%v", err)
		return
	}

	logs.Infof("eino simple state graph output is: %v", message)
}
