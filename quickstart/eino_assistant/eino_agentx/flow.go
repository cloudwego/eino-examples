package eino_agent

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
)

func defaultEinoAgentConfig(ctx context.Context) (*react.AgentConfig, error) {
	config := &react.AgentConfig{
		MaxStep:            25,
		ToolReturnDirectly: map[string]struct{}{},
	}
	configOfChatModel, err := defaultArkChatModelConfig(ctx)
	if err != nil {
		return nil, err
	}
	ArkChatModel, err := NewArkChatModel(ctx, configOfChatModel)
	if err != nil {
		return nil, err
	}
	config.Model = ArkChatModel
	DDGSearch, err := NewDDGSearch(ctx, nil)
	if err != nil {
		return nil, err
	}

	OpenFileTool, err := NewOpenFileTool(ctx)
	if err != nil {
		return nil, err
	}

	GitCloneFile, err := NewGitCloneFile(ctx)
	if err != nil {
		return nil, err
	}

	EinoAssistantTool, err := NewEinoAssistantTool(ctx)
	if err != nil {
		return nil, err
	}

	TodoTool, err := NewTodoTool(ctx)
	if err != nil {
		return nil, err
	}

	config.ToolsConfig.Tools = []tool.BaseTool{DDGSearch, OpenFileTool, GitCloneFile, EinoAssistantTool, TodoTool}

	return config, nil
}

func NewEinoAgent(ctx context.Context, config *react.AgentConfig) (lb *compose.Lambda, err error) {
	if config == nil {
		config, err = defaultEinoAgentConfig(ctx)
		if err != nil {
			return nil, err
		}
	}

	agent, err := react.NewAgent(ctx, config)
	if err != nil {
		return nil, err
	}

	lb, err = compose.AnyLambda(agent.Generate, agent.Stream, nil, nil)
	if err != nil {
		return nil, err
	}

	return lb, nil
}
