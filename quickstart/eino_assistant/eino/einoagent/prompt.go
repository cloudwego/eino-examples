package eino_agent

import (
	"context"

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

type ChatTemplateConfig struct {
	FormatType schema.FormatType
	Templates  []schema.MessagesTemplate
}

func defaultChatTemplateConfig(ctx context.Context) (*ChatTemplateConfig, error) {
	config := &ChatTemplateConfig{
		FormatType: schema.FormatType(1)}
	return config, nil
}

func NewChatTemplate(ctx context.Context, config *ChatTemplateConfig) (ctp prompt.ChatTemplate, err error) {
	if config == nil {
		config, err = defaultChatTemplateConfig(ctx)
		if err != nil {
			return nil, err
		}
	}
	ctp = prompt.FromMessages(config.FormatType, config.Templates...)
	return ctp, nil
}
