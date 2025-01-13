package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/tool/duckduckgo"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"

	"github.com/cloudwego/eino-examples/agent/tool/einotool"
	"github.com/cloudwego/eino-examples/agent/tool/gitclone"
	"github.com/cloudwego/eino-examples/agent/tool/open"
	"github.com/cloudwego/eino-examples/agent/tool/todo"
)

// 1. 可联网查询
// 2. 可下载 github 项目
// 3. 可打开 web 页面
// 4. 可打开文件
// 5. eino 助手(eino 项目信息、eino 文档、eino 示例项目)
// 6. todo 工具
// func NewAgent(ctx context.Context) (compose.Runnable[*eino_agent.UserMessage, *schema.Message], error) {
// 	// 	AgentGraph: &eino_agent.AgentGraphBuildConfig{
// 	// 		EinoRetriveKeyOfRetriever: &eino_agent.EinoRetrieverConfig{
// 	// 			RedisVectorStoreConfig: redis.RedisVectorStoreConfig{
// 	// 				RedisAddr:      "127.0.0.1:6379",
// 	// 				Embedding:      embedding,
// 	// 				RedisKeyPrefix: "doc:",
// 	// 				Dimension:      1536,
// 	// 				TopK:           3,
// 	// 				MinScore:       0.1,
// 	// 			},
// 	// 		},
// 	// 		PromptTemplateKeyOfChatTemplate: &eino_agent.PromptTemplateConfig{
// 	// 			FormatType: schema.FString,
// 	// 			Templates: []schema.MessagesTemplate{
// 	// 				schema.SystemMessage("{documents}"),
// 	// 				schema.MessagesPlaceholder("history", true),
// 	// 				schema.UserMessage("{content}"),
// 	// 			},
// 	// 		},
// 	// 	},
// 	graph, err := eino_agent.BuildAgentGraph(ctx, eino_agent.BuildConfig{})
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to build agent graph: %w", err)
// 	}

// 	runner, err := graph.Compile(ctx)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to compile agent graph: %w", err)
// 	}

// 	return runner, nil
// }

// func NewAgent(ctx context.Context, persona string, modelName string, apiKey string) (*react.Agent, error) {
// 	model, err := PrepareChatModel(ctx, modelName, apiKey)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to prepare chat model: %w", err)
// 	}

// 	tools, err := PreloadTools(ctx)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to preload tools: %w", err)
// 	}

// 	reactAgent, err := react.NewAgent(ctx, &react.AgentConfig{
// 		Model: model,
// 		ToolsConfig: compose.ToolsNodeConfig{
// 			Tools: tools,
// 		},
// 		MessageModifier: react.NewPersonaModifier(persona),
// 	})
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to create react agent: %w", err)
// 	}

// 	return reactAgent, nil
// }

func PrepareChatModel(ctx context.Context, modelName string, apiKey string) (model.ChatModel, error) {
	model, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		Model:  modelName,
		APIKey: apiKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create chat model: %w", err)
	}
	return model, nil
}

func PreloadTools(ctx context.Context) ([]tool.BaseTool, error) {
	tools := []tool.BaseTool{}

	// 可打开本地文件/文件夹/web url
	openFileTool, err := open.NewOpenFileTool(ctx, &open.OpenFileToolConfig{})
	if err != nil {
		return nil, fmt.Errorf("failed to create open file tool: %w", err)
	}
	tools = append(tools, openFileTool)

	// 可下载 github 项目
	gitCloneTool, err := gitclone.NewGitCloneFile(ctx, &gitclone.GitCloneFileConfig{
		BaseDir: "./data",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create git clone tool: %w", err)
	}
	tools = append(tools, gitCloneTool)

	// 可联网查询
	ddg, err := duckduckgo.NewTool(ctx, &duckduckgo.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create duckduckgo tool: %w", err)
	}
	tools = append(tools, ddg)

	// eino 助手
	etTool, err := einotool.NewEinoAssistantTool(ctx, &einotool.EinoAssistantToolConfig{
		BaseDir: "./data/",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create eino tool: %w", err)
	}
	tools = append(tools, etTool)

	// todo 工具
	todoTool, err := todo.NewTodoTool(ctx, &todo.TodoToolConfig{
		Storage: todo.GetDefaultStorage(),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create todo tool: %w", err)
	}
	tools = append(tools, todoTool)

	return tools, nil
}

type LogCallbackConfig struct {
	Detail bool
	Debug  bool
	Writer io.Writer
}

func LogCallback(config *LogCallbackConfig) callbacks.Handler {
	if config == nil {
		config = &LogCallbackConfig{
			Detail: true,
			Writer: os.Stdout,
		}
	}
	if config.Writer == nil {
		config.Writer = os.Stdout
	}
	builder := callbacks.NewHandlerBuilder()
	builder.OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
		fmt.Fprintf(config.Writer, "[view]: start [%s:%s:%s]\n", info.Component, info.Type, info.Name)
		if config.Detail {
			var b []byte
			if config.Debug {
				b, _ = json.MarshalIndent(input, "", "  ")
			} else {
				b, _ = json.Marshal(input)
			}
			fmt.Fprintf(config.Writer, "%s\n", string(b))
		}
		return ctx
	})
	builder.OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
		fmt.Fprintf(config.Writer, "[view]: end [%s:%s:%s]\n", info.Component, info.Type, info.Name)
		return ctx
	})
	return builder.Build()
}
