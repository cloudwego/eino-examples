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
	"encoding/json"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino/schema"
)

type Step struct {
	Index int    `json:"index"`
	Desc  string `json:"desc"`
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
