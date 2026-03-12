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
	"fmt"
	"log"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"

	"github.com/cloudwego/eino-examples/adk/common/model"
)

func NewBookRecommendAgent() adk.Agent {
	ctx := context.Background()

	a, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "BookRecommender",
		Description: "An agent that can recommend books",
		Instruction: `You are an expert book recommender.
Based on the user's request, use the "search_book" tool to find relevant books. Finally, present the results to the user.`,
		Model: model.NewChatModel(),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{NewBookRecommender(), NewAskForClarificationTool()},
			},
		},
	})
	if err != nil {
		log.Fatal(fmt.Errorf("failed to create chatmodel: %w", err))
	}

	return a
}
