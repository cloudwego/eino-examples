package eino_agent

import (
	"context"

	"github.com/cloudwego/eino-examples/agent/tool/einotool"
	"github.com/cloudwego/eino-examples/agent/tool/gitclone"
	"github.com/cloudwego/eino-examples/agent/tool/open"
	"github.com/cloudwego/eino-examples/agent/tool/todo"
	"github.com/cloudwego/eino-ext/components/tool/duckduckgo"
	"github.com/cloudwego/eino/components/tool"
)

func defaultDDGSearchConfig(ctx context.Context) (*duckduckgo.Config, error) {
	config := &duckduckgo.Config{}
	return config, nil
}

func NewDDGSearch(ctx context.Context, config *duckduckgo.Config) (tn tool.BaseTool, err error) {
	if config == nil {
		config, err = defaultDDGSearchConfig(ctx)
		if err != nil {
			return nil, err
		}
	}
	tn, err = duckduckgo.NewTool(ctx, config)
	if err != nil {
		return nil, err
	}
	return tn, nil
}

func NewOpenFileTool(ctx context.Context) (tn tool.BaseTool, err error) {
	return open.NewOpenFileTool(ctx, nil)
}

func NewGitCloneFile(ctx context.Context) (tn tool.BaseTool, err error) {
	return gitclone.NewGitCloneFile(ctx, nil)
}

func NewEinoAssistantTool(ctx context.Context) (tn tool.BaseTool, err error) {
	return einotool.NewEinoAssistantTool(ctx, nil)
}

func NewTodoTool(ctx context.Context) (tn tool.BaseTool, err error) {
	return todo.NewTodoTool(ctx, nil)
}
