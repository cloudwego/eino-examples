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

package report

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/generic"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/params"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/tools"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/utils"
)

func NewReportAgent(ctx context.Context, operator commandline.Operator) (adk.Agent, error) {
	cm, err := utils.NewChatModel(ctx,
		utils.WithMaxTokens(15000),
		utils.WithTemperature(0.1),
		utils.WithTopP(1),
	)
	if err != nil {
		return nil, err
	}

	var imageReaderTool tool.InvokableTool
	if modelName := os.Getenv("ARK_VISION_MODEL"); modelName != "" {
		visionModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
			APIKey:  os.Getenv("ARK_VISION_API_KEY"),
			BaseURL: os.Getenv("ARK_VISION_BASE_URL"),
			Region:  os.Getenv("ARK_VISION_REGION"),
			Model:   modelName,
		})
		if err != nil {
			return nil, err
		}
		imageReaderTool = tools.NewToolImageReader(visionModel)
	}

	preprocess := []tools.ToolRequestPreprocess{tools.ToolRequestRepairJSON}
	agentTools := []tool.BaseTool{
		tools.NewWrapTool(tools.NewBashTool(operator), preprocess, nil),
		tools.NewWrapTool(tools.NewTreeTool(operator), preprocess, nil),
		tools.NewWrapTool(tools.NewEditFileTool(operator), preprocess, nil),
		tools.NewWrapTool(tools.NewReadFileTool(operator), preprocess, nil),
		tools.NewWrapTool(tools.NewToolSubmitResult(operator), preprocess, nil),
	}
	if imageReaderTool != nil {
		agentTools = append(agentTools, tools.NewWrapTool(imageReaderTool, preprocess, nil))
	}

	ra, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name: "Report",
		Description: `This is a report agent responsible for reading a file from a given file_path and generating a comprehensive report based on its contents.
Its workflow includes reading the file, analyzing the data and information, summarizing key findings and insights, and producing a clear, concise report that addresses the user's query.
If the file contains charts or visualizations, the agent will reference them appropriately in the report. The React agent should call this sub-agent when a detailed, data-driven report generation from a specified file is needed.`,
		Instruction: `You are a report agent. Your task is to read the file at the given file_path and generate a comprehensive report based on its contents.

**Workflow:**
1.  Read the content of the file specified by 'input file path' and 'working directory'.
2.  Analyze the data and information within the files.
3.  Summarize the key findings and insights.
4.  Generate a clear and concise report that addresses the user's query.
5.  If there are any charts or visualizations, refer to them in your report.
6.  If work is done, must call SubmitResult tool before finishing.
`,
		Model: cm,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: agentTools,
			},
			ReturnDirectly: tools.SubmitResultReturnDirectly,
		},
		GenModelInput: func(ctx context.Context, instruction string, input *adk.AgentInput) ([]adk.Message, error) {
			planExecuteResult := input.Messages
			if len(input.Messages) > 0 && input.Messages[len(input.Messages)-1].Role == schema.Tool {
				planExecuteResult = []*schema.Message{input.Messages[len(input.Messages)-1]}
			}

			fp, ok := params.GetTypedContextParams[string](ctx, params.FilePathSessionKey)
			if !ok {
				return nil, fmt.Errorf("file path session key not found")
			}

			plan, ok := utils.GetSessionValue[*generic.Plan](ctx, planexecute.PlanSessionKey)
			if !ok {
				return nil, fmt.Errorf("plan not found")
			}

			planStr, err := json.MarshalIndent(plan, "", "\t")
			if err != nil {
				return nil, err
			}

			wd, ok := params.GetTypedContextParams[string](ctx, params.WorkDirSessionKey)
			if !ok {
				return nil, fmt.Errorf("work dir not found")
			}

			files, err := generic.ListDir(wd)
			if err != nil {
				return nil, err
			}

			tpl := prompt.FromMessages(schema.Jinja2,
				schema.SystemMessage(instruction),
				schema.UserMessage(`
User Query: {{ user_query }}
Input File Path: {{ file_path }}
Working Directory: {{ work_dir }}
Working Directory Files: {{ work_dir_files }}
Current Time: {{ current_time }}

**Plan Details:**
{{ plan }}
`))

			msgs, err := tpl.Format(ctx, map[string]any{
				"file_path":      fp,
				"work_dir":       wd,
				"work_dir_files": utils.ToJSONString(files),
				"user_query":     utils.FormatInput(planExecuteResult),
				"plan":           string(planStr),
				"current_time":   utils.GetCurrentTime(),
			})
			if err != nil {
				return nil, err
			}

			return msgs, nil
		},
		MaxIterations: 20,
	})
	if err != nil {
		return nil, err
	}

	return ra, nil
}
