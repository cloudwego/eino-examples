/*
 * Copyright 2024 CloudWeGo Authors
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

package eino_agent

import (
	"context"

	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
)

func defaultEinoAgentConfig(ctx context.Context) (*react.AgentConfig, error) {
	config := &react.AgentConfig{
		MaxStep:            25,
		ToolReturnDirectly: map[string]struct{}{},
	}
	configOfChatModel, err := defaultArkChatModelConfig(ctx)
	if err != nil {
		return nil, err
	}
	ArkChatModel, err := NewArkChatModel(ctx, configOfChatModel)
	if err != nil {
		return nil, err
	}
	config.Model = ArkChatModel
	DDGSearch, err := NewDDGSearch(ctx, nil)
	if err != nil {
		return nil, err
	}

	OpenFileTool, err := NewOpenFileTool(ctx)
	if err != nil {
		return nil, err
	}

	GitCloneFile, err := NewGitCloneFile(ctx)
	if err != nil {
		return nil, err
	}

	EinoAssistantTool, err := NewEinoAssistantTool(ctx)
	if err != nil {
		return nil, err
	}

	TodoTool, err := NewTodoTool(ctx)
	if err != nil {
		return nil, err
	}

	config.ToolsConfig.Tools = []tool.BaseTool{DDGSearch, OpenFileTool, GitCloneFile, EinoAssistantTool, TodoTool}

	return config, nil
}

func NewEinoAgent(ctx context.Context, config *react.AgentConfig) (lb *compose.Lambda, err error) {
	if config == nil {
		config, err = defaultEinoAgentConfig(ctx)
		if err != nil {
			return nil, err
		}
	}

	agent, err := react.NewAgent(ctx, config)
	if err != nil {
		return nil, err
	}

	lb, err = compose.AnyLambda(agent.Generate, agent.Stream, nil, nil)
	if err != nil {
		return nil, err
	}

	return lb, nil
}
