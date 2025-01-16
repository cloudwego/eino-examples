package eino_agent

import (
	"context"
)

func NewInputToQuery(ctx context.Context, input *UserMessage, opts ...any) (output string, err error) {
	panic("implement me")
}

func NewInputToHistory(ctx context.Context, input *UserMessage, opts ...any) (output map[string]any, err error) {
	panic("implement me")
}
