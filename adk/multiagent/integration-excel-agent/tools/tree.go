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
	"strings"

	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/utils"
	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

var (
	treeToolInfo = &schema.ToolInfo{
		Name: "tree",
		Desc: "This tool is used to view the directory tree structure; the parameter is the path to be viewed, and it returns the complete directory tree structure under that path.",
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type:     schema.String,
				Desc:     "absolute path",
				Required: true,
			},
		}),
	}
)

func NewTreeTool(op commandline.Operator) tool.InvokableTool {
	return &tree{op: op}
}

type tree struct {
	op commandline.Operator
}

func (t *tree) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return treeToolInfo, nil
}

type treeInput struct {
	Path string `json:"path"`
}

func (t *tree) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	input := &treeInput{}

	err := json.Unmarshal([]byte(argumentsInJSON), input)
	if err != nil {
		return "", err
	}
	if len(input.Path) == 0 {
		return "path can not be empty", nil
	}
	o := tool.GetImplSpecificOptions(&options{t.op}, opts...)
	output, err := o.op.RunCommand(ctx, []string{"find", input.Path})
	if err != nil {
		if strings.HasPrefix(err.Error(), "internal error") {
			return err.Error(), nil
		}
		return "", err
	}
	return utils.FormatCommandOutput(output), nil
}
