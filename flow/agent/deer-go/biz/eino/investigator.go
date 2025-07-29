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

package eino

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/RanFeng/ilog"
	"github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"

	"github.com/cloudwego/eino-examples/flow/agent/deer-go/biz/consts"
	"github.com/cloudwego/eino-examples/flow/agent/deer-go/biz/infra"
	"github.com/cloudwego/eino-examples/flow/agent/deer-go/biz/model"
)

func search(ctx context.Context, name string, opts ...any) (output string, err error) {
	var searchTool tool.InvokableTool
	for _, cli := range infra.MCPServer {
		if searchTool != nil {
			break
		}
		ts, err := mcp.GetTools(ctx, &mcp.Config{Cli: cli})
		if err != nil {
			ilog.EventError(ctx, err, "builder_error")
			continue
		}
		for _, t := range ts {
			info, _ := t.Info(ctx)
			if strings.HasSuffix(info.Name, "search") {
				searchTool, _ = t.(tool.InvokableTool)
				break
			}
		}
	}

	err = compose.ProcessState[*model.State](ctx, func(_ context.Context, state *model.State) error {
		args := map[string]any{
			"query": state.Messages[len(state.Messages)-1].Content,
		}
		argsBytes, err := json.Marshal(args)
		if err != nil {
			ilog.EventError(ctx, err, "json_marshal_error")
			return err
		}
		result, err := searchTool.InvokableRun(ctx, string(argsBytes))
		if err != nil {
			ilog.EventError(ctx, err, "search_result_error")
		}
		ilog.EventDebug(ctx, "back_search_result", "result", result)
		state.BackgroundInvestigationResults = result
		return nil
	})
	return output, err
}

func bIRouter(ctx context.Context, input string, opts ...any) (output string, err error) {
	err = compose.ProcessState[*model.State](ctx, func(_ context.Context, state *model.State) error {
		defer func() {
			output = state.Goto
		}()
		state.Goto = consts.Planner
		return nil
	})
	return output, nil
}

func NewBAgent[I, O any](ctx context.Context) *compose.Graph[I, O] {
	cag := compose.NewGraph[I, O]()

	_ = cag.AddLambdaNode("search", compose.InvokableLambdaWithOption(search))
	_ = cag.AddLambdaNode("router", compose.InvokableLambdaWithOption(bIRouter))

	_ = cag.AddEdge(compose.START, "search")
	_ = cag.AddEdge("search", "router")
	_ = cag.AddEdge("router", compose.END)
	return cag
}
