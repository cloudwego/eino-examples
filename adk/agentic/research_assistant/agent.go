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

	localbackend "github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino/adk"
	fsmiddleware "github.com/cloudwego/eino/adk/middlewares/filesystem"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func newResearchAssistant(ctx context.Context, reportPath string) (adk.TypedAgent[*schema.AgenticMessage], error) {
	agenticModel, err := newAgenticModel(ctx)
	if err != nil {
		return nil, fmt.Errorf("create agentic model: %w", err)
	}

	tools, err := buildTools(reportPath)
	if err != nil {
		return nil, fmt.Errorf("build tools: %w", err)
	}

	fsBackend, err := localbackend.NewBackend(ctx, &localbackend.Config{})
	if err != nil {
		return nil, fmt.Errorf("create filesystem backend: %w", err)
	}

	fsMiddleware, err := fsmiddleware.NewTyped[*schema.AgenticMessage](ctx, &fsmiddleware.MiddlewareConfig{
		Backend: fsBackend,
	})
	if err != nil {
		return nil, fmt.Errorf("create filesystem middleware: %w", err)
	}

	agent, err := adk.NewTypedChatModelAgent[*schema.AgenticMessage](ctx, &adk.TypedChatModelAgentConfig[*schema.AgenticMessage]{
		Name:        "AgenticResearchAssistant",
		Description: "Creates an evidence-backed research report with server search, local tools, and filesystem middleware.",
		Instruction: agentInstruction(reportPath),
		Model:       agenticModel,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		},
		Handlers: []adk.TypedChatModelAgentMiddleware[*schema.AgenticMessage]{
			fsMiddleware,
		},
		MaxIterations: 8,
	})
	if err != nil {
		return nil, fmt.Errorf("create agent: %w", err)
	}

	return agent, nil
}
