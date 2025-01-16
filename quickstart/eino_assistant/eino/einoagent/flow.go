package eino_agent

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
)

func defaultReactAgentConfig(ctx context.Context) (*react.AgentConfig, error) {
	config := &react.AgentConfig{
		MaxStep:            5,
		ToolReturnDirectly: map[string]struct{}{}}
	chatModelCfg11, err := defaultArkChatModelConfig(ctx)
	if err != nil {
		return nil, err
	}
	chatModelIns11, err := NewArkChatModel(ctx, chatModelCfg11)
	if err != nil {
		return nil, err
	}
	config.Model = chatModelIns11
	toolCfg21, err := defaultDuckDuckGoToolConfig(ctx)
	if err != nil {
		return nil, err
	}
	toolIns21, err := NewDuckDuckGoTool(ctx, toolCfg21)
	if err != nil {
		return nil, err
	}
	toolCfg22, err := defaultTaskToolConfig(ctx)
	if err != nil {
		return nil, err
	}
	toolIns22, err := NewTaskTool(ctx, toolCfg22)
	if err != nil {
		return nil, err
	}
	toolCfg23, err := defaultEinoToolConfig(ctx)
	if err != nil {
		return nil, err
	}
	toolIns23, err := NewEinoTool(ctx, toolCfg23)
	if err != nil {
		return nil, err
	}
	toolCfg24, err := defaultOpenURIToolConfig(ctx)
	if err != nil {
		return nil, err
	}
	toolIns24, err := NewOpenURITool(ctx, toolCfg24)
	if err != nil {
		return nil, err
	}
	toolCfg25, err := defaultGitCloneToolConfig(ctx)
	if err != nil {
		return nil, err
	}
	toolIns25, err := NewGitCloneTool(ctx, toolCfg25)
	if err != nil {
		return nil, err
	}
	config.ToolsConfig.Tools = []tool.BaseTool{toolIns21, toolIns22, toolIns23, toolIns24, toolIns25}
	return config, nil
}

func NewReactAgent(ctx context.Context, config *react.AgentConfig) (lba *compose.Lambda, err error) {
	if config == nil {
		config, err = defaultReactAgentConfig(ctx)
		if err != nil {
			return nil, err
		}
	}
	ins, err := react.NewAgent(ctx, config)
	if err != nil {
		return nil, err
	}
	lba, err = compose.AnyLambda(ins.Generate, ins.Stream, nil, nil)
	if err != nil {
		return nil, err
	}
	return lba, nil
}
