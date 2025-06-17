package tools

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type Shell struct{}

func (s *Shell) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name:        "Shell",
		Desc:        "",
		ParamsOneOf: nil,
	}, nil
}

func (s *Shell) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return "", nil
}

type Plan struct{}

func (p *Plan) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return nil, nil
}

func (p *Plan) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	return "", nil
}
