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

func RegisterSimpleGraph(ctx context.Context) {
	g := compose.NewGraph[string, string]()

	_ = g.AddLambdaNode("node_1", compose.InvokableLambda(func(ctx context.Context, input string) (output string, err error) {
		return input + " process by node_1,", nil
	}))

	sg := compose.NewGraph[string, string]()
	_ = sg.AddLambdaNode("sg_node_1", compose.InvokableLambda(func(ctx context.Context, input string) (output string, err error) {
		return input + " process by sg_node_1,", nil
	}))

	_ = sg.AddEdge(compose.START, "sg_node_1")

	_ = sg.AddEdge("sg_node_1", compose.END)

	_ = g.AddGraphNode("node_2", sg)

	_ = g.AddLambdaNode("node_3", compose.InvokableLambda(func(ctx context.Context, input string) (output string, err error) {
		return input + " process by node_3,", nil
	}))

	_ = g.AddEdge(compose.START, "node_1")

	_ = g.AddEdge("node_1", "node_2")

	_ = g.AddEdge("node_2", "node_3")

	_ = g.AddEdge("node_3", compose.END)

	r, err := g.Compile(ctx)
	if err != nil {
		logs.Errorf("compile graph failed, err=%v", err)
		return
	}

	message, err := r.Invoke(ctx, "eino graph test")
	if err != nil {
		logs.Errorf("invoke graph failed, err=%v", err)
		return
	}

	logs.Infof("eino simple graph output is: %v", message)
}
