package executor

import (
	"context"
	"os"

	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/utils"
	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
)

func NewExecutor(ctx context.Context, operator commandline.Operator) (adk.Agent, error) {
	cm, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey:      os.Getenv("ARK_API_KEY"),
		BaseURL:     os.Getenv("ARK_BASE_URL"),
		Region:      os.Getenv("ARK_REGION"),
		Model:       os.Getenv("ARK_MODEL"),
		MaxTokens:   utils.PtrOf(4096),
		Temperature: utils.PtrOf(float32(0)),
		TopP:        utils.PtrOf(float32(0)),
	})
	if err != nil {
		return nil, err
	}

	ca, err := newCodeAgent(ctx, operator)
	if err != nil {
		return nil, err
	}

	a, err := planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
		Model: cm,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					adk.NewAgentTool(ctx, ca),
				},
			},
		},
		MaxIterations: 20,
	})
	if err != nil {
		return nil, err
	}

	return a, nil
}
