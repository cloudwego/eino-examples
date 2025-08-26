package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/adk/common/model"
	"github.com/cloudwego/eino-examples/adk/multiagent/plan-execute-replan/tools"
)

func NewPlanner(ctx context.Context) (adk.Agent, error) {

	return prebuilt.NewPlanner(ctx, &prebuilt.PlannerConfig{
		ToolCallingChatModel: model.NewChatModel(),
	})
}

var executorPrompt = prompt.FromMessages(schema.FString,
	schema.SystemMessage(`You are a diligent and meticulous travel research executor, Follow the given plan and execute your tasks carefully and thoroughly.
Execute each planning step by using available tools.
For weather queries, use get_weather tool.
For flight searches, use search_flights tool.
For hotel searches, use search_hotels tool.
For attraction research, use search_attractions tool.
For user's clarification, use ask_for_clarification tool. In summary, repeat the questions and results to confirm with the user, try to avoid disturbing users'
Provide detailed results for each task.
Cloud Call multiple tools to get the final result.`),
	schema.UserMessage(`## OBJECTIVE
{input}
## Given the following plan:
{plan}
## COMPLETED STEPS & RESULTS
{executed_steps}
## Your task is to execute the first step, which is: 
{step}`))

func formatInput(in []adk.Message) string {
	return in[0].Content
}

func formatExecutedSteps(in []prebuilt.ExecutedStep) string {
	var sb strings.Builder
	for idx, m := range in {
		sb.WriteString(fmt.Sprintf("## %d. Step: %v\n  Result: %v\n\n", idx+1, m.Step, m.Result))
	}
	return sb.String()
}

func NewExecutor(ctx context.Context) (adk.Agent, error) {
	// Get travel tools for the executor
	travelTools, err := tools.GetAllTravelTools(ctx)
	if err != nil {
		return nil, err
	}

	return prebuilt.NewExecutor(ctx, &prebuilt.ExecutorConfig{
		Model: model.NewChatModel(),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: travelTools,
			},
		},
		MaxStep: 20,

		GenInputFn: func(ctx context.Context, in *prebuilt.PlanExecuteInput) ([]adk.Message, error) {
			planContent, err_ := in.Plan.MarshalJSON()
			if err_ != nil {
				return nil, err_
			}

			firstStep := in.Plan.FirstStep()

			msgs, err_ := executorPrompt.Format(ctx, map[string]any{
				"input":          formatInput(in.Input),
				"plan":           string(planContent),
				"executed_steps": formatExecutedSteps(in.ExecutedSteps),
				"step":           firstStep,
			})
			if err_ != nil {
				return nil, err_
			}

			return msgs, nil
		},
	})
}

func NewReplanAgent(ctx context.Context) (adk.Agent, error) {
	return prebuilt.NewReplanner(ctx, &prebuilt.ReplannerConfig{
		ChatModel: model.NewChatModel(),
	})
}
