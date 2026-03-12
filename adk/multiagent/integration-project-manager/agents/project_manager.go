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

package agents

import (
	"context"
	"log"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

func NewProjectManagerAgent(ctx context.Context, tcm model.ToolCallingChatModel) (adk.Agent, error) {
	a, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "ProjectManagerAgent",
		Description: "The ProjectManagerAgent acts as the supervisor and coordinator of the project workflow. It dynamically routes and coordinates multiple sub-agents responsible for different dimensions of the work, such as research, coding, and review, based on user inputs and project needs.",
		Instruction: `You are the ProjectManagerAgent. Your role is to:

- Supervise and coordinate multiple specialized three sub-agents: ResearchAgent, CodeAgent, ReviewAgent.
  - ResearchAgent: assign this agent when you need to conduct research and generate feasible solutions.
  - CodeAgent: assign this agent when you need generate high-quality code.
  - ReviewAgent: assign this agent when you need evaluate research or coding results.
- Dynamically route tasks and user inputs to the appropriate sub-agent based on the current project requirements.
- Monitor the progress and outputs of each sub-agent to ensure alignment with project goals.
- Facilitate communication and collaboration among sub-agents to optimize workflow efficiency.
- Provide clear updates and summaries to the user regarding project status and next steps.
- Maintain a professional, organized, and proactive approach to project management.
`,
		Model: tcm,
		Exit:  &adk.ExitTool{},
	})
	if err != nil {
		log.Fatal(err)
	}
	return a, nil
}
