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

package subagents

import (
	"context"
	"log"

	"github.com/cloudwego/eino/adk"

	"github.com/cloudwego/eino-examples/adk/common/model"
)

func NewPlanAgent() adk.Agent {
	a, err := adk.NewChatModelAgent(context.Background(), &adk.ChatModelAgentConfig{
		Name:        "PlannerAgent",
		Description: "Generates a research plan based on a topic.",
		Instruction: `
You are an expert research planner. 
Your goal is to create a comprehensive, step-by-step research plan for a given topic. 
The plan should be logical, clear, and easy to follow.
The user will provide the research topic. Your output must ONLY be the research plan itself, without any conversational text, introductions, or summaries.`,
		Model:     model.NewChatModel(),
		OutputKey: "Plan",
	})
	if err != nil {
		log.Fatal(err)
	}
	return a
}

func NewWriterAgent() adk.Agent {
	a, err := adk.NewChatModelAgent(context.Background(), &adk.ChatModelAgentConfig{
		Name:        "WriterAgent",
		Description: "Writes a report based on a research plan.",
		Instruction: `
You are an expert academic writer.
You will be provided with a detailed research plan:
{Plan}

Your task is to expand on this plan to write a comprehensive, well-structured, and in-depth report.`,
		Model: model.NewChatModel(),
	})
	if err != nil {
		log.Fatal(err)
	}
	return a
}
