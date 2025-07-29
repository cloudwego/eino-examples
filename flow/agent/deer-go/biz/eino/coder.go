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
	"fmt"
	"strings"
	"time"

	"github.com/RanFeng/ilog"
	"github.com/cloudwego/eino-ext/components/tool/mcp"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/flow/agent/deer-go/biz/consts"
	"github.com/cloudwego/eino-examples/flow/agent/deer-go/biz/infra"
	"github.com/cloudwego/eino-examples/flow/agent/deer-go/biz/model"
)

func loadCoderMsg(ctx context.Context, name string, opts ...any) (output []*schema.Message, err error) {
	err = compose.ProcessState[*model.State](ctx, func(_ context.Context, state *model.State) error {
		sysPrompt, err := infra.GetPromptTemplate(ctx, name)
		if err != nil {
			ilog.EventError(ctx, err, "get prompt template error")
			return err
		}

		promptTemp := prompt.FromMessages(schema.Jinja2,
			schema.SystemMessage(sysPrompt),
			schema.MessagesPlaceholder("user_input", true),
		)

		var curStep *model.Step
		for i := range state.CurrentPlan.Steps {
			if state.CurrentPlan.Steps[i].ExecutionRes == nil {
				curStep = &state.CurrentPlan.Steps[i]
				break
			}
		}

		if curStep == nil {
			panic("no step found")
		}

		msg := []*schema.Message{}
		msg = append(msg,
			schema.UserMessage(fmt.Sprintf("#Task\n\n##title\n\n %v \n\n##description\n\n %v \n\n##locale\n\n %v", curStep.Title, curStep.Description, state.Locale)),
		)
		variables := map[string]any{
			"locale":              state.Locale,
			"max_step_num":        state.MaxStepNum,
			"max_plan_iterations": state.MaxPlanIterations,
			"CURRENT_TIME":        time.Now().Format("2006-01-02 15:04:05"),
			"user_input":          msg,
		}
		output, err = promptTemp.Format(context.Background(), variables)
		return err
	})
	return output, err
}

func routerCoder(ctx context.Context, input *schema.Message, opts ...any) (output string, err error) {
	//ilog.EventInfo(ctx, "routerResearcher", "input", input)
	last := input
	err = compose.ProcessState[*model.State](ctx, func(_ context.Context, state *model.State) error {
		defer func() {
			output = state.Goto
		}()
		for i, step := range state.CurrentPlan.Steps {
			if step.ExecutionRes == nil {
				str := strings.Clone(last.Content)
				state.CurrentPlan.Steps[i].ExecutionRes = &str
				break
			}
		}
		ilog.EventInfo(ctx, "coder_end", "plan", state.CurrentPlan)
		state.Goto = consts.ResearchTeam
		return nil
	})
	return output, nil
}

func modifyCoderfunc(ctx context.Context, input []*schema.Message) []*schema.Message {
	sum := 0
	maxLimit := 50000
	for i := range input {
		if input[i] == nil {
			ilog.EventWarn(ctx, "modify_inputfunc_nil", "input", input[i])
			continue
		}
		l := len(input[i].Content)
		if l > maxLimit {
			ilog.EventWarn(ctx, "modify_inputfunc_clip", "raw_len", l)
			input[i].Content = input[i].Content[l-maxLimit:]
		}
		sum += len(input[i].Content)
	}
	ilog.EventInfo(ctx, "modify_inputfunc", "sum", sum, "input_len", input)
	return input
}

func NewCoder[I, O any](ctx context.Context) *compose.Graph[I, O] {
	cag := compose.NewGraph[I, O]()

	researchTools := []tool.BaseTool{}
	for mcpName, cli := range infra.MCPServer {
		ts, err := mcp.GetTools(ctx, &mcp.Config{Cli: cli})
		if err != nil {
			ilog.EventError(ctx, err, "builder_error")
		}
		if strings.HasPrefix(mcpName, "python") {
			researchTools = append(researchTools, ts...)
		}
	}
	ilog.EventDebug(ctx, "coder_end", "coder_tools", researchTools)

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		MaxStep:               40,
		ToolCallingModel:      infra.ChatModel,
		ToolsConfig:           compose.ToolsNodeConfig{Tools: researchTools},
		MessageModifier:       modifyCoderfunc,
		StreamToolCallChecker: toolCallChecker,
	})

	agentLambda, err := compose.AnyLambda(agent.Generate, agent.Stream, nil, nil)
	if err != nil {
		panic(err)
	}

	_ = cag.AddLambdaNode("load", compose.InvokableLambdaWithOption(loadCoderMsg))
	_ = cag.AddLambdaNode("agent", agentLambda)
	_ = cag.AddLambdaNode("router", compose.InvokableLambdaWithOption(routerCoder))

	_ = cag.AddEdge(compose.START, "load")
	_ = cag.AddEdge("load", "agent")
	_ = cag.AddEdge("agent", "router")
	_ = cag.AddEdge("router", compose.END)
	return cag
}
