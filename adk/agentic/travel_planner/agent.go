/*
 * Copyright 2026 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     https://www.apache.org/licenses/LICENSE-2.0
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

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func newTravelRunner(ctx context.Context, name, description, instruction string, mdl model.AgenticModel, tools []tool.BaseTool) (*adk.TypedRunner[*schema.AgenticMessage], error) {
	conf := &adk.TypedChatModelAgentConfig[*schema.AgenticMessage]{
		Name:          name,
		Description:   description,
		Instruction:   instruction,
		Model:         mdl,
		MaxIterations: 8,
	}

	if len(tools) > 0 {
		conf.ToolsConfig = adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: tools,
			},
		}
	}

	agent, err := adk.NewTypedChatModelAgent[*schema.AgenticMessage](ctx, conf)
	if err != nil {
		return nil, err
	}

	return adk.NewTypedRunner[*schema.AgenticMessage](adk.TypedRunnerConfig[*schema.AgenticMessage]{
		Agent:           agent,
		EnableStreaming: true,
	}), nil
}
