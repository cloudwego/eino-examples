package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

var (
	treeToolInfo = &schema.ToolInfo{
		Name: "Tree",
		Desc: "This tool is used to view the directory tree structure; the parameter is the path to be viewed, and it returns the complete directory tree structure under that path.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type:     schema.String,
				Desc:     "absolute path",
				Required: true,
			},
		}),
	}
)

func NewTreeTool(op commandline.Operator) tool.InvokableTool {
	return &tree{op: op}
}

type tree struct {
	op commandline.Operator
}

func (t *tree) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return treeToolInfo, nil
}

type treeInput struct {
	Path string `json:"path"`
}

func (t *tree) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	input := &treeInput{}

	err := json.Unmarshal([]byte(argumentsInJSON), input)
	if err != nil {
		return "", err
	}
	if len(input.Path) == 0 {
		return "path can not be empty", nil
	}
	o := tool.GetImplSpecificOptions(&options{t.op}, opts...)
	return o.op.RunCommand(ctx, fmt.Sprintf("find %s", input.Path))
}
