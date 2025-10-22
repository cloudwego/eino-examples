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

package generic

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/cloudwego/eino-ext/components/tool/commandline"
)

type FullPlan struct {
	TaskID     int           `json:"task_id,omitempty"`
	Status     PlanStatus    `json:"status,omitempty"`
	AgentName  string        `json:"agent_name,omitempty"`
	Desc       string        `json:"desc,omitempty"`
	ExecResult *SubmitResult `json:"exec_result,omitempty"`
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
	}
	return planStr
}

func Write2PlanMD(ctx context.Context, op commandline.Operator, wd string, plan []*FullPlan) error {
	planStr := FullPlan2String(plan)
	filePath := filepath.Join(wd, "plan.md")
	return op.WriteFile(ctx, filePath, planStr)
}
