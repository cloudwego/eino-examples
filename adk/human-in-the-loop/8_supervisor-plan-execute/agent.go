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
	"github.com/cloudwego/eino/adk/prebuilt/supervisor"
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

type namedAgent struct {
	adk.ResumableAgent
	name        string
	description string
}

func (n *namedAgent) Name(_ context.Context) string {
	return n.name
}

func (n *namedAgent) Description(_ context.Context) string {
	return n.description
}

func buildResearchAgent(ctx context.Context) (adk.Agent, error) {
	m := newRateLimitedModel()

	researchTools, err := GetResearchTools(ctx)
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "research_agent",
		Description: "the agent responsible for quick research and information gathering tasks",
		Instruction: `You are a research assistant agent.

INSTRUCTIONS:
- Assist ONLY with research and information gathering tasks
- Use the search_info tool to find relevant information
- Provide concise summaries of your findings
- After you're done with your tasks, respond to the supervisor directly
- Respond ONLY with the results of your research, do NOT include ANY other text.`,
		Model: m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: researchTools,
			},
		},
	})
}

var executorPrompt = prompt.FromMessages(schema.FString,
	schema.SystemMessage(`You are a project execution assistant. Follow the given plan and execute your tasks carefully.
Execute each planning step by using available tools.
For requirements analysis, use analyze_requirements tool.
For design creation, use create_design tool.
For budget allocation, use allocate_budget tool - this requires approval as it's a financial operation.
For team assignment, use assign_team tool.
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

func buildProjectExecutionAgent(ctx context.Context) (adk.Agent, error) {
	planAgent, err := planexecute.NewPlanner(ctx, &planexecute.PlannerConfig{
		ToolCallingChatModel: newRateLimitedModel(),
	})
	if err != nil {
		return nil, err
	}

	projectTools, err := GetProjectExecutionTools(ctx)
	if err != nil {
		return nil, err
	}

	executeAgent, err := planexecute.NewExecutor(ctx, &planexecute.ExecutorConfig{
		Model: newRateLimitedModel(),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: projectTools,
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
	if err != nil {
		return nil, err
	}

	replanAgent, err := planexecute.NewReplanner(ctx, &planexecute.ReplannerConfig{
		ChatModel: newRateLimitedModel(),
	})
	if err != nil {
		return nil, err
	}

	agent, err := planexecute.New(ctx, &planexecute.Config{
		Planner:       planAgent,
		Executor:      executeAgent,
		Replanner:     replanAgent,
		MaxIterations: 20,
	})
	if err != nil {
		return nil, err
	}

	return &namedAgent{
		ResumableAgent: agent,
		name:           "project_execution_agent",
		description:    "the agent responsible for complex project execution tasks that require planning, including requirements analysis, design creation, budget allocation, and team assignment",
	}, nil
}

func buildProjectManagerSupervisor(ctx context.Context) (adk.Agent, error) {
	m := newRateLimitedModel()

	sv, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "project_manager",
		Description: "the supervisor agent responsible for coordinating project management tasks",
		Instruction: `You are a project manager supervisor managing two agents:

- a research_agent: Assign quick research and information gathering tasks to this agent (market research, technology trends, competitor analysis)
- a project_execution_agent: Assign complex project execution tasks to this agent (project setup, requirements analysis, design creation, budget allocation, team assignment)

INSTRUCTIONS:
- Analyze the user's request and delegate to the appropriate agent
- For simple research queries, use research_agent
- For complex project tasks that require multiple steps (like setting up a new project), use project_execution_agent
- The project_execution_agent will create a plan and execute it step by step
- Budget allocation requires user approval - the project_execution_agent will handle this
- Assign work to one agent at a time, do not call agents in parallel
- Do not do any work yourself - always delegate to the appropriate agent
- After all tasks are complete, summarize the results for the user`,
		Model: m,
		Exit:  &adk.ExitTool{},
	})
	if err != nil {
		return nil, err
	}

	researchAgent, err := buildResearchAgent(ctx)
	if err != nil {
		return nil, err
	}

	projectExecutionAgent, err := buildProjectExecutionAgent(ctx)
	if err != nil {
		return nil, err
	}

	return supervisor.New(ctx, &supervisor.Config{
		Supervisor: sv,
		SubAgents:  []adk.Agent{researchAgent, projectExecutionAgent},
	})
}
