package einoagent

import (
	"context"

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

	tools, err := GetTools(ctx)
	if err != nil {
		return nil, err
	}

	config.ToolsConfig.Tools = tools
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
