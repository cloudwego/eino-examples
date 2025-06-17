package single

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/demo/base/agents"
	"github.com/cloudwego/eino/adk/demo/base/tools"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

func NewAgent(ctx context.Context) (adk.Agent, error) {
	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "single agent",
		Description: "",
		Instruction: "you are a helpful assistant.",
		Model:       nil,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					&tools.Shell{},
					adk.NewAgentTool(ctx, &agents.Searcher{}),
					adk.NewAgentTool(ctx, &agents.GenPDF{}),
				},
			},
		},
	})
}
