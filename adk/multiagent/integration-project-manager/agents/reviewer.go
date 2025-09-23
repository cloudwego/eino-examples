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

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

func NewReviewAgent(ctx context.Context, tcm model.ToolCallingChatModel) (adk.Agent, error) {
	// these sub-agents don't need description because they'll be set in a fixed workflow.
	questionAnalysisAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "question_analysis_agent",
		Description: "question analysis agent",
		Instruction: `You are the Question Analysis Agent. Your responsibilities include:

- Analyzing the given research or coding results to identify critical questions and evaluation criteria.
- Breaking down complex issues into clear, manageable components.
- Highlighting potential problems or areas of concern.
- Preparing a structured framework to guide the subsequent review generation.
- Ensuring thorough understanding of the content before passing it on.`,
		Model: tcm,
	})
	if err != nil {
		return nil, err
	}

	generateReviewAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "generate_review_agent",
		Description: "generate review agent",
		Instruction: `You are the Generate Review Agent. Your role is to:

- Produce comprehensive and balanced reviews based on the question analysis.
- Highlight strengths, weaknesses, and areas for improvement.
- Provide constructive and actionable feedback.
- Maintain objectivity and clarity in your evaluations.
- Prepare the review content for validation in the next step.`,
		Model: tcm,
	})
	if err != nil {
		return nil, err
	}

	reviewValidationAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "review_validation_agent",
		Description: "review validation agent",
		Instruction: `You are the Review Validation Agent. Your tasks are to:

- Validate the generated review for accuracy, coherence, and fairness.
- Check for logical consistency and completeness.
- Identify any biases or errors and suggest corrections.
- Confirm that the review aligns with the original analysis and project goals.
- Approve the review for final presentation or request revisions if necessary.`,
		Model: tcm,
	})
	if err != nil {
		return nil, err
	}

	return adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
		Name:        "ReviewAgent",
		Description: "The ReviewAgent is responsible for evaluating research and coding results through a sequential workflow. It orchestrates three key steps—question analysis, review generation, and review validation—to provide well-reasoned assessments that support project management decisions.",
		SubAgents:   []adk.Agent{questionAnalysisAgent, generateReviewAgent, reviewValidationAgent},
	})
}
