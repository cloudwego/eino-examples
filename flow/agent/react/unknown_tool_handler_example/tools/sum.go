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
	"encoding/json"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

type SumTool struct{}

func SumToolFn() tool.InvokableTool { return &SumTool{} }

func (t *SumTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return &schema.ToolInfo{
		Name: "sum",
		Desc: "Add two integers",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"a": {Type: "number", Desc: "first operand", Required: true},
			"b": {Type: "number", Desc: "second operand", Required: true},
		}),
	}, nil
}

func (t *SumTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	var p struct {
		A int `json:"a"`
		B int `json:"b"`
	}
	if err := json.Unmarshal([]byte(argumentsInJSON), &p); err != nil {
		return "", err
	}
	res := map[string]int{"sum": p.A + p.B}
	b, err := json.Marshal(res)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
