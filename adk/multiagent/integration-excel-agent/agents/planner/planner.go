package planner

import (
	"context"
	"os"

	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/agents"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/consts"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/utils"
	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"
	arkmodel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

var (
	plannerPromptTemplate = prompt.FromMessages(schema.Jinja2,
		schema.SystemMessage(`You are an expert planner specializing in Excel data processing tasks. Your goal is to understand user requirements and break them down into a clear, step-by-step plan.

**1. Understanding the Goal:**
- Carefully analyze the user's request to determine the ultimate objective.
- Identify the input data (Excel files) and the desired output format.

**2. Deliverables:**
- The final output should be a JSON object representing the plan, containing a list of steps.
- Each step must be a clear and concise instruction for the agent that will execute this step.

**3. Plan Decomposition Principles:**
- **Granularity:** Break down the task into the smallest possible logical steps. For example, instead of "process the data," use "read the Excel file," "filter out rows with missing values," "calculate the average of the 'Sales' column," etc.
- **Sequence:** The steps should be in the correct order of execution.
- **Clarity:** Each step should be unambiguous and easy for the agent that will execute this step to understand.

**4. Output Format (Few-shot Example):**
Here is an example of a good plan:
User Request: "Please calculate the average sales for each product category in the attached 'sales_data.xlsx' file and generate a report."
{
  "steps": [
    {
      "instruction": "Read the 'sales_data.xlsx' file into a pandas DataFrame."
    },
    {
      "instruction": "Group the DataFrame by 'Product Category' and calculate the mean of the 'Sales' column for each group."
    },
    {
      "instruction": "Summarize the average sales for each product category and present the results in a table."
    }
  ]
}

**5. Restrictions:**
- Do not generate code directly in the plan.
- Ensure that the plan is logical and achievable.
- The final step should always be to generate a report or provide the final result.
`),
		schema.UserMessage(`
User Query: {{ user_query }}
Current Time: {{ current_time }}
File Preview (If file has xlsx extension, the preview will provide the specific contents of the first 20 lines, otherwise only the file path will be provided):
{{ file_preview }}
`),
	)
)

func NewPlanner(ctx context.Context) (adk.Agent, error) {
	cm, err := newPlannerChatModel(ctx)
	if err != nil {
		return nil, err
	}

	a, err := planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		ChatModelWithFormattedOutput: cm,
		GenInputFn:                   newPlannerInputGen(plannerPromptTemplate),
		NewPlan: func(ctx context.Context) planexecute.Plan {
			return &agents.Plan{}
		},
	})
	if err != nil {
		return nil, err
	}

	return agents.NewWrite2PlanMDWrapper(a), nil
}

func newPlannerChatModel(ctx context.Context) (model.ToolCallingChatModel, error) {
	sc, err := agents.PlanToolInfo.ToJSONSchema()
	if err != nil {
		return nil, err
	}

	return ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey:  os.Getenv("ARK_API_KEY"),
		BaseURL: os.Getenv("ARK_BASE_URL"),
		Region:  os.Getenv("ARK_REGION"),
		Model:   os.Getenv("ARK_MODEL"),
		Thinking: &arkmodel.Thinking{
			Type: arkmodel.ThinkingTypeDisabled,
		},
		ResponseFormat: &ark.ResponseFormat{
			Type: arkmodel.ResponseFormatJSONSchema,
			JSONSchema: &arkmodel.ResponseFormatJSONSchemaJSONSchemaParam{
				Name:        agents.PlanToolInfo.Name,
				Description: agents.PlanToolInfo.Desc,
				Schema:      sc,
				Strict:      true,
			},
		},
		MaxTokens:   utils.PtrOf(4096),
		Temperature: utils.PtrOf(float32(0)),
		TopP:        utils.PtrOf(float32(0)),
	})
}

func newPlannerInputGen(plannerPrompt prompt.ChatTemplate) planexecute.GenPlannerModelInputFn {
	return func(ctx context.Context, userInput []adk.Message) ([]adk.Message, error) {
		pf, _ := consts.GetSessionValue[string](ctx, consts.UserAllPreviewFilesSessionKey)

		msgs, err := plannerPrompt.Format(ctx, map[string]any{
			"user_query":   utils.FormatInput(userInput),
			"current_time": utils.GetCurrentTime(),
			"file_preview": pf,
		})
		if err != nil {
			return nil, err
		}

		return msgs, nil
	}
}
