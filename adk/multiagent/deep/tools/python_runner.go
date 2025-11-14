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
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cloudwego/eino-examples/adk/multiagent/deep/utils"
	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	jsoniter "github.com/json-iterator/go"

	"github.com/cloudwego/eino-examples/adk/multiagent/deep/params"
)

var (
	toolPythonRunnerInfo = &schema.ToolInfo{
		Name: "python_runner",
		Desc: `Write Python code to a file and execute it, and return the execution result. 
Code would be overwritten to the same file when this tool is called multiple times.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"code": {
				Type: "string",
				Desc: "Python code to be executed. " +
					"The code MUST be enclosed in a Markdown code block starting with ```python and ending with ```. " +
					"CRITICAL: The code within the block must be a raw, multi-line string with real newlines. " +
					"It MUST NOT be a JSON-escaped string containing literal '\\n' or '\\\"' sequences. " +
					"The code must be ready for direct execution without any unescaping. " +
					"Do not generate code comments.",
				Required: true,
			},
		}),
	}
)

func NewPythonRunnerTool(op commandline.Operator) tool.InvokableTool {
	return &pythonRunnerTool{op: op}
}

type pythonRunnerTool struct {
	op commandline.Operator
}

func (p *pythonRunnerTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	return toolPythonRunnerInfo, nil
}

func (p *pythonRunnerTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	code := tryExtractCodeSnippet(argumentsInJSON)
	if code == "" {
		return "", fmt.Errorf("python code is empty, original=%s", argumentsInJSON)
	}
	taskID := params.MustGetContextParams[string](ctx, params.TaskIDKey)
	wd, ok := params.GetTypedContextParams[string](ctx, params.WorkDirSessionKey)
	if !ok {
		return "", fmt.Errorf("work dir not found")
	}
	filePath := filepath.Join(wd, fmt.Sprintf("%s.py", taskID))

	if err := p.op.WriteFile(ctx, filePath, code); err != nil {
		return fmt.Sprintf("failed to create python file %s: %v", filePath, err), nil
	}

	pyExecutablePath := os.Getenv("EXCEL_AGENT_PYTHON_EXECUTABLE_PATH")
	if pyExecutablePath == "" {
		pyExecutablePath = "python"
	}
	result, err := p.op.RunCommand(ctx, []string{pyExecutablePath, filePath})
	if err != nil {
		if strings.HasPrefix(err.Error(), "internal error") {
			return err.Error(), nil
		}
		return "", fmt.Errorf("execute error: %w", err)
	}
	return utils.FormatCommandOutput(result), nil
}

func tryExtractCodeSnippet(res string) string {
	var input struct {
		Code string `json:"code"`
	}

	var codeToProcess string
	err := jsoniter.Unmarshal([]byte(res), &input)
	if err != nil {
		codeToProcess = res
	} else {
		codeToProcess = input.Code
	}

	rawCode := extractCodeSnippet(codeToProcess)
	unescapedCode := strings.NewReplacer(
		"\\\\", "\\",
		"\\n", "\n",
		"\\r", "\r",
		"\\t", "\t",
		"\\\"", "\"",
		"\\'", "'",
	).Replace(rawCode)

	return unescapedCode
}

func extractCodeSnippet(res string) string {
	codePattern := regexp.MustCompile("(?s)```python\\s*(.*?)\\s*```")
	codeMatch := codePattern.FindStringSubmatch(res)

	if len(codeMatch) > 1 {
		return strings.TrimSpace(codeMatch[1])
	} else {
		fallbackPattern := regexp.MustCompile("(?s)```\\s*(.*?)\\s*```")
		fallbackMatch := fallbackPattern.FindStringSubmatch(res)
		if len(fallbackMatch) > 1 {
			return strings.TrimSpace(fallbackMatch[1])
		} else {
			return strings.TrimSpace(res)
		}
	}
}
