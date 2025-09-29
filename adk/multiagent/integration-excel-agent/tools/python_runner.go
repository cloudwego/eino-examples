package tools

import (
	"context"
	"fmt"
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
					"Starting with ```python and end with ```, enclosing the complete Python code within.",
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
	var input struct {
		Code string `json:"code"`
	}

	err := jsoniter.Unmarshal([]byte(argumentsInJSON), &input)
	if err != nil {
		return "", err
	}

	code := extractCodeSnippet(input.Code)
	if code == "" {
		return "", fmt.Errorf("python code is empty")
	}
	taskID := params.GetCozeSpaceTaskID(ctx)
	fileName := fmt.Sprintf("script_%s.py", taskID)

	if err = p.op.WriteFile(ctx, fileName, code); err != nil {
		return fmt.Sprintf("failed to create python file %s: %v", fileName, err), nil
	}

	result, err := p.op.RunCommand(ctx, "python3 "+fileName)
	if err != nil {
		return "", fmt.Errorf("execute error: %w", err)
	}

	return result, nil
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
