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

func NewMainAgent() adk.Agent {
	a, err := adk.NewChatModelAgent(context.Background(), &adk.ChatModelAgentConfig{
		Name:        "MainAgent",
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
	a, err := adk.NewChatModelAgent(context.Background(), &adk.ChatModelAgentConfig{
		Name:        "CritiqueAgent",
		Description: "Critique agent that reviews the main agent's work and provides feedback.",
		Instruction: `You are a critique agent responsible for reviewing the main agent's work.
Analyze the provided solution for accuracy, completeness, and quality.
If you find issues or areas for improvement, provide specific feedback.
If the work is satisfactory, call the 'exit' tool and provide a final summary response.`,
		Model: model.NewChatModel(),
		// Exit:  nil, // use default exit tool
	})
	if err != nil {
		log.Fatal(err)
	}
	return a
}
