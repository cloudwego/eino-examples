package agents

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"

	"github.com/cloudwego/eino-examples/adk/base/tools"
)

func NewPlanner(ctx context.Context) (adk.Agent, error) {
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "single agent",
		Description: "",
		Instruction: "you are a helpful assistant.",
		Model:       nil,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					&tools.Plan{},
					&tools.Shell{},
					adk.NewAgentTool(ctx, &Searcher{}),
					adk.NewAgentTool(ctx, &GenPDF{}),
				},
			},
		},
	})
}
