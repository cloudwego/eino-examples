package tools

import (
	"context"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/generic"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/params"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/utils"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

var (
	submitResultReplannerToolInfo = &schema.ToolInfo{
		Name: "SubmitResult",
		Desc: "When all steps are completed without obvious problems, call this tool to end the task and report the final execution results to the user.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"is_success": {
				Type: schema.Boolean,
				Desc: "success or not，true/false",
			},
			"result": {
				Type: schema.String,
				Desc: "Task execution process and result",
			},
			"files": {
				Type: schema.Array,
				ElemInfo: &schema.ParameterInfo{
					Desc: `The final file that needs to be delivered to the user (only the files that are successfully generated in the end are included, and Python scripts are not included by default unless explicitly requested by the user).
Select only the documents that can meet the original needs of users, and put the documents that best meet the needs to the first.
If there are many documents that meet the original needs of users, the report integrating these documents shall be delivered first, and the number of documents finally submitted shall be controlled within 3 as far as possible.`,
					Type: schema.Object,
					SubParams: map[string]*schema.ParameterInfo{
						"path": {
							Desc: "absolute path",
							Type: schema.String,
						},
						"desc": {
							Desc: "file content description",
							Type: schema.String,
						},
					},
				},
			},
		}),
	}
)

func NewToolSubmitResultReplanner() tool.InvokableTool {
	return &submitResultReplanner{}
}

type submitResultReplanner struct{}

func (t *submitResultReplanner) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return submitResultReplannerToolInfo, nil
}

func (t *submitResultReplanner) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	args := &generic.SubmitResult{}
	if err := sonic.Unmarshal([]byte(argumentsInJSON), args); err != nil {
		return "", err
	}
	plan := params.GetPlan(ctx)
	currentStep := params.GetCurrentStep(ctx)

	if currentStep < len(plan) {
		plan[currentStep].Status = generic.PlanStatusDone
		plan[currentStep].ExecResult = args
	}
	for i := currentStep + 1; i < len(plan); i++ {
		plan[i].Status = generic.PlanStatusSkipped
	}
	params.ResetPlanAndStep(ctx, plan, currentStep)
	_ = generic.Write2PlanMD(params.GetCozeSpaceTaskID(ctx), plan)
	return utils.ToJSONString(&generic.FullPlan{AgentName: compose.END}), nil
}
