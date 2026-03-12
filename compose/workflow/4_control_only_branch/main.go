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
	"os"

	clc "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/compose"
	"github.com/coze-dev/cozeloop-go"

	"github.com/cloudwego/eino-examples/devops/visualize"
	"github.com/cloudwego/eino-examples/internal/logs"
)

func main() {
	cozeloopApiToken := os.Getenv("COZELOOP_API_TOKEN")
	cozeloopWorkspaceID := os.Getenv("COZELOOP_WORKSPACE_ID") // use cozeloop trace, from https://loop.coze.cn/open/docs/cozeloop/go-sdk#4a8c980e

	ctx := context.Background()
	var handlers []callbacks.Handler
	if cozeloopApiToken != "" && cozeloopWorkspaceID != "" {
		client, err := cozeloop.NewClient(
			cozeloop.WithAPIToken(cozeloopApiToken),
			cozeloop.WithWorkspaceID(cozeloopWorkspaceID),
		)
		if err != nil {
			panic(err)
		}
		defer client.Close(ctx)
		handlers = append(handlers, clc.NewLoopHandler(client))
	}
	callbacks.AppendGlobalHandlers(handlers...)

	bidder1 := func(ctx context.Context, in float64) (float64, error) {
		return in + 1.0, nil
	}

	bidder2 := func(ctx context.Context, in float64) (float64, error) {
		return in + 2.0, nil
	}

	announcer := func(ctx context.Context, in any) (any, error) {
		logs.Infof("bidder1 had lodged his bid!")
		return nil, nil
	}

	wf := compose.NewWorkflow[float64, map[string]float64]()

	wf.AddLambdaNode("b1", compose.InvokableLambda(bidder1)).
		AddInput(compose.START)

	// just add a node to announce bidder1 had lodged his bid!
	// It should be executed strictly after bidder1, so we use `AddDependency("b1")`.
	// Note that `AddDependency()` will only form control relationship,
	// but not data passing relationship.
	wf.AddLambdaNode("announcer", compose.InvokableLambda(announcer)).
		AddDependency("b1")

	// add a branch just like adding branch in Graph.
	wf.AddBranch("b1", compose.NewGraphBranch(func(ctx context.Context, in float64) (string, error) {
		if in > 5.0 {
			return compose.END, nil
		}
		return "b2", nil
	}, map[string]bool{compose.END: true, "b2": true}))

	wf.AddLambdaNode("b2", compose.InvokableLambda(bidder2)).
		// b2 executes strictly after b1 (through branch dependency),
		// but does not rely on b1's output,
		// which means b2 depends on b1 conditionally,
		// but no data passing between them.
		AddInputWithOptions(compose.START, nil, compose.WithNoDirectDependency())

	wf.End().AddInput("b1", compose.ToField("bidder1")).
		AddInput("b2", compose.ToField("bidder2"))

	gen := visualize.NewMermaidGenerator("compose/workflow/4_control_only_branch")
	runner, err := wf.Compile(context.Background(), compose.WithGraphCompileCallbacks(gen), compose.WithGraphName("Workflow-Control-Only-Branch"))
	if err != nil {
		logs.Errorf("workflow compile error: %v", err)
		return
	}

	// Mermaid markdown and images are auto-generated in compose/workflow/4_control_only_branch

	result, err := runner.Invoke(context.Background(), 3.0)
	if err != nil {
		logs.Errorf("workflow run err: %v", err)
		return
	}

	logs.Infof("%v", result)
}
