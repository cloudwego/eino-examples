package eino_agent

import (
	"context"
	"time"

	"github.com/cloudwego/eino/schema"
)

func NewDocumentsConvert(ctx context.Context, input []*schema.Document) (output map[string]any, err error) {
	if len(input) == 0 {
		return map[string]any{
			"documents": "",
		}, nil
	}

	outputStr := "## retrieved documents\n"
	for _, doc := range input {
		outputStr += doc.String() + "\n"
	}
	return map[string]any{
		"documents": outputStr,
	}, nil
}

func NewRetrieverInputConvert(ctx context.Context, input *UserMessage) (output string, err error) {
	if input == nil {
		return "", nil
	}
	return input.Query, nil
}

func NewInputConvertor(ctx context.Context, input *UserMessage) (output map[string]any, err error) {
	if input == nil {
		return map[string]any{}, nil
	}

	return map[string]any{
		"content": input.Query,
		"history": input.History,
		"date":    time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}
