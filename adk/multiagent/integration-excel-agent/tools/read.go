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
	readFileToolInfo = &schema.ToolInfo{
		Name: "ReadFile",
		Desc: `This tool is used for reading file content, with parameters including the file path, starting line, and the number of lines to read. Content will be truncated if it is too long.  
For xlsx files, each sheet's information will be returned sequentially upon a single call. If multiple sheets' information is needed, only one call is required. The call will return the headers, merged cell information, and the first n_rows of data for each sheet.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type:     schema.String,
				Desc:     "file absolute path",
				Required: true,
			},
			"start_row": {
				Type: schema.Integer,
				Desc: "The starting line defaults to 1, meaning reading begins from the first line.",
			},
			"n_rows": {
				Type: schema.Integer,
				Desc: "Number of rows to read, -1 means reading from start_row to the end of the file, default is 20 rows. For xlsx, xls, and xlsm files, the default is 10 rows.",
			},
		}),
	}
)

func NewReadFileTool(op commandline.Operator) tool.InvokableTool {
	return &readFile{op: op}
}

type readFile struct {
	op commandline.Operator
}

func (r *readFile) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return readFileToolInfo, nil
}

type readFileInput struct {
	Path     string `json:"path"`
	StartRow int    `json:"start_row"`
	NRows    int    `json:"n_rows"`
}

func (r *readFile) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	input := &readFileInput{}
	err := json.Unmarshal([]byte(argumentsInJSON), input)
	if err != nil {
		return "", err
	}
	if input.Path == "" {
		return "path can not be empty", nil
	}
	if input.StartRow <= 0 {
		input.StartRow = 1
	}
	if input.NRows <= 0 {
		input.NRows = 20
	}
	o := tool.GetImplSpecificOptions(&options{op: r.op})
	cmd := fmt.Sprintf("python3 -c \"import sys; lines = (line for idx, line in enumerate(open(sys.argv[1], encoding='utf-8')) if %d <= idx < %d); print(''.join(lines))\" %s",
		input.StartRow, input.StartRow+input.NRows, input.Path)
	content, err := o.op.RunCommand(ctx, cmd)
	if err != nil {
		return "", err
	}

	return content, nil
}
