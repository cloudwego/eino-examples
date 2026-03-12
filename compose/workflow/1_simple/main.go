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
	"strconv"

	clc "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/compose"
	"github.com/coze-dev/cozeloop-go"

	"github.com/cloudwego/eino-examples/internal/logs"
)

// creates and invokes a simple workflow with only a Lambda node.
// Since all field mappings are ALL to ALL mappings
// (by using AddInput without field mappings),
// this simple workflow is equivalent to a Graph: START -> lambda -> END.
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

	// create a Workflow, just like creating a Graph
	wf := compose.NewWorkflow[int, string]()

	// add a lambda node to the Workflow, just like adding the lambda to a Graph
	wf.AddLambdaNode("lambda", compose.InvokableLambda(
		func(ctx context.Context, in int) (string, error) {
			return strconv.Itoa(in), nil
		})).
		// add an input to this lambda node from START.
		// this means mapping all output of START to the input of the lambda.
		// the effect of AddInput is to set both a control dependency
		// and a data dependency.
		AddInput(compose.START)

	// obtain the compose.END of the workflow for method chaining
	wf.End().
		// add an input to compose.END,
		// which means 'using ALL output of lambda node as output of END'.
		AddInput("lambda")

	// compile the Workflow, just like compiling a Graph
	run, err := wf.Compile(context.Background())
	if err != nil {
		logs.Errorf("workflow compile error: %v", err)
		return
	}

	// invoke the Workflow, just like invoking a Graph
	result, err := run.Invoke(context.Background(), 1)
	if err != nil {
		logs.Errorf("workflow run err: %v", err)
		return
	}

	logs.Infof("%v", result)
}
