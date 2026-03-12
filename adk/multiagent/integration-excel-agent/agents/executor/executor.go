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

package executor

import (
	"context"

	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/utils"
)

var executorPrompt = prompt.FromMessages(schema.FString,
	schema.SystemMessage(`You are a diligent and meticulous executor agent. Follow the given plan and execute your tasks carefully and thoroughly.

Available Tools:
- CodeAgent: This tool is a code agent specialized in Excel file handling. It takes step-by-step plans, processes each task by generating Python code (leveraging pandas for data analysis/manipulation, matplotlib for plotting/visualization, and openpyxl for Excel reading/writing), and executes tasks sequentially. The React agent should invoke it when stepwise Python coding for Excel operations is needed to ensure precise, efficient task completion.

Notice:
- Do not transfer to other agents, use tools only.
`),
	schema.UserMessage(`## OBJECTIVE
{input}
## Given the following plan:
{plan}
## COMPLETED STEPS & RESULTS
{executed_steps}
## Your task is to execute the first step, which is: 
{step}`))

func NewExecutor(ctx context.Context, operator commandline.Operator) (adk.Agent, error) {
	cm, err := utils.NewChatModel(ctx,
		utils.WithMaxTokens(4096),
		utils.WithTemperature(float32(0)),
		utils.WithTopP(float32(0)),
	)
	if err != nil {
		return nil, err
	}

	ca, err := newCodeAgent(ctx, operator)
	if err != nil {
		return nil, err
	}

	sa, err := newWebSearchAgent(ctx)
	if err != nil {
		return nil, err
	}

	a, err := planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
		Model: cm,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					adk.NewAgentTool(ctx, ca),
					adk.NewAgentTool(ctx, sa),
				},
			},
		},
		MaxIterations: 20,
		GenInputFn: func(ctx context.Context, in *planexecute.ExecutionContext) ([]adk.Message, error) {
			planContent, err := in.Plan.MarshalJSON()
			if err != nil {
				return nil, err
			}

			return executorPrompt.Format(ctx, map[string]any{
				"input":          utils.FormatInput(in.UserInput),
				"plan":           string(planContent),
				"executed_steps": utils.FormatExecutedSteps(in.ExecutedSteps),
				"step":           in.Plan.FirstStep(),
			})
		},
	})
	if err != nil {
		return nil, err
	}

	return a, nil
}
