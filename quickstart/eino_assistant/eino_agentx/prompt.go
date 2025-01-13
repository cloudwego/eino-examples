package eino_agent

import (
	"context"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

type PromptTemplateConfig struct {
	FormatType schema.FormatType
	Templates  []schema.MessagesTemplate
}

func defaultPromptTemplateConfig(ctx context.Context) (*PromptTemplateConfig, error) {
	config := &PromptTemplateConfig{
		FormatType: schema.FString,
		Templates: []schema.MessagesTemplate{
			schema.SystemMessage("{documents}"),
			schema.MessagesPlaceholder("history", true),
			schema.UserMessage("{content}"),
		},
	}
	return config, nil
}

func NewPromptTemplate(ctx context.Context, config *PromptTemplateConfig) (ct prompt.ChatTemplate, err error) {
	if config == nil {
		config, err = defaultPromptTemplateConfig(ctx)
		if err != nil {
			return nil, err
		}
	}
	ct = prompt.FromMessages(config.FormatType, config.Templates...)
	return ct, nil
}
