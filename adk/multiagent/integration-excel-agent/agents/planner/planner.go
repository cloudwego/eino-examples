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

package planner

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/agents"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/generic"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/params"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/utils"
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

func NewPlanner(ctx context.Context, op commandline.Operator) (adk.Agent, error) {
	cm, err := newPlannerChatModel(ctx)
	if err != nil {
		return nil, err
	}

	a, err := planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		ChatModelWithFormattedOutput: cm,
		GenInputFn:                   newPlannerInputGen(plannerPromptTemplate),
		NewPlan: func(ctx context.Context) planexecute.Plan {
			return &generic.Plan{}
		},
	})
	if err != nil {
		return nil, err
	}

	return agents.NewWrite2PlanMDWrapper(a, op), nil
}

func newPlannerChatModel(ctx context.Context) (model.ToolCallingChatModel, error) {
	sc, err := generic.PlanToolInfo.ToJSONSchema()
	if err != nil {
		return nil, err
	}

	return utils.NewChatModel(ctx,
		utils.WithMaxTokens(4096),
		utils.WithTemperature(0),
		utils.WithTopP(0),
		utils.WithDisableThinking(true),
		utils.WithResponseFormatJsonSchema(&openai.ChatCompletionResponseFormatJSONSchema{
			Name:        generic.PlanToolInfo.Name,
			Description: generic.PlanToolInfo.Desc,
			JSONSchema:  sc,
			Strict:      true,
		}),
	)
}

func newPlannerInputGen(plannerPrompt prompt.ChatTemplate) planexecute.GenPlannerModelInputFn {
	return func(ctx context.Context, userInput []adk.Message) ([]adk.Message, error) {
		pf, _ := params.GetTypedContextParams[string](ctx, params.UserAllPreviewFilesSessionKey)

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
