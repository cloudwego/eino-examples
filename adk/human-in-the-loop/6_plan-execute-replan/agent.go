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

package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	commonModel "github.com/cloudwego/eino-examples/adk/common/model"
)

type rateLimitedModel struct {
	m     model.ToolCallingChatModel
	delay time.Duration
}

func (r *rateLimitedModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	newM, err := r.m.WithTools(tools)
	if err != nil {
		return nil, err
	}
	return &rateLimitedModel{newM, r.delay}, nil
}

func (r *rateLimitedModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	time.Sleep(r.delay)
	return r.m.Generate(ctx, input, opts...)
}

func (r *rateLimitedModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	time.Sleep(r.delay)
	return r.m.Stream(ctx, input, opts...)
}

func getRateLimitDelay() time.Duration {
	delayMs := os.Getenv("RATE_LIMIT_DELAY_MS")
	if delayMs == "" {
		return 0
	}
	ms, err := strconv.Atoi(delayMs)
	if err != nil {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}

func newRateLimitedModel() model.ToolCallingChatModel {
	delay := getRateLimitDelay()
	if delay == 0 {
		return commonModel.NewChatModel()
	}
	return &rateLimitedModel{
		m:     commonModel.NewChatModel(),
		delay: delay,
	}
}

func NewPlanner(ctx context.Context) (adk.Agent, error) {
	return planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		ToolCallingChatModel: newRateLimitedModel(),
	})
}

var executorPrompt = prompt.FromMessages(schema.FString,
	schema.SystemMessage(`You are a diligent travel booking assistant. Follow the given plan and execute your tasks carefully.
Execute each planning step by using available tools.
For weather queries, use get_weather tool.
For flight bookings, use book_flight tool - this will require user review before confirmation.
For hotel bookings, use book_hotel tool - this will require user review before confirmation.
For attraction research, use search_attractions tool.
Provide detailed results for each task.`),
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

func formatExecutedSteps(in []planexecute.ExecutedStep) string {
	var sb strings.Builder
	for idx, m := range in {
		sb.WriteString(fmt.Sprintf("## %d. Step: %v\n  Result: %v\n\n", idx+1, m.Step, m.Result))
	}
	return sb.String()
}

func NewExecutor(ctx context.Context) (adk.Agent, error) {
	travelTools, err := GetAllTravelTools(ctx)
	if err != nil {
		return nil, err
	}

	return planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
		Model: newRateLimitedModel(),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: travelTools,
			},
		},

		GenInputFn: func(ctx context.Context, in *planexecute.ExecutionContext) ([]adk.Message, error) {
			planContent, err_ := in.Plan.MarshalJSON()
			if err_ != nil {
				return nil, err_
			}

			firstStep := in.Plan.FirstStep()

			msgs, err_ := executorPrompt.Format(ctx, map[string]any{
				"input":          formatInput(in.UserInput),
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

func NewReplanner(ctx context.Context) (adk.Agent, error) {
	return planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
		ChatModel: newRateLimitedModel(),
	})
}

func NewTravelPlanningAgent(ctx context.Context) (adk.Agent, error) {
	planAgent, err := NewPlanner(ctx)
	if err != nil {
		return nil, err
	}

	executeAgent, err := NewExecutor(ctx)
	if err != nil {
		return nil, err
	}

	replanAgent, err := NewReplanner(ctx)
	if err != nil {
		return nil, err
	}

	return planexecute.New(ctx, &planexecute.Config{
		Planner:       planAgent,
		Executor:      executeAgent,
		Replanner:     replanAgent,
		MaxIterations: 20,
	})
}
