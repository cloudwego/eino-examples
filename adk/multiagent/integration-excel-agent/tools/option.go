package tools

import (
	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/components/tool"
)

type options struct {
	op commandline.Operator
}

func WithOperator(op commandline.Operator) tool.Option {
	return tool.WrapImplSpecificOptFn(func(opt *options) {
		opt.op = op
	})
}
