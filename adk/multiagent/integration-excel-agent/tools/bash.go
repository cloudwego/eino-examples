package tools

import (
	"context"
	"encoding/json"

	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

var (
	bashToolInfo = &schema.ToolInfo{
		Name: "Bash",
		Desc: `Run commands in a bash shell
* When invoking this tool, the contents of the \"command\" parameter does NOT need to be XML-escaped.
* You don't have access to the internet via this tool.
* You do have access to a mirror of common linux and python packages via apt and pip.
* State is persistent across command calls and discussions with the user.
* To inspect a particular line range of a file, e.g. lines 10-25, try 'sed -n 10,25p /path/to/the/file'.
* Please avoid commands that may produce a very large amount of output.
* Please run long lived commands in the background, e.g. 'sleep 10 &' or start a server in the background.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"command": {
				Type:     "string",
				Desc:     "The command to execute",
				Required: true,
			},
		}),
	}
)

func NewBashTool(op commandline.Operator) tool.InvokableTool {
	return &bashTool{op: op}
}

type bashTool struct {
	op commandline.Operator
}

func (b *bashTool) Info(_ context.Context) (*schema.ToolInfo, error) {
	return bashToolInfo, nil
}

type shellInput struct {
	Command string `json:"command"`
}

func (b *bashTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	input := &shellInput{}
	err := json.Unmarshal([]byte(argumentsInJSON), input)
	if err != nil {
		return "", err
	}
	if len(input.Command) == 0 {
		return "command cannot be empty", nil
	}
	o := tool.GetImplSpecificOptions(&options{b.op}, opts...)
	return o.op.RunCommand(ctx, input.Command)
}
