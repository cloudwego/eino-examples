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
	"io"
	"os"
	"strings"

	clc "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/coze-dev/cozeloop-go"

	"github.com/cloudwego/eino-examples/internal/logs"
)

// demonstrates the stream field mapping ability of eino workflow.
// It's modified from 2_field_mapping.
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

	// wordCounter is a transformable lambda function that
	// count occurrences of SubStr within FullStr, for each trunk.
	wordCounter := func(ctx context.Context, c *schema.StreamReader[counter]) (
		*schema.StreamReader[int], error) {
		var subStr, cachedStr string
		return schema.StreamReaderWithConvert(c, func(co counter) (int, error) {
			if len(co.SubStr) > 0 {
				// static values will not always come in the first chunk,
				// so before the static value (SubStr) comes in,
				// we need to cache the full string
				subStr = co.SubStr
				fullStr := cachedStr + co.FullStr
				cachedStr = ""
				return strings.Count(fullStr, subStr), nil
			}

			if len(subStr) > 0 {
				return strings.Count(co.FullStr, subStr), nil
			}
			cachedStr += co.FullStr
			return 0, schema.ErrNoValue
		}), nil
	}

	// create a workflow just like a Graph
	wf := compose.NewWorkflow[*schema.Message, map[string]int]()

	// add lambda c1 just like in Graph
	wf.AddLambdaNode("c1", compose.TransformableLambda(wordCounter)).
		AddInput(compose.START, // add an input from START, specifying 2 field mappings
			// map START's Message's Content field to lambda c1's FullStr field
			compose.MapFields("Content", "FullStr")).
		// we can set static values even if the input will be stream
		SetStaticValue([]string{"SubStr"}, "o")

	// add lambda c2 just like in Graph
	wf.AddLambdaNode("c2", compose.TransformableLambda(wordCounter)).
		AddInput(compose.START, // add an input from START, specifying 2 field mappings
			// map START's Message's ReasoningContent field to lambda c1's FullStr field
			compose.MapFields("ReasoningContent", "FullStr")).
		SetStaticValue([]string{"SubStr"}, "o")

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

	// call the workflow using Transform just like calling a Graph with Transform
	result, err := run.Transform(context.Background(),
		schema.StreamReaderFromArray([]*schema.Message{
			{
				Role:             schema.Assistant,
				ReasoningContent: "I need to say something meaningful",
			},
			{
				Role:    schema.Assistant,
				Content: "Hello world!",
			},
		}))
	if err != nil {
		logs.Errorf("workflow run err: %v", err)
		return
	}

	var contentCount, reasoningCount int
	for {
		chunk, err := result.Recv()
		if err != nil {
			if err == io.EOF {
				result.Close()
				break
			}

			logs.Errorf("workflow receive err: %v", err)
			return
		}

		logs.Infof("%v", chunk)

		contentCount += chunk["content_count"]
		reasoningCount += chunk["reasoning_content_count"]
	}

	logs.Infof("content count: %d", contentCount)
	logs.Infof("reasoning count: %d", reasoningCount)
}
