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
	"strings"

	clc "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/coze-dev/cozeloop-go"

	"github.com/cloudwego/eino-examples/internal/logs"
)

// demonstrates the field mapping ability of eino workflow.
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

	type counter struct {
		FullStr string // exported because we will do field mapping for this field
		SubStr  string // exported because we will do field mapping for this field
	}

	// wordCounter is a lambda function that count occurrences of SubStr within FullStr
	wordCounter := func(ctx context.Context, c counter) (int, error) {
		return strings.Count(c.FullStr, c.SubStr), nil
	}

	type message struct {
		*schema.Message        // exported because we will do field mapping for this field
		SubStr          string // exported because we will do field mapping for this field
	}

	// create a workflow just like a Graph
	wf := compose.NewWorkflow[message, map[string]any]()

	// add lambda c1 just like in Graph
	wf.AddLambdaNode("c1", compose.InvokableLambda(wordCounter)).
		AddInput(compose.START, // add an input from START, specifying 2 field mappings
			// map START's SubStr field to lambda c1's SubStr field
			compose.MapFields("SubStr", "SubStr"),
			// map START's Message's Content field to lambda c1's FullStr field
			compose.MapFieldPaths([]string{"Message", "Content"}, []string{"FullStr"}))

	// add lambda c2 just like in Graph
	wf.AddLambdaNode("c2", compose.InvokableLambda(wordCounter)).
		AddInput(compose.START, // add an input from START, specifying 2 field mappings
			// map START's SubStr field to lambda c1's SubStr field
			compose.MapFields("SubStr", "SubStr"),
			// map START's Message's ReasoningContent field to lambda c1's FullStr field
			compose.MapFieldPaths([]string{"Message", "ReasoningContent"}, []string{"FullStr"}))

	wf.End(). // Obtain the compose.END for method chaining
		// add an input from c1,
		// mapping full output of c1 to the map key 'content_count'
		AddInput("c1", compose.ToField("content_count")).
		// also add an input from c2,
		// mapping full output of c2 to the map key 'reasoning_content_count'
		AddInput("c2", compose.ToField("reasoning_content_count"))

	// compile the workflow just like compiling a Graph
	run, err := wf.Compile(context.Background())
	if err != nil {
		logs.Errorf("workflow compile error: %v", err)
		return
	}

	// invoke the workflow just like invoking a Graph
	result, err := run.Invoke(context.Background(), message{
		Message: &schema.Message{
			Role:             schema.Assistant,
			Content:          "Hello world!",
			ReasoningContent: "I need to say something meaningful",
		},
		SubStr: "o", // would like to count the occurrences of 'o'
	})
	if err != nil {
		logs.Errorf("workflow run err: %v", err)
		return
	}

	logs.Infof("%v", result)
}
