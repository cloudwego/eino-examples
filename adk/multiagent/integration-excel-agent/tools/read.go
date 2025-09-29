package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

var (
	readFileToolInfo = &schema.ToolInfo{
		Name: "ReadFile",
		Desc: `This tool is used for reading file content, with parameters including the file path, starting line, and the number of lines to read. Content will be truncated if it is too long.  
For xls and xlsx files, each sheet's information will be returned sequentially upon a single call. If multiple sheets' information is needed, only one call is required. The call will return the headers, merged cell information, and the first n_rows of data for each sheet.`,
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
	if input.StartRow == 0 {
		input.StartRow = 1
	}
	if input.NRows == 0 {
		input.NRows = 20
	}
	o := tool.GetImplSpecificOptions(&options{op: r.op})
	fileContent, err := o.op.ReadFile(ctx, input.Path)
	if err != nil {
		return "", err
	}
	fileName := filepath.Base(filepath.Clean(input.Path))

	tmpDir, err := os.MkdirTemp("", "process_dir")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	tmpFile := filepath.Join(tmpDir, fileName)
	err = os.WriteFile(tmpFile, []byte(fileContent), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write temp file: %v\n", err)
	}

	cmd := exec.Command("python", os.Getenv("tool_preview_path"), tmpFile, strconv.Itoa(input.StartRow), strconv.Itoa(input.NRows))
	buf := bytes.NewBuffer(nil)
	cmd.Stdout = buf
	cmd.Stderr = buf
	err = cmd.Run()
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
