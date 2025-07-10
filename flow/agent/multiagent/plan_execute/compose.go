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
	"io"
	"strings"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/flow/agent/multiagent/plan_execute/prompt"
)

// Config “计划——执行”多智能体的配置.
type Config struct {
	PlannerModel        model.BaseChatModel // planner 智能体使用的大模型
	PlannerSystemPrompt string              // planner 智能体的 system prompt

	ExecutorModel        model.ToolCallingChatModel // executor 智能体使用的大模型
	ToolsConfig          compose.ToolsNodeConfig    // executor 智能体使用的工具执行器配置
	ExecutorSystemPrompt string                     // executor 智能体的 system prompt

	ReviserModel        model.BaseChatModel // reviser 智能体使用的大模型
	ReviserSystemPrompt string              // reviser 智能体的 system prompt

	MaxStep int // 多智能体的最大执行步骤数，避免无限循环
}

// PlanExecuteMultiAgent “计划——执行”多智能体.
type PlanExecuteMultiAgent struct {
	// 图编排后的可执行体，输入是 Message 数组，输出是单条 Message
	runnable compose.Runnable[[]*schema.Message, *schema.Message]
}

// state 以多智能体一次运行为 scope 的全局状态，用于记录上下文
type state struct {
	messages []*schema.Message
}

const (
	nodeKeyPlanner        = "planner"          // planner 智能体的节点 key
	nodeKeyExecutor       = "executor"         // executor 智能体的节点 key
	nodeKeyReviser        = "reviser"          // reviser 智能体的节点 key
	nodeKeyTools          = "tools"            // tools 执行器的节点 key
	nodeKeyPlannerToList  = "planner_to_list"  // planner->executor 之间的 converter 节点 key
	nodeKeyExecutorToList = "executor_to_list" // executor->reviser 之间的 converter 节点 key
	nodeKeyReviserToList  = "reviser_to_list"  // reviser->executor 之间的 converter 节点 key
	defaultMaxStep        = 100                // 默认的最大执行步骤数量
)

// NewMultiAgent 根据配置编排一个“计划——执行”多智能体.
func NewMultiAgent(ctx context.Context, config *Config) (*PlanExecuteMultiAgent, error) {
	var (
		toolInfos      []*schema.ToolInfo
		toolsNode      *compose.ToolsNode
		err            error
		plannerPrompt  = config.PlannerSystemPrompt
		executorPrompt = config.ExecutorSystemPrompt
		reviserPrompt  = config.ReviserSystemPrompt
		maxStep        = config.MaxStep
	)

	if len(plannerPrompt) == 0 {
		plannerPrompt = prompt.DefaultPlannerPrompt
	}

	if len(executorPrompt) == 0 {
		executorPrompt = prompt.DefaultExecutorPrompt
	}

	if len(reviserPrompt) == 0 {
		reviserPrompt = prompt.DefaultReviserPrompt
	}

	if maxStep == 0 {
		maxStep = defaultMaxStep
	}

	if toolInfos, err = genToolInfos(ctx, config.ToolsConfig); err != nil {
		return nil, err
	}

	// 为 Executor 配置工具
	if config.ExecutorModel, err = config.ExecutorModel.WithTools(toolInfos); err != nil {
		return nil, err
	}

	// 初始化 Tool 执行器节点，传入可执行的工具
	if toolsNode, err = compose.NewToolNode(ctx, &config.ToolsConfig); err != nil {
		return nil, err
	}

	// 创建一个待编排的 graph，规定整体的输入输出类型，配置全局状态的初始化方法
	graph := compose.NewGraph[[]*schema.Message, *schema.Message](compose.WithGenLocalState(func(ctx context.Context) *state {
		return &state{}
	}))

	// 在大模型执行之前，向全局状态中保存上下文，并组装本次的上下文
	modelPreHandle := func(systemPrompt string, isDeepSeek bool) compose.StatePreHandler[[]*schema.Message, *state] {
		return func(ctx context.Context, input []*schema.Message, state *state) ([]*schema.Message, error) {
			for _, msg := range input {
				state.messages = append(state.messages, msg)
			}

			if isDeepSeek {
				return append([]*schema.Message{schema.SystemMessage(systemPrompt)}, convertMessagesForDeepSeek(state.messages)...), nil
			}

			return append([]*schema.Message{schema.SystemMessage(systemPrompt)}, state.messages...), nil
		}
	}

	// 定义 Executor 后的分支判断用的条件函数。该函数的输出是运行时选中的 NodeKey
	executorPostBranchCondition := func(_ context.Context, msg *schema.Message) (endNode string, err error) {
		if len(msg.ToolCalls) == 0 {
			return nodeKeyExecutorToList, nil
		}

		return nodeKeyTools, nil
	}

	// 定义 Reviser 后的分支判断用的条件函数。
	reviserPostBranchCondition := func(_ context.Context, sr *schema.StreamReader[*schema.Message]) (endNode string, err error) {
		defer sr.Close()

		var content string
		for {
			msg, err := sr.Recv()
			if err != nil {
				if err == io.EOF {
					return nodeKeyReviserToList, nil
				}
				return "", err
			}

			content += msg.Content

			if strings.Contains(content, "最终答案") {
				return compose.END, nil
			}
		}
	}

	// 添加 Planner 节点，同时添加 StatePreHandler 读写上下文
	_ = graph.AddChatModelNode(nodeKeyPlanner, config.PlannerModel, compose.WithStatePreHandler(modelPreHandle(plannerPrompt, true)), compose.WithNodeName(nodeKeyPlanner))

	// 添加 Executor 节点，同时添加 StatePreHandler 读写上下文
	_ = graph.AddChatModelNode(nodeKeyExecutor, config.ExecutorModel, compose.WithStatePreHandler(modelPreHandle(executorPrompt, false)), compose.WithNodeName(nodeKeyExecutor))

	// 添加 Reviser 节点，同时添加 StatePreHandler 读写上下文
	_ = graph.AddChatModelNode(nodeKeyReviser, config.ReviserModel, compose.WithStatePreHandler(modelPreHandle(reviserPrompt, true)), compose.WithNodeName(nodeKeyReviser))

	// 添加 Tool 执行器节点，同时添加 StatePreHandler 读写上下文
	_ = graph.AddToolsNode(nodeKeyTools, toolsNode, compose.WithStatePreHandler(func(ctx context.Context, in *schema.Message, state *state) (*schema.Message, error) {
		state.messages = append(state.messages, in)
		return in, nil
	}))

	// 添加三个 ToList 转换节点
	_ = graph.AddLambdaNode(nodeKeyPlannerToList, compose.ToList[*schema.Message]())
	_ = graph.AddLambdaNode(nodeKeyExecutorToList, compose.ToList[*schema.Message]())
	_ = graph.AddLambdaNode(nodeKeyReviserToList, compose.ToList[*schema.Message]())

	// 添加节点之间的边和分支
	_ = graph.AddEdge(compose.START, nodeKeyPlanner)
	_ = graph.AddEdge(nodeKeyPlanner, nodeKeyPlannerToList)
	_ = graph.AddEdge(nodeKeyPlannerToList, nodeKeyExecutor)
	_ = graph.AddBranch(nodeKeyExecutor, compose.NewGraphBranch(executorPostBranchCondition, map[string]bool{
		nodeKeyTools:          true,
		nodeKeyExecutorToList: true,
	}))
	_ = graph.AddEdge(nodeKeyTools, nodeKeyExecutor)
	_ = graph.AddEdge(nodeKeyExecutorToList, nodeKeyReviser)
	_ = graph.AddBranch(nodeKeyReviser, compose.NewStreamGraphBranch(reviserPostBranchCondition, map[string]bool{
		nodeKeyReviserToList: true,
		compose.END:          true,
	}))
	_ = graph.AddEdge(nodeKeyReviserToList, nodeKeyExecutor)

	// 编译 graph，将节点、边、分支转化为面向运行时的结构。由于 graph 中存在环，使用 AnyPredecessor 模式，同时设置运行时最大步数。
	runnable, err := graph.Compile(ctx, compose.WithNodeTriggerMode(compose.AnyPredecessor), compose.WithMaxRunSteps(maxStep))
	if err != nil {
		return nil, err
	}

	return &PlanExecuteMultiAgent{
		runnable: runnable,
	}, nil
}

