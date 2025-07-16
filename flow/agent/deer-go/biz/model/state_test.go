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

package model

import (
	"fmt"
	"testing"
)

func TestState(t *testing.T) {
	state := State{
		Messages:          nil,
		Goto:              "",
		PlanIterations:    0,
		MaxPlanIterations: 0,
		MaxStepNum:        0,
		CurrentPlan: &Plan{
			Thought: "asdas",
			Steps: []Step{
				{
					Title: "asdas",
				},
			},
		},
		Locale: "",
		//Server:                         nil,
		InterruptFeedback:              "",
		AutoAcceptedPlan:               false,
		EnableBackgroundInvestigation:  false,
		BackgroundInvestigationResults: "",
	}
	bt, err := state.MarshalJSON()
	//bt, err := json.Marshal(state)
	if err != nil {
		fmt.Println("编码失败,错误原因: ", err)
		return
	}
	fmt.Println(string(bt))
}
