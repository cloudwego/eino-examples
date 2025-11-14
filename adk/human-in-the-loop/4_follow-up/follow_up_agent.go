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
	tool2 "github.com/cloudwego/eino-examples/adk/common/tool"
)

// the follow-up agent uses the FollowUpTool to ask questions and fetch answers.
// It then needs to extract from answer required information.
// If failed to extract, it should formulate new questions and ask again.
// Until one of the following conditions is met:
// 1. Successfully extract required information.
// 2. Reach the maximum number of attempts.
// 3. The user explicitly indicates that the information is not available or refuse to answer.
// After that, the agent should summarize the information extracted and return it.

func NewFollowUpAgent() adk.Agent {
	ctx := context.Background()

	a, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "FollowUpAgent",
		Description: "An agent that can ask questions and extract information from answers",
		Instruction: `You are an expert question asker.
Based on the user's request, use the "FollowUpTool" to ask questions and extract information from answers.
If you failed to extract, you should formulate new questions or reuse the remaining unanswered questions and ask again.
Until one of the following conditions is met:
1. Successfully extracted required information.
2. The user explicitly indicates that the information is not available or refuse to answer.
After that, you should summarize the information extracted and return it in a matter of fact tone.
In your final response, DO NOT make any suggestions, just summarize the information extracted.
DO NOT ask more than 3 questions at a time.`,
		Model: model.NewChatModel(),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{tool2.GetFollowUpTool()},
			},
		},
		MaxIterations: 10,
	})
	if err != nil {
		log.Fatal(fmt.Errorf("failed to create chatmodel: %w", err))
	}

	return a
}
