package adk

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

var plannerPrompt = `
当前的时间是：{{current_time}}
`

type Step struct {
	TaskID    int    `json:"task_id"`
	Status    string `json:"status"`
	AgentName string `json:"agent_name"`
	Desc      string `json:"desc"`
}

type Plan []Step

func (p *Plan) FirstStep() string {
	if len(*p) == 0 {
		return ""
	}
	return fmt.Sprintf(`{"task_id": %d, "status": "%s", "agent_name": "%s", "desc": "%s"}`, (*p)[0].TaskID, (*p)[0].Status, (*p)[0].AgentName, (*p)[0].Desc)
}

func (p *Plan) MarshalJSON() ([]byte, error) {
	type Alias Plan
	return json.Marshal((*Alias)(p))
}

func (p *Plan) UnmarshalJSON(bytes []byte) error {
	type Alias Plan
	a := (*Alias)(p)
	return json.Unmarshal(bytes, a)
}

func plannerInputGen(ctx context.Context, in *prebuilt.PlannerInput) ([]adk.Message, error) {
	ct, ok := adk.GetSessionValue(ctx, CurrentTimeSessionKey)
	if !ok {
		return nil, fmt.Errorf("cannot find current time in session")
	}
	fp, ok := adk.GetSessionValue(ctx, FilePreviewSessionKey)
	if !ok {
		return nil, fmt.Errorf("cannot find preview file in session")
	}

	inputTpl := []schema.MessagesTemplate{schema.SystemMessage(plannerPrompt)}
	if in != nil {
		for _, inMsg := range in.Input {
			inputTpl = append(inputTpl, inMsg)
		}
	}

	return prompt.FromMessages(
		schema.FString,
		inputTpl...,
	).Format(ctx, map[string]any{
		"current_time": ct,
		"headers":      fp,
	})
}

var PlanToolInfo = &schema.ToolInfo{
	Name: "Plan",
	Desc: "Plan with a list of steps to execute in order. Each step should be clear, actionable, and arranged in a logical sequence. The output will be used to guide the execution process.",
	ParamsOneOf: schema.NewParamsOneOfByParams(
		map[string]*schema.ParameterInfo{
			"steps": {
				Type: schema.Array,
				ElemInfo: &schema.ParameterInfo{
					Type: schema.String,
				},
				Desc:     "different steps to follow, should be in sorted order",
				Required: true,
			},
		},
	),
}
