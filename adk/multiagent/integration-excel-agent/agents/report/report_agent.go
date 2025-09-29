package report

import (
	"context"
	"encoding/json"
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

func NewReportAgent(ctx context.Context, operator commandline.Operator) (adk.Agent, error) {
	cm, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey:      os.Getenv("ARK_API_KEY"),
		BaseURL:     os.Getenv("ARK_BASE_URL"),
		Region:      os.Getenv("ARK_REGION"),
		Model:       os.Getenv("ARK_MODEL"),
		Temperature: utils.PtrOf(float32(0.1)),
		MaxTokens:   utils.PtrOf(15000),
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
6.  If work is done, call SubmitResult tool to finish.
`,
		Model: cm,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					tools.NewBashTool(operator),
					tools.NewTreeTool(operator),
					tools.NewEditFileTool(operator),
					tools.NewReadFileTool(operator),
					tools.NewToolImageReader(visionModel),
					tools.NewToolSubmitResultReplanner(),
				},
			},
			ReturnDirectly: tools.SubmitResultReturnDirectly,
		},
		GenModelInput: func(ctx context.Context, instruction string, input *adk.AgentInput) ([]adk.Message, error) {
			planExecuteResult := input.Messages
			if len(input.Messages) > 0 && input.Messages[len(input.Messages)-1].Role == schema.Tool {
				fmt.Println("got submit result !!")
				planExecuteResult = []*schema.Message{input.Messages[len(input.Messages)-1]}
			}

			fp, ok := consts.GetSessionValue[string](ctx, consts.FilePathSessionKey)
			if !ok {
				return nil, fmt.Errorf("file path session key not found")
			}

			pathUrlMap, ok := consts.GetSessionValue[map[string]string](ctx, consts.PathUrlMapSessionKey)
			if !ok {
				return nil, fmt.Errorf("path url map key not found")
			}
			var pathUrlMapStr string
			for k, v := range pathUrlMap {
				pathUrlMapStr += fmt.Sprintf("image_name:%s,url:%s\n", k, v)
			}

			plan, ok := consts.GetSessionValue[*agents.Plan](ctx, planexecute.PlanSessionKey)
			if !ok {
				return nil, fmt.Errorf("plan not found")
			}

			planStr, err := json.MarshalIndent(plan, "", "\t")
			if err != nil {
				return nil, err
			}

			wd, ok := consts.GetSessionValue[string](ctx, consts.WorkDirSessionKey)
			if !ok {
				return nil, fmt.Errorf("work dir not found")
			}

			tpl := prompt.FromMessages(schema.Jinja2,
				schema.SystemMessage(instruction),
				schema.UserMessage(`
User Query: {{ user_query }}
Input File Path: {{ file_path }}
Working Directory: {{ work_dir }}
Current Time: {{ current_time }}

**Plan Details:**
{{ plan }}

**Image Mappings:**
{{ path_url_map }}
`))

			msgs, err := tpl.Format(ctx, map[string]any{
				"file_path":    fp,
				"work_dir":     wd,
				"user_query":   utils.FormatInput(planExecuteResult),
				"plan":         planStr,
				"path_url_map": pathUrlMapStr,
				"current_time": utils.GetCurrentTime(),
			})
			if err != nil {
				return nil, err
			}

			return msgs, nil
		},
		MaxIterations: 5,
	})
	if err != nil {
		return nil, err
	}

	return ra, nil
}
