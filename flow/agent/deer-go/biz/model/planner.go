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

// StepType 定义步骤类型的枚举
type StepType string

const (
	Research   StepType = "research"
	Processing StepType = "processing"
)

// Step 定义单个步骤的结构体
type Step struct {
	NeedWebSearch bool     `json:"need_web_search" validate:"required"`
	Title         string   `json:"title" validate:"required"`
	Description   string   `json:"description" validate:"required"`
	StepType      StepType `json:"step_type" validate:"required"`
	ExecutionRes  *string  `json:"execution_res,omitempty"`
}

// Plan 定义计划的结构体
type Plan struct {
	Locale           string `json:"locale" validate:"required"`
	HasEnoughContext bool   `json:"has_enough_context" validate:"required"`
	Thought          string `json:"thought" validate:"required"`
	Title            string `json:"title" validate:"required"`
	Steps            []Step `json:"steps"`
}
