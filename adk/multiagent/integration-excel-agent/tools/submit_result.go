package tools

import (
	"context"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/generic"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/params"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/utils"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

var (
	SubmitResultReturnDirectly = map[string]bool{
		"SubmitResult": true,
	}

	toolSubmitResultInfo = &schema.ToolInfo{
		Name: "SubmitResult",
		Desc: `The tool used for submitting task results, with parameters including the task execution outcome as well as the file paths and descriptions of the files generated during task execution.

- Note: Do not read and submit the original content of the files generated during task execution; instead, submit the file paths and descriptions.
- Intermediate files must be specified as individual files, not folders.
- It is necessary to make an overall assessment of the task’s completion status, such as whether the task is finished, failed, or requires re-execution.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"is_success": {
				Type: schema.Boolean,
				Desc: "success or not，true/false",
			},
			"result": {
				Type: schema.String,
				Desc: "Task execution result probability, and data analysis (optional)",
			},
			"files": {
				Type: schema.Array,
				ElemInfo: &schema.ParameterInfo{
					Desc: "Absolute paths and descriptions of files generated during task execution, by default excluding Python scripts.",
					Type: schema.Object,
					SubParams: map[string]*schema.ParameterInfo{
						"path": {
							Desc: "Absolute path of current file",
							Type: schema.String,
						},
						"desc": {
							Desc: "Description of the file, and an overview of the file's data content (optional).",
							Type: schema.String,
						},
					},
				},
			},
		}),
	}
)

func NewSubmitResultTool() tool.InvokableTool {
	return &SubmitResult{}
}

type SubmitResult struct{}

func (t *SubmitResult) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return toolSubmitResultInfo, nil
}

func (t *SubmitResult) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	args := &generic.SubmitResult{}
	if err := sonic.Unmarshal([]byte(argumentsInJSON), args); err != nil {
		return "", err
	}
	args.IsSuccess = nil
	params.SetCurrentAgentResult(ctx, args)
	return utils.ToJSONString(args), nil
}
