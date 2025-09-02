package adk

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
)

var (
	replannerPrompt = prompt.FromMessages(schema.FString, schema.SystemMessage(`
当前的时间是：{{current_time}}

# 用户上传的文件预览(file_preview)：
{{files_preview}}
`), schema.UserMessage(`# OBJECTIVE
{input}

# ORIGINAL PLAN
{plan}

# COMPLETED STEPS & RESULTS
{executed_steps}`))
)

func replannerInputGen(ctx context.Context, in *prebuilt.PlanExecuteInput) ([]adk.Message, error) {
	replannerPrompt.Format(ctx, map[string]any{
		"input":          in.Input,
		"plan":           in.Plan,
		"executed_steps": in.ExecutedSteps,
	})
	return nil, nil
}
