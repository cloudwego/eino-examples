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

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/supervisor"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"

	"github.com/cloudwego/eino-examples/adk/common/model"
	"github.com/cloudwego/eino-examples/flow/agent/multiagent/plan_execute/tools"
)

func buildSearchAgent(ctx context.Context) (adk.Agent, error) {
	m := model.NewChatModel()

	type searchReq struct {
		Query string `json:"query"`
	}

	type searchResp struct {
		Result string `json:"result"`
	}

	search := func(ctx context.Context, req *searchReq) (*searchResp, error) {
		return &searchResp{
			Result: "In 2024, the US GDP was $29.18 trillion and New York State's GDP was $2.297 trillion",
		}, nil
	}

	searchTool, err := tools.SafeInferTool("search", "search the internet for info", search)
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "research_agent",
		Description: "the agent responsible to search the internet for info",
		Instruction: `
		You are a research agent.


        INSTRUCTIONS:
        - Assist ONLY with research-related tasks, DO NOT do any math
        - After you're done with your tasks, respond to the supervisor directly
        - Respond ONLY with the results of your work, do NOT include ANY other text.`,
		Model: m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{searchTool},
				UnknownToolsHandler: func(ctx context.Context, name, input string) (string, error) {
					return fmt.Sprintf("unknown tool: %s", name), nil
				},
			},
		},
	})
}

func buildSubtractAgent(ctx context.Context) (adk.Agent, error) {
	m := model.NewChatModel()

	type subtractReq struct {
		A float64 `json:"a"`
		B float64 `json:"b"`
	}

	type subtractResp struct {
		Result float64
	}

	subtract := func(ctx context.Context, req *subtractReq) (*subtractResp, error) {
		return &subtractResp{
			Result: req.A - req.B,
		}, nil
	}

	subtractTool, err := tools.SafeInferTool("subtract", "subtract two numbers", subtract)
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "subtract_agent",
		Description: "the agent responsible to do math subtractions",
		Instruction: `
		You are a math subtraction agent.


        INSTRUCTIONS:
        - Assist ONLY with math subtraction-related tasks
        - After you're done with your tasks, respond to the supervisor directly
        - Respond ONLY with the results of your work, do NOT include ANY other text.`,
		Model: m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{subtractTool},
				UnknownToolsHandler: func(ctx context.Context, name, input string) (string, error) {
					return fmt.Sprintf("unknown tool: %s", name), nil
				},
			},
		},
	})
}

func buildMultiplyAgent(ctx context.Context) (adk.Agent, error) {
	m := model.NewChatModel()

	type multiplyReq struct {
		A float64 `json:"a"`
		B float64 `json:"b"`
	}

	type multiplyResp struct {
		Result float64
	}

	multiply := func(ctx context.Context, req *multiplyReq) (*multiplyResp, error) {
		return &multiplyResp{
			Result: req.A * req.B,
		}, nil
	}

	multiplyTool, err := tools.SafeInferTool("multiply", "multiply two numbers", multiply)
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "multiply_agent",
		Description: "the agent responsible to do math multiplications",
		Instruction: `
		You are a math multiplication agent.


        INSTRUCTIONS:
        - Assist ONLY with math multiplication-related tasks
        - After you're done with your tasks, respond to the supervisor directly
        - Respond ONLY with the results of your work, do NOT include ANY other text.`,
		Model: m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{multiplyTool},
				UnknownToolsHandler: func(ctx context.Context, name, input string) (string, error) {
					return fmt.Sprintf("unknown tool: %s", name), nil
				},
			},
		},
	})
}

func buildDivideAgent(ctx context.Context) (adk.Agent, error) {
	m := model.NewChatModel()

	type divideReq struct {
		A float64 `json:"a"`
		B float64 `json:"b"`
	}

	type divideResp struct {
		Result float64
	}

	divide := func(ctx context.Context, req *divideReq) (*divideResp, error) {
		return &divideResp{
			Result: req.A / req.B,
		}, nil
	}

	divideTool, err := tools.SafeInferTool("divide", "divide two numbers", divide)
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "divide_agent",
		Description: "the agent responsible to do math division",
		Instruction: `
		You are a math division agent.


        INSTRUCTIONS:
        - Assist ONLY with math division-related tasks
        - After you're done with your tasks, respond to the supervisor directly
        - Respond ONLY with the results of your work, do NOT include ANY other text.`,
		Model: m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{divideTool},
				UnknownToolsHandler: func(ctx context.Context, name, input string) (string, error) {
					return fmt.Sprintf("unknown tool: %s", name), nil
				},
			},
		},
	})
}

func buildMathAgent(ctx context.Context) (adk.Agent, error) {
	m := model.NewChatModel()

	sa, err := buildSubtractAgent(ctx)
	if err != nil {
		return nil, err
	}

	ma, err := buildMultiplyAgent(ctx)
	if err != nil {
		return nil, err
	}

	da, err := buildDivideAgent(ctx)
	if err != nil {
		return nil, err
	}

	mathA, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "math_agent",
		Description: "the agent responsible to do math",
		Instruction: `
		You are a math agent.


        INSTRUCTIONS:
        - Assist ONLY with math-related tasks
        - After you're done with your tasks, respond to the supervisor directly
        - Respond ONLY with the results of your work, do NOT include ANY other text.
		- You are yourself also a supervisor managing three agents:
		- an subtract_agent, a multiply_agent, a divide_agent. Assign math-related tasks to these agents.
		- Assign work to one agent at a time, do not call agents in parallel.
		- Do not do any real math work yourself, always transfer to your sub agents to do actual computation.`,
		Model: m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				UnknownToolsHandler: func(ctx context.Context, name, input string) (string, error) {
					return fmt.Sprintf("unknown tool: %s", name), nil
				},
			},
		},
	})

	return supervisor.New(ctx, &supervisor.Config{
		Supervisor: mathA,
		SubAgents:  []adk.Agent{sa, ma, da},
	})
}

func buildSupervisor(ctx context.Context) (adk.Agent, error) {
	m := model.NewChatModel()

	sv, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "supervisor",
		Description: "the agent responsible to supervise tasks",
		Instruction: `
		You are a supervisor managing two agents:

        - a research agent. Assign research-related tasks to this agent
        - a math agent. Assign math-related tasks to this agent
        Assign work to one agent at a time, do not call agents in parallel.
        Do not do any work yourself.`,
		Model: m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				UnknownToolsHandler: func(ctx context.Context, name, input string) (string, error) {
					return fmt.Sprintf("unknown tool: %s", name), nil
				},
			},
		},
		Exit: &adk.ExitTool{},
	})
	if err != nil {
		return nil, err
	}

	searchAgent, err := buildSearchAgent(ctx)
	if err != nil {
		return nil, err
	}
	mathAgent, err := buildMathAgent(ctx)
	if err != nil {
		return nil, err
	}

	return supervisor.New(ctx, &supervisor.Config{
		Supervisor: sv,
		SubAgents:  []adk.Agent{searchAgent, mathAgent},
	})
}
