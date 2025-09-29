package generic

import (
	"fmt"
)

type FullPlan struct {
	TaskID              int           `json:"task_id,omitempty"`
	Status              PlanStatus    `json:"status,omitempty"`
	AgentName           string        `json:"agent_name,omitempty"`
	Desc                string        `json:"desc,omitempty"`
	ExecResult          *SubmitResult `json:"exec_result,omitempty"`
	EvaluatorExecResult *SubmitResult `json:"evaluator_exec_result,omitempty"`
}

type PlanStatus string

const (
	PlanStatusTodo    PlanStatus = "todo"
	PlanStatusDoing   PlanStatus = "doing"
	PlanStatusDone    PlanStatus = "done"
	PlanStatusFailed  PlanStatus = "failed"
	PlanStatusSkipped PlanStatus = "skipped"
)

var (
	PlanStatusMapping = map[PlanStatus]string{
		PlanStatusTodo:    "待执行",
		PlanStatusDoing:   "执行中",
		PlanStatusDone:    "已完成",
		PlanStatusFailed:  "执行失败",
		PlanStatusSkipped: "已跳过",
	}
)

func (p *FullPlan) String() string {
	status, ok := PlanStatusMapping[p.Status]
	if !ok {
		status = string(p.Status)
	}
	res := fmt.Sprintf("%d. **[%s]** %s", p.TaskID, status, p.Desc)
	if p.Status == PlanStatusFailed && p.EvaluatorExecResult != nil {
		if p.EvaluatorExecResult.IsSuccess != nil {
			var isSuccess string
			if p.EvaluatorExecResult.IsSuccess != nil && *p.EvaluatorExecResult.IsSuccess {
				isSuccess = "执行成功"
			} else {
				isSuccess = "执行失败"
			}
			res += fmt.Sprintf("\n	- **结果校验**：%s，%s", isSuccess, p.EvaluatorExecResult.Result)
		} else {
			res += fmt.Sprintf("\n	- **结果校验**：%s", p.EvaluatorExecResult.Result)
		}
	}
	if p.ExecResult != nil {
		res += fmt.Sprintf("\n%s", p.ExecResult.String())
	}
	return res
}

func (p *FullPlan) PlanString(n int) string {
	if p.Status != PlanStatusDoing && p.Status != PlanStatusTodo {
		return fmt.Sprintf("- [x] %d. %s", n, p.Desc)
	}
	return fmt.Sprintf("- [ ] %d. %s", n, p.Desc)
}

func FullPlan2String(plan []*FullPlan) string {
	var planStr = "### 任务计划\n"
	for i, p := range plan {
		planStr += p.PlanString(i+1) + "\n"
		if p.Status == PlanStatusFailed && p.EvaluatorExecResult != nil {
			planStr += fmt.Sprintf("	- 执行失败，%s\n", p.EvaluatorExecResult.Result)
		}
	}
	return planStr
}

func Write2PlanMD(taskID string, plan []*FullPlan) error {
	planStr := FullPlan2String(plan)
	filename := "plan.md"
	// TODO: write local
	_ = planStr
	_ = filename
	_ = taskID
	// err := sandbox_fs.WriteFile2Sandbox(ctx, taskID, filename, []byte(planStr))
	// if err != nil {
	// 	logs.CtxError(ctx, "[tools.Write2PlanMD] WriteToFile error: %v", err)
	// 	return err
	// }

	return nil
}
