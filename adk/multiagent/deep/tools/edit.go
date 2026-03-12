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

	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

var (
	editFileToolInfo = &schema.ToolInfo{
		Name: "edit_file",
		Desc: `This is a tool for editing file, with parameters including the file path and the content to be edited.
During task processing, if there is a need to create a file or overwrite file content, this tool can be used.

Notice:
- If the file does not exist, this tool creates it with permissions perm (0666); otherwise it will truncates it before writing, without changing permissions.
- When using this tool, be sure that the file content is the complete full text; otherwise, it may cause loss or errors in the file content.
- Only supports writing to text file s; writing to xls/xlsx files is not supported.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"path": {
				Type:     schema.String,
				Desc:     "file absolute path",
				Required: true,
			},
			"content": {
				Type:     schema.String,
				Desc:     "file content",
				Required: true,
			},
		}),
	}
)

func NewEditFileTool(op commandline.Operator) tool.InvokableTool {
	return &editFile{op: op}
}

type editFile struct {
	op commandline.Operator
}

func (e *editFile) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return editFileToolInfo, nil
}

type editFileInput struct {
	Path    string `name:"path"`
	Content string `name:"content"`
}

func (e *editFile) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	input := &editFileInput{}
	err := json.Unmarshal([]byte(argumentsInJSON), input)
	if err != nil {
		return "", err
	}
	if len(input.Path) == 0 {
		return "path can not be empty", nil
	}
	o := tool.GetImplSpecificOptions(&options{op: e.op}, opts...)
	err = o.op.WriteFile(ctx, input.Path, input.Content)
	if err != nil {
		return err.Error(), nil
	}
	return "edit file success", nil
}
