package einoagent

import (
	"context"

	"github.com/cloudwego/eino-examples/quickstart/eino_assistant/pkg/tool/einotool"
	"github.com/cloudwego/eino-examples/quickstart/eino_assistant/pkg/tool/gitclone"
	"github.com/cloudwego/eino-examples/quickstart/eino_assistant/pkg/tool/open"
	"github.com/cloudwego/eino-examples/quickstart/eino_assistant/pkg/tool/task"
	"github.com/cloudwego/eino-ext/components/tool/duckduckgo"
	"github.com/cloudwego/eino/components/tool"
)

func GetTools(ctx context.Context) ([]tool.BaseTool, error) {
	einoAssistantTool, err := NewEinoAssistantTool(ctx)
	if err != nil {
		return nil, err
	}

	toolTask, err := NewTaskTool(ctx)
	if err != nil {
		return nil, err
	}

	toolOpen, err := NewOpenFileTool(ctx)
	if err != nil {
		return nil, err
	}

	toolGitClone, err := NewGitCloneFile(ctx)
	if err != nil {
		return nil, err
	}

	toolDDGSearch, err := NewDDGSearch(ctx, nil)
	if err != nil {
		return nil, err
	}

	return []tool.BaseTool{
		einoAssistantTool,
		toolTask,
		toolOpen,
		toolGitClone,
		toolDDGSearch,
	}, nil
}

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

func NewTaskTool(ctx context.Context) (tn tool.BaseTool, err error) {
	return task.NewTaskTool(ctx, nil)
}
