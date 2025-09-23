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
	"fmt"
	"log"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
)

func NewResearchAgent(ctx context.Context, tcm model.ToolCallingChatModel) (adk.Agent, error) {
	type webSearchInput struct {
		CurrentContext string `json:"current_context" jsonschema:"description=current context for web search"`
	}
	type webSearchOutput struct {
		Result []string
	}
	webSearchTool, err := utils.InferTool(
		"web_search",
		"web search tool",
		func(ctx context.Context, input *webSearchInput) (output *webSearchOutput, err error) {
			// replace it with real web search tool
			if input.CurrentContext == "" {
				return nil, fmt.Errorf("web search input is required")
			}
			return &webSearchOutput{}, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "ResearchAgent",
		Description: "The ResearchAgent is responsible for conducting research and generating feasible solutions. It supports interruption to receive additional contextual information from the user, which helps improve the accuracy and relevance of the research outcomes. It utilizes web search tools to gather up-to-date information.",
		Instruction: `You are the ResearchAgent. Your role is to:

- Conduct thorough research on the given topic or problem.
- Generate feasible and well-informed solutions based on your findings.
- Support interruptions by accepting additional context or information from the user at any time to refine your research.
- Use web search tools effectively to gather relevant and current data.
- Communicate your research results clearly and logically.
- Ask clarifying questions if needed to improve the quality of your research.
- Maintain a professional and helpful tone throughout the interaction.

Tool Handling:
- When you think the input information is insufficient to support the research, please use ask_for_clarification tool to ask the user to supplement the context.
- If context is fulfilled, you can use the web_search tool to obtain more data from internet.
`,
		Model: tcm,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{webSearchTool, newAskForClarificationTool()},
			},
		},
		MaxIterations: 5,
	})
}

type askForClarificationOptions struct {
	NewInput *string
}

func WithNewInput(input string) tool.Option {
	return tool.WrapImplSpecificOptFn(func(t *askForClarificationOptions) {
		t.NewInput = &input
	})
}

type AskForClarificationInput struct {
	Question string `json:"question" jsonschema:"description=The specific question you want to ask the user to get the missing information"`
}

func newAskForClarificationTool() tool.InvokableTool {
	t, err := utils.InferOptionableTool(
		"ask_for_clarification",
		"Call this tool when the user's request is ambiguous or lacks the necessary information to proceed. Use it to ask a follow-up question to get the details you need, such as the book's genre, before you can use other tools effectively.",
		func(ctx context.Context, input *AskForClarificationInput, opts ...tool.Option) (output string, err error) {
			o := tool.GetImplSpecificOptions[askForClarificationOptions](nil, opts...)
			if o.NewInput == nil {
				return "", compose.NewInterruptAndRerunErr(input.Question)
			}
			output = *o.NewInput
			o.NewInput = nil
			return output, nil
		})
	if err != nil {
		log.Fatal(err)
	}
	return t
}
