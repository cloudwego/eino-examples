package agents

import (
	"encoding/json"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino/schema"
)

type Step struct {
	Desc string `json:"desc"`
}

type Plan struct {
	Steps []Step `json:"steps"`
}

func (p *Plan) FirstStep() string {
	if len(p.Steps) == 0 {
		return ""
	}
	stepStr, _ := sonic.MarshalString(p.Steps[0])
	return stepStr
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

var PlanToolInfo = &schema.ToolInfo{
	Name: "create_plan",
	Desc: "Generates a structured, step-by-step execution plan to solve a given complex task. Each step in the plan must be assigned to a specialized agent and must have a clear, actionable description.",
	ParamsOneOf: schema.NewParamsOneOfByParams(
		map[string]*schema.ParameterInfo{
			"steps": {
				Type: schema.Array,
				ElemInfo: &schema.ParameterInfo{
					Type: schema.Object,
					SubParams: map[string]*schema.ParameterInfo{
						"index": {
							Type:     schema.Integer,
							Desc:     "The sequential number of this step in the overall plan. **Must start from 1 and increment by exactly 1 for each subsequent step.**",
							Required: true,
						},
						"agent_name": {
							Type:     schema.String,
							Desc:     "The name of the agent responsible for executing this specific step. **Must be one of the following exact values: 'GeneralAgent', 'CodeAgent', 'ReportAgent'.**",
							Enum:     []string{"GeneralAgent", "CodeAgent", "ReportAgent"},
							Required: true,
						},
						"desc": {
							Type:     schema.String,
							Desc:     "A clear, concise, and actionable description of the specific task to be performed in this step. It should be a direct instruction for the assigned agent.",
							Required: true,
						},
					},
				},
				Desc:     "different steps to follow, should be in sorted order",
				Required: true,
			},
		},
	),
}
