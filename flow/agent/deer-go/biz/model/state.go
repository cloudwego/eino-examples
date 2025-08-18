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
	"context"
	"encoding/json"
	
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func init() {
	err := compose.RegisterSerializableType[State]("DeerState")
	if err != nil {
		panic(err)
	}
}

type State struct {
	// 用户输入的信息
	Messages []*schema.Message `json:"messages,omitempty"`

	// 子图共享变量
	Goto                           string `json:"goto,omitempty"`
	CurrentPlan                    *Plan  `json:"current_plan,omitempty"`
	Locale                         string `json:"locale,omitempty"`
	PlanIterations                 int    `json:"plan_iterations,omitempty"`
	BackgroundInvestigationResults string `json:"background_investigation_results"`
	InterruptFeedback              string `json:"interrupt_feedback,omitempty"`

	// 全局配置变量
	MaxPlanIterations             int  `json:"max_plan_iterations,omitempty"`
	MaxStepNum                    int  `json:"max_step_num,omitempty"`
	AutoAcceptedPlan              bool `json:"auto_accepted_plan"`
	EnableBackgroundInvestigation bool `json:"enable_background_investigation"`
}

func (s *State) MarshalJSON() ([]byte, error) {
	b, err := json.Marshal(*s)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func (s *State) UnmarshalJSON(b []byte) error {
	type Alias State
	var tmp Alias
	if err := json.Unmarshal(b, &tmp); err != nil {
		return err
	}
	*s = State(tmp)
	return nil
}

// DeerCheckPoint DeerGo的全局状态存储点，
// 实现CheckPointStore接口，用checkPointID进行索引
// 此处粗略使用map实现，工程上可以用工业存储组件实现
type DeerCheckPoint struct {
	buf map[string][]byte
}

func (dc *DeerCheckPoint) Get(ctx context.Context, checkPointID string) ([]byte, bool, error) {
	data, ok := dc.buf[checkPointID]
	return data, ok, nil
}

func (dc *DeerCheckPoint) Set(ctx context.Context, checkPointID string, checkPoint []byte) error {
	dc.buf[checkPointID] = checkPoint
	return nil
}

// 创建一个全局状态存储点实例并返回
var deerCheckPoint = DeerCheckPoint{
	buf: make(map[string][]byte),
}

func NewDeerCheckPoint(ctx context.Context) compose.CheckPointStore {
	return &deerCheckPoint
}
