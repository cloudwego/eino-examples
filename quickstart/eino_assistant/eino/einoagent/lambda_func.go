package einoagent

import (
	"context"
	"time"
)

func NewInputToQuery(ctx context.Context, input *UserMessage, opts ...any) (output string, err error) {
	return input.Query, nil
}

func NewInputToHistory(ctx context.Context, input *UserMessage, opts ...any) (output map[string]any, err error) {
	return map[string]any{
		"content": input.Query,
		"history": input.History,
		"date":    time.Now().Format("2006-01-02 15:04:05"),
	}, nil
}
