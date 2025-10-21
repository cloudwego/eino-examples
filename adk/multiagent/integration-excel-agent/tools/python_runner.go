package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/params"
	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
	jsoniter "github.com/json-iterator/go"
)

var (
	toolPythonRunnerInfo = &schema.ToolInfo{
		Name: "PythonRunner",
		Desc: `Write Python code to a file and execute it, and return the execution result.`,
		ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
			"code": {
				Type: "string",
				Desc: "Python code to be executed should be output using Markdown code block syntax." +
					"Starting with ```python and end with ```, enclosing the complete Python code within." +
					"Do not generate code comment.",
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
	result, err := p.op.RunCommand(ctx, fmt.Sprintf("%s %s", pyExecutablePath, filePath))
	if err != nil {
		return "", fmt.Errorf("execute error: %w", err)
	}

	return result, nil
}

func tryExtractCodeSnippet(res string) string {
	var input struct {
		Code string `json:"code"`
	}

	var rawCode string
	err := jsoniter.Unmarshal([]byte(res), &input)
	if err != nil {
		rawCode = extractCodeSnippet(res)
	} else {
		rawCode = extractCodeSnippet(input.Code)
	}

	return strings.NewReplacer(
		"\\\\", "\\",
		"\\n", "\n",
		"\\r", "\r",
		"\\t", "\t",
		"\\\"", "\"",
		"\\'", "'",
	).Replace(rawCode)
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
