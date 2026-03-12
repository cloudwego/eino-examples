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
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"

	"github.com/cloudwego/eino-examples/adk/common/model"
)

func NewMainAgent() adk.Agent {
	a, err := adk.NewChatModelAgent(context.Background(), &adk.ChatModelAgentConfig{
		Name:        "main_agent",
		Description: "Main agent that attempts to solve the user's task.",
		Instruction: `You are the main agent responsible for solving the user's task. 
Provide a comprehensive solution based on the given requirements. 
Focus on delivering accurate and complete results.`,
		Model: model.NewChatModel(),
	})
	if err != nil {
		log.Fatal(err)
	}
	return a
}

func NewCritiqueAgent() adk.Agent {

	exitAndSummarizeTool, err := utils.InferTool("exit_and_summarize", "exit from the loop and provide a final summary response",
		func(ctx context.Context, req *exitAndSummarize) (string, error) {
			_ = adk.SendToolGenAction(ctx, "exit_and_summarize", adk.NewBreakLoopAction("critique_agent"))
			return req.Summary, nil
		})
	if err != nil {
		log.Fatalf("create tool failed, name=%v, err=%v", "exit_and_summarize", err)
	}
	a, err := adk.NewChatModelAgent(context.Background(), &adk.ChatModelAgentConfig{
		Name:        "critique_agent",
		Description: "Critique agent that reviews the main agent's work and provides feedback.",
		Instruction: `You are a critique agent responsible for reviewing the main agent's work.
Analyze the provided solution for accuracy, completeness, and quality.
If you find issues or areas for improvement, provide specific feedback.
If the work is satisfactory, call the 'exit_and_summarize' tool and provide a final summary response.`,
		Model: model.NewChatModel(),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					exitAndSummarizeTool,
				},
			},
			ReturnDirectly: map[string]bool{
				"exit_and_summarize": true,
			},
		},
	})
	if err != nil {
		log.Fatalf("create agent failed, name=%v, err=%v", "critique_agent", err)
	}
	return a
}

type exitAndSummarize struct {
	Summary string `json:"summary" jsonschema_description:"final summary of the solution"`
}
