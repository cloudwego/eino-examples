package executor

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/params"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/tools"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/utils"
	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func newCodeAgent(ctx context.Context, operator commandline.Operator) (adk.Agent, error) {
	cm, err := utils.NewChatModel(ctx,
		utils.WithMaxTokens(14125),
		utils.WithTemperature(float32(1)),
		utils.WithTopP(float32(1)),
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
		tools.NewWrapTool(tools.NewBashTool(operator), preprocess, []tools.ToolResponsePostprocess{tools.FilePostProcess}),
		tools.NewWrapTool(tools.NewTreeTool(operator), preprocess, nil),
		tools.NewWrapTool(tools.NewEditFileTool(operator), preprocess, []tools.ToolResponsePostprocess{tools.EditFilePostProcess}),
		tools.NewWrapTool(tools.NewReadFileTool(operator), preprocess, nil), // TODO: compress post process
		tools.NewWrapTool(tools.NewPythonRunnerTool(operator), preprocess, []tools.ToolResponsePostprocess{tools.FilePostProcess}),
	}
	if imageReaderTool != nil {
		agentTools = append(agentTools, tools.NewWrapTool(imageReaderTool, preprocess, nil))
	}

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
				Tools: agentTools,
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
