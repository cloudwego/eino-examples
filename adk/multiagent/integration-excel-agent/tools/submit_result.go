/*
 * Copyright 2025 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package tools

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/generic"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/params"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/utils"
)

var (
	submitResultToolInfo = &schema.ToolInfo{
		Name: "submit_result",
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

	SubmitResultReturnDirectly = map[string]bool{
		"SubmitResult": true,
	}
)

func NewToolSubmitResult(op commandline.Operator) tool.InvokableTool {
	return &submitResultTool{op: op}
}

type submitResultTool struct {
	op commandline.Operator
}

func (t *submitResultTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return submitResultToolInfo, nil
}

func (t *submitResultTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	args := &generic.SubmitResult{}
	if err := sonic.Unmarshal([]byte(argumentsInJSON), args); err != nil {
		return "", err
	}

	plan, _ := utils.GetSessionValue[*generic.Plan](ctx, planexecute.PlanSessionKey)
	steps, _ := utils.GetSessionValue[[]planexecute.ExecutedStep](ctx, planexecute.ExecutedStepsSessionKey)

	var fullPlan []*generic.FullPlan
	for i, step := range steps {
		fullPlan = append(fullPlan, &generic.FullPlan{
			TaskID: i + 1,
			Status: generic.PlanStatusDone,
			Desc:   step.Step,
			ExecResult: &generic.SubmitResult{
				IsSuccess: utils.PtrOf(true),
				Result:    step.Result,
			},
		})
	}

	for i := len(steps); i < len(plan.Steps); i++ {
		step := plan.Steps[i]
		fullPlan = append(fullPlan, &generic.FullPlan{
			TaskID: len(fullPlan) + 1,
			Status: generic.PlanStatusSkipped,
			Desc:   step.Desc,
		})
	}

	wd, ok := params.GetTypedContextParams[string](ctx, params.WorkDirSessionKey)
	if !ok {
		return "", fmt.Errorf("work dir not found")
	}

	_ = t.op.WriteFile(ctx, filepath.Join(wd, "final_report.json"), argumentsInJSON)
	_ = generic.Write2PlanMD(ctx, t.op, wd, fullPlan)
	return utils.ToJSONString(&generic.FullPlan{AgentName: compose.END}), nil
}
