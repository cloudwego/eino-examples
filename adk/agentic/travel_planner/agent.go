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

	"github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/filesystem"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func newTravelPlannerAgent(ctx context.Context, planPath string) (adk.TypedAgent[*schema.AgenticMessage], error) {
	agenticModel, err := newAgenticModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("create agentic model: %w", err)
	}

	tools, err := buildTools(planPath)
	if err != nil {
		return nil, fmt.Errorf("build tools: %w", err)
	}

	backend, err := local.NewBackend(ctx, &local.Config{})
	if err != nil {
		return nil, fmt.Errorf("create local filesystem backend: %w", err)
	}

	filesystemPrompt := fmt.Sprintf(`当前示例可以使用 filesystem 工具。
除非用户明确要求，否则只在本示例 workspace 内读写文件。
最终旅行计划路径：%s`, planPath)
	fsMiddleware, err := filesystem.NewTyped[*schema.AgenticMessage](ctx, &filesystem.MiddlewareConfig{
		Backend:            backend,
		CustomSystemPrompt: &filesystemPrompt,
	})
	if err != nil {
		return nil, fmt.Errorf("create filesystem middleware: %w", err)
	}

	handlers := []adk.TypedChatModelAgentMiddleware[*schema.AgenticMessage]{
		fsMiddleware,
		newTravelPolicyMiddleware(maxAutoBookingOptionCNY),
	}

	agent, err := adk.NewTypedChatModelAgent[*schema.AgenticMessage](ctx, &adk.TypedChatModelAgentConfig[*schema.AgenticMessage]{
		Name:        "AgenticTravelPlanner",
		Description: "生成经过联网核验的杭州旅行计划，并用 middleware 检查预算和付款策略。",
		Instruction: agentInstruction(planPath),
		Model:       agenticModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		},
		Handlers:      handlers,
		MaxIterations: 10,
	})
	if err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}

	return agent, nil
}
