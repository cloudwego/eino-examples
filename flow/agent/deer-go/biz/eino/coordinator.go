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
	"time"

	"github.com/RanFeng/ilog"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/flow/agent/deer-go/biz/consts"
	"github.com/cloudwego/eino-examples/flow/agent/deer-go/biz/infra"
	"github.com/cloudwego/eino-examples/flow/agent/deer-go/biz/model"
)

func loadMsg(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*model.State](ctx, func(_ context.Context, state *model.State) error {
		sysPrompt, err := infra.GetPromptTemplate(ctx, name)
		if err != nil {
			ilog.EventInfo(ctx, "get prompt template fail")
			return err
		}

		promptTemp := prompt.FromMessages(schema.Jinja2,
			schema.SystemMessage(sysPrompt),
			schema.MessagesPlaceholder("user_input", true),
		)

		variables := map[string]any{
			"locale":              state.Locale,
			"max_step_num":        state.MaxStepNum,
			"max_plan_iterations": state.MaxPlanIterations,
			"CURRENT_TIME":        time.Now().Format("2006-01-02 15:04:05"),
			"user_input":          state.Messages,
		}
		output, err = promptTemp.Format(context.Background(), variables)
		return err
	})
	return output, err
}

func router(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	err = compose.ProcessState[*model.State](ctx, func(_ context.Context, state *model.State) error {
		defer func() {
			output = state.Goto
		}()
		state.Goto = compose.END
		if len(input.ToolCalls) > 0 && input.ToolCalls[0].Function.Name == "hand_to_planner" {
			argMap := map[string]string{}
			_ = json.Unmarshal([]byte(input.ToolCalls[0].Function.Arguments), &argMap)
			state.Locale, _ = argMap["locale"]
			if state.EnableBackgroundInvestigation {
				state.Goto = consts.BackgroundInvestigator
			} else {
				state.Goto = consts.Planner
			}
		}
		return nil
	})
	return output, nil
}

func NewCAgent[I, O any](ctx context.Context) *compose.Graph[I, O] {
	cag := compose.NewGraph[I, O]()

	hand_to_planner := &schema.ToolInfo{
		Name: "hand_to_planner",
		Desc: "Handoff to planner agent to do plan.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"task_title": {
				Type:     schema.String,
				Desc:     "The title of the task to be handed off.",
				Required: true,
			},
			"locale": {
				Type:     schema.String,
				Desc:     "The user's detected language locale (e.g., en-US, zh-CN).",
				Required: true,
			},
		}),
	}

	coorModel, _ := infra.ChatModel.WithTools([]*schema.ToolInfo{hand_to_planner})

	_ = cag.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadMsg))
	_ = cag.AddChatModelNode("agent", coorModel)
	_ = cag.AddLambdaNode("router", compose.InvokableLambdaWithOption(router))

	_ = cag.AddEdge(compose.START, "load")
	_ = cag.AddEdge("load", "agent")
	_ = cag.AddEdge("agent", "router")
	_ = cag.AddEdge("router", compose.END)
	return cag
}