// Generate 以非流式的方式调用多智能体.
func (r *PlanExecuteMultiAgent) Generate(ctx context.Context, input []*schema.Message, opts ...agent.AgentOption) (output *schema.Message, err error) {
	output, err = r.runnable.Invoke(ctx, input, agent.GetComposeOptions(opts...)...)
	if err != nil {
		return nil, err
	}

	return output, nil
}

// Stream 以流式的方式调用多智能体.
func (r *PlanExecuteMultiAgent) Stream(ctx context.Context, input []*schema.Message, opts ...agent.AgentOption) (
	output *schema.StreamReader[*schema.Message], err error) {
	res, err := r.runnable.Stream(ctx, input, agent.GetComposeOptions(opts...)...)
	if err != nil {
		return nil, err
	}

	return res, nil
}

// 把可执行的 Tool 转化为大模型可用的 Tool 信息
func genToolInfos(ctx context.Context, config compose.ToolsNodeConfig) ([]*schema.ToolInfo, error) {
	toolInfos := make([]*schema.ToolInfo, 0, len(config.Tools))
	for _, t := range config.Tools {
		tl, err := t.Info(ctx)
		if err != nil {
			return nil, err
		}

		toolInfos = append(toolInfos, tl)
	}

	return toolInfos, nil
}

func convertMessagesForDeepSeek(messages []*schema.Message) (converted []*schema.Message) {
	converted = make([]*schema.Message, 0, len(messages)*2)
	for _, message := range messages {
		if message.Role == schema.Tool { // 有 DeepSeek 服务商如火山引擎，目前不支持传入 ToolMessage
			converted = append(converted, schema.AssistantMessage(message.Content, nil))
		} else if message.Role == schema.Assistant {
			if len(message.ToolCalls) == 0 {
				converted = append(converted, message)
			} else {
				if len(message.Content) > 0 {
					converted = append(converted, schema.AssistantMessage(message.Content, nil))
				}
				for _, toolCall := range message.ToolCalls {
					converted = append(converted, schema.AssistantMessage(fmt.Sprintf("call %s with %s, got response:", toolCall.Function.Name, toolCall.Function.Arguments), nil))
				}
			}
		} else {
			converted = append(converted, message)
		}
	}

	return converted
}
