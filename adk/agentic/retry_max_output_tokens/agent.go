/*
 * Copyright 2026 CloudWeGo Authors
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
	"strings"

	"github.com/cloudwego/eino/adk"
	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func newTravelAgent(
	ctx context.Context,
	model einoModel.AgenticModel,
	inspect responseInspector,
) (adk.TypedAgent[*schema.AgenticMessage], error) {
	tools, err := buildTools()
	if err != nil {
		return nil, fmt.Errorf("build tools: %w", err)
	}

	agent, err := adk.NewTypedChatModelAgent[*schema.AgenticMessage](ctx, &adk.TypedChatModelAgentConfig[*schema.AgenticMessage]{
		Name:        "travel-agent-with-token-budget-retry",
		Description: "Plans a travel route with one local tool call and retries incomplete Responses API outputs by increasing max_output_tokens.",
		Instruction: agentInstruction(),
		Model:       model,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		},
		MaxIterations:    6,
		ModelRetryConfig: newMaxOutputTokenRetryConfig(retryMaxTokens, inspect),
	})
	if err != nil {
		return nil, fmt.Errorf("create ADK agent: %w", err)
	}

	return agent, nil
}

func agentInstruction() string {
	return strings.TrimSpace(fmt.Sprintf(`
You are a travel planning agent.
Call the %s tool exactly once before giving the final answer.
After the tool returns status=complete, do not call any tool again; write the final answer directly from the returned route.
Answer in Chinese with a practical, structured travel itinerary.
`, travelRouteToolName))
}
