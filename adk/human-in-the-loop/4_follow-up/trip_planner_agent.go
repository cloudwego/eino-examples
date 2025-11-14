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
	"log"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"

	"github.com/cloudwego/eino-examples/adk/common/model"
)

func NewItineraryAgent() adk.Agent {
	ctx := context.Background()

	a, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "ItineraryAgent",
		Description: "An agent that can plan trips.",
		Instruction: `You are an expert trip planner. Your goal is to create a personalized itinerary.
If you have enough information (e.g., location and interests), create the plan.
If you are missing information, you MUST use the 'FollowUpAgent' tool to ask the user for their interests.`,
		Model: model.NewChatModel(),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					adk.NewAgentTool(ctx, NewFollowUpAgent()),
				},
			},
		},
	})
	if err != nil {
		log.Fatal(fmt.Errorf("failed to create chatmodel: %w", err))
	}

	return a
}
