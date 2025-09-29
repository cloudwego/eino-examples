package executor

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/agents"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/consts"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/tools"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/utils"
	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func newCodeAgent(ctx context.Context, operator commandline.Operator) (adk.Agent, error) {
	cm, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey:      os.Getenv("ARK_API_KEY"),
		BaseURL:     os.Getenv("ARK_BASE_URL"),
		Region:      os.Getenv("ARK_REGION"),
		Model:       os.Getenv("ARK_MODEL"),
		Temperature: utils.PtrOf(float32(1)),
		MaxTokens:   utils.PtrOf(14125),
		TopP:        utils.PtrOf(float32(1)),
	})
	if err != nil {
		return nil, err
	}

	getOrDefault := func(key, keyDefault string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return os.Getenv(keyDefault)
	}

	visionModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey:  getOrDefault("ARK_VISION_API_KEY", "ARK_API_KEY"),
		BaseURL: getOrDefault("ARK_VISION_BASE_URL", "ARK_BASE_URL"),
		Region:  getOrDefault("ARK_VISION_REGION", "ARK_REGION"),
		Model:   getOrDefault("ARK_VISION_MODEL", "ARK_MODEL"),
	})
	if err != nil {
		return nil, err
	}

	preprocess := []tools.ToolRequestPreprocess{tools.ToolRequestRepairJSON}
	ca, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name: "CodeAgent",
		Description: `This sub-agent is a code agent specialized in handling Excel files. 
It receives a step-by-step plan and processes one step at a time by generating Python code to accomplish each task. 
The agent leverages pandas for data analysis and manipulation, matplotlib for plotting and visualization, and openpyxl for reading and writing Excel files. 
The React agent should invoke this sub-agent whenever stepwise Python coding for Excel file operations is required, ensuring precise and efficient task execution.`,
		Instruction: `You are a code agent. Your workflow is as follows:
1. You will be given a step-by-step plan to handle Excel files.
2. You will be given one step at a time.
3. You should write python code to finish the step.

You are in a react mode, and you should use the following libraries to help you finish the task:
- pandas: for data analysis and manipulation
- matplotlib: for plotting and visualization
- openpyxl: for reading and writing Excel files`,
		Model: cm,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					tools.NewWrapTool(tools.NewBashTool(operator), preprocess, []tools.ToolResponsePostprocess{tools.FilePostProcess}),
					tools.NewWrapTool(tools.NewTreeTool(operator), preprocess, nil),
					tools.NewWrapTool(tools.NewEditFileTool(operator), preprocess, []tools.ToolResponsePostprocess{tools.EditFilePostProcess}),
					tools.NewWrapTool(tools.NewSubmitResultTool(), preprocess, nil),
					tools.NewWrapTool(tools.NewToolImageReader(visionModel), preprocess, nil),
					tools.NewWrapTool(tools.NewReadFileTool(operator), preprocess, nil), // TODO: compress post process
					tools.NewWrapTool(tools.NewPythonRunnerTool(operator), preprocess, []tools.ToolResponsePostprocess{tools.FilePostProcess}),
				},
			},
		},
		GenModelInput: func(ctx context.Context, instruction string, input *adk.AgentInput) ([]adk.Message, error) {
			wd, ok := consts.GetSessionValue[string](ctx, consts.WorkDirSessionKey)
			if !ok {
				return nil, fmt.Errorf("work dir not found")
			}

			plan, ok := consts.GetSessionValue[*agents.Plan](ctx, planexecute.PlanSessionKey)
			if !ok {
				return nil, fmt.Errorf("plan not found")
			}

			executedSteps, ok := consts.GetSessionValue[[]planexecute.ExecutedStep](ctx, planexecute.ExecutedStepsSessionKey)
			if ok {
				return nil, fmt.Errorf("executed steps not found")
			}

			remainingSteps, err := plan.MarshalJSON()
			if err != nil {
				return nil, err
			}

			tpl := prompt.FromMessages(schema.Jinja2,
				schema.SystemMessage(instruction),
				schema.UserMessage(`WorkingDirectory: {{ working_dir }}
UserQuery: {{ user_query }}
RemainingSteps: {{ remaining_steps }}
CurrentStep: {{ current_step }}
ExecutedSteps: {{ executed_steps }}
CurrentTime: {{ current_time }}
`))

			msgs, err := tpl.Format(ctx, map[string]any{
				"working_dir":     wd,
				"user_query":      utils.FormatInput(input.Messages),
				"current_step":    plan.FirstStep(),
				"executed_steps":  executedSteps,
				"remaining_steps": string(remainingSteps),
				"current_time":    utils.GetCurrentTime(),
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
