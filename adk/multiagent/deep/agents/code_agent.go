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

package agents

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/adk/multiagent/deep/params"
	"github.com/cloudwego/eino-examples/adk/multiagent/deep/tools"
	"github.com/cloudwego/eino-examples/adk/multiagent/deep/utils"
)

func NewCodeAgent(ctx context.Context, operator commandline.Operator) (adk.Agent, error) {
	cm, err := utils.NewChatModel(ctx,
		utils.WithMaxTokens(14125),
		utils.WithTemperature(float32(1)),
		utils.WithTopP(float32(1)),
	)
	if err != nil {
		return nil, err
	}

	preprocess := []tools.ToolRequestPreprocess{tools.ToolRequestRepairJSON}

	ca, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name: "CodeAgent",
		Description: `This sub-agent is a code agent specialized in handling Excel files. 
It receives a clear task and accomplish the task by generating Python code and execute it. 
The agent leverages pandas for data analysis and manipulation, matplotlib for plotting and visualization, and openpyxl for reading and writing Excel files. 
The React agent should invoke this sub-agent whenever stepwise Python coding for Excel file operations is required, ensuring precise and efficient task execution.`,
		Instruction: `You are a code agent. Your workflow is as follows:
1. You will be given a clear task to handle Excel files.
2. You should analyse the task and use right tools to help coding.
3. You should write python code to finish the task.
4. You are preferred to write code execution result to another file for further usages. 

You are in a react mode, and you should use the following libraries to help you finish the task:
- pandas: for data analysis and manipulation
- matplotlib: for plotting and visualization
- openpyxl: for reading and writing Excel files

Notice:
1. Tool Calls argument must be a valid json.
2. Tool Calls argument should do not contains invalid suffix like ']<|FunctionCallEnd|>'. 
`,
		Model: cm,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					tools.NewWrapTool(tools.NewBashTool(operator), preprocess, []tools.ToolResponsePostprocess{tools.FilePostProcess}),
					tools.NewWrapTool(tools.NewTreeTool(operator), preprocess, nil),
					tools.NewWrapTool(tools.NewEditFileTool(operator), preprocess, []tools.ToolResponsePostprocess{tools.EditFilePostProcess}),
					tools.NewWrapTool(tools.NewReadFileTool(operator), preprocess, nil), // TODO: compress post process
					tools.NewWrapTool(tools.NewPythonRunnerTool(operator), preprocess, []tools.ToolResponsePostprocess{tools.FilePostProcess}),
				},
			},
		},
		GenModelInput: func(ctx context.Context, instruction string, input *adk.AgentInput) ([]adk.Message, error) {
			wd, ok := params.GetTypedContextParams[string](ctx, params.WorkDirSessionKey)
			if !ok {
				return nil, fmt.Errorf("work dir not found")
			}

			tpl := prompt.FromMessages(schema.Jinja2,
				schema.SystemMessage(instruction),
				schema.UserMessage(`WorkingDirectory: {{ working_dir }}
UserQuery: {{ user_query }}
CurrentTime: {{ current_time }}
`))

			msgs, err := tpl.Format(ctx, map[string]any{
				"working_dir":  wd,
				"user_query":   utils.FormatInput(input.Messages),
				"current_time": utils.GetCurrentTime(),
			})
			if err != nil {
				return nil, err
			}

			return msgs, nil
		},
		OutputKey:     "",
		MaxIterations: 1000,
	})
	if err != nil {
		return nil, err
	}

	return ca, nil
}
