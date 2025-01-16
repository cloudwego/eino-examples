package eino_agent

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/model"
)

func defaultArkChatModelConfig(ctx context.Context) (*ark.ChatModelConfig, error) {
	config := &ark.ChatModelConfig{
		Stop:      []string{},
		LogitBias: map[string]int{}}
	return config, nil
}

func NewArkChatModel(ctx context.Context, config *ark.ChatModelConfig) (cm model.ChatModel, err error) {
	if config == nil {
		config, err = defaultArkChatModelConfig(ctx)
		if err != nil {
			return nil, err
		}
	}
	cm, err = ark.NewChatModel(ctx, config)
	if err != nil {
		return nil, err
	}
	return cm, nil
}
