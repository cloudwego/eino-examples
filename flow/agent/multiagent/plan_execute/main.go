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
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/deepseek"
	callbacks2 "github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/eino/utils/callbacks"

	"github.com/cloudwego/eino-examples/flow/agent/multiagent/plan_execute/debug"
	"github.com/cloudwego/eino-examples/flow/agent/multiagent/plan_execute/tools"
)

func main() {
	ctx := context.Background()

	deepSeekModel, err := deepseek.NewChatModel(ctx, &deepseek.ChatModelConfig{
		Model:   os.Getenv("DEEPSEEK_MODEL_NAME"),
		APIKey:  os.Getenv("DEEPSEEK_API_KEY"),
		BaseURL: os.Getenv("DEEPSEEK_BASE_URL"),
	})
	if err != nil {
		log.Fatalf("new DeepSeek model failed: %v", err)
	}

	arkModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey: os.Getenv("ARK_API_KEY"),
		Model:  os.Getenv("ARK_MODEL_NAME"),
	})
	if err != nil {
		log.Fatalf("new Ark model failed: %v", err)
	}

	toolsConfig, err := tools.GetTools(ctx)
	if err != nil {
		log.Fatalf("get tools config failed: %v", err)
	}

	// 创建多智能体的配置，system prompt 都用默认值
	config := &Config{
		// planner 在调试时大部分场景不需要真的去生成，可以用 mock 输出替代
		PlannerModel: &debug.ChatModelDebugDecorator{
			Model: deepSeekModel,
		},
		ExecutorModel: arkModel,
		ToolsConfig:   compose.ToolsNodeConfig{Tools: toolsConfig},
		ReviserModel: &debug.ChatModelDebugDecorator{
			Model: deepSeekModel,
		},
	}

	planExecuteAgent, err := NewMultiAgent(ctx, config)
	if err != nil {
		log.Fatalf("new plan execute multi agent failed: %v", err)
	}

	printer := newIntermediateOutputPrinter() // 创建一个中间结果打印器
	printer.printStream()                     // 开始异步输出到 console
	handler := printer.toCallbackHandler()    // 转化为 Eino 框架的 callback handler

	// 以流式方式调用多智能体，实际的 OutputStream 不再需要关注，因为所有输出都由 intermediateOutputPrinter 处理了
	_, err = planExecuteAgent.Stream(ctx, []*schema.Message{schema.UserMessage("我们一家三口去乐园玩，孩子身高 120 cm，预算 2000 元，希望能尽可能多的看表演，游乐设施则比较偏爱刺激项目，希望能在一天内尽可能多体验不同的活动，请帮忙规划一个可操作的一日行程。我们会在乐园开门的时候入场，玩到晚上闭园的时候。")},
		agent.WithComposeOptions(compose.WithCallbacks(handler)), // 将中间结果打印的 callback handler 注入进来
		// 给 planner 指定 mock 输出
		//agent.WithComposeOptions(compose.WithChatModelOption(debug.WithDebugOutput(schema.AssistantMessage(debug.PlannerOutput, nil))).DesignateNode(nodeKeyPlanner)),
		// 给 reviser 指定 mock 输出
		//agent.WithComposeOptions(compose.WithChatModelOption(debug.WithDebugOutput(schema.AssistantMessage("最终答案", nil))).DesignateNode(nodeKeyReviser)),
	)
	if err != nil {
		log.Fatalf("stream error: %v", err)
	}

	printer.wait() // 等待所有输出都处理完再结束
}

type coloredString struct {
	str  string
	code string
}

// intermediateOutputPrinter 利用 Eino 的 callback 机制，收集多智能体各步骤的实时输出.
type intermediateOutputPrinter struct {
	ch               chan coloredString
	currentAgentName string          // 当前智能体名称
	agentReasoning   map[string]bool // 智能体处在“推理”阶段还是“最终答案”阶段
	mu               sync.Mutex
	wg               sync.WaitGroup
}

func newIntermediateOutputPrinter() *intermediateOutputPrinter {
	return &intermediateOutputPrinter{
		ch: make(chan coloredString),
		agentReasoning: map[string]bool{
			nodeKeyPlanner:  false,
			nodeKeyExecutor: false,
			nodeKeyReviser:  false,
		},
	}
}

func (s *intermediateOutputPrinter) printStream() {
	go func() {
		for m := range s.ch {
			fmt.Print(m.code + m.str + Reset)
		}
	}()
}

func (s *intermediateOutputPrinter) toCallbackHandler() callbacks2.Handler {
	return callbacks.NewHandlerHelper().ChatModel(&callbacks.ModelCallbackHandler{
		OnEndWithStreamOutput: s.onChatModelEndWithStreamOutput,
	}).Tool(&callbacks.ToolCallbackHandler{
		OnStart: s.onToolStart,
		OnEnd:   s.onToolEnd,
	}).Handler()
}

func (s *intermediateOutputPrinter) wait() {
	s.wg.Wait()
}

const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	White  = "\033[97m"
	Cyan   = "\033[36m"
	Gray   = "\033[37m"
)

// onChatModelEndWithStreamOutput 当 ChatModel 结束时，获取它的流式输出并格式化处理.
func (s *intermediateOutputPrinter) onChatModelEndWithStreamOutput(ctx context.Context, runInfo *callbacks2.RunInfo, output *schema.StreamReader[*model.CallbackOutput]) context.Context {
	name := runInfo.Name
	if name != s.currentAgentName {
		s.ch <- coloredString{fmt.Sprintf("\n\n=======\n%s: \n=======\n", name), Cyan}
		s.currentAgentName = name
	}

	s.wg.Add(1)

	go func() {
		defer output.Close()
		defer s.wg.Done()

		for {
			chunk, err := output.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				log.Fatalf("internal error: %s\n", err)
			}

			if len(chunk.Message.Content) > 0 {
				if s.agentReasoning[name] { // 切换到最终答案阶段
					s.ch <- coloredString{"\nanswer begin: \n", Green}
					s.mu.Lock()
					s.agentReasoning[name] = false
					s.mu.Unlock()
				}
				s.ch <- coloredString{chunk.Message.Content, Yellow}
			} else if reasoningContent, ok := deepseek.GetReasoningContent(chunk.Message); ok {
				if !s.agentReasoning[name] { // 切换到推理阶段
					s.ch <- coloredString{"\nreasoning begin: \n", Green}
					s.mu.Lock()
					s.agentReasoning[name] = true
					s.mu.Unlock()
				}
				s.ch <- coloredString{reasoningContent, White}
			}
		}
	}()

	return ctx
}

// onToolStart 当 Tool 执行开始时，获取并输出调用信息.
func (s *intermediateOutputPrinter) onToolStart(ctx context.Context, info *callbacks2.RunInfo, input *tool.CallbackInput) context.Context {
	arguments := make(map[string]any)
	err := sonic.Unmarshal([]byte(input.ArgumentsInJSON), &arguments)
	if err != nil {
		s.ch <- coloredString{fmt.Sprintf("\ncall %s: %s\n", info.Name, input.ArgumentsInJSON), Red}
		return ctx
	}

	formatted, err := sonic.MarshalIndent(arguments, "  ", "  ")
	if err != nil {
		s.ch <- coloredString{fmt.Sprintf("\ncall %s: %s\n", info.Name, input.ArgumentsInJSON), Red}
		return ctx
	}

	s.ch <- coloredString{fmt.Sprintf("\ncall %s: %s\n", info.Name, string(formatted)), Red}
	return ctx
}

// onToolEnd 当 Tool 执行结束时，获取并输出返回结果.
func (s *intermediateOutputPrinter) onToolEnd(ctx context.Context, info *callbacks2.RunInfo, output *tool.CallbackOutput) context.Context {
	response := make(map[string]any)
	err := sonic.Unmarshal([]byte(output.Response), &response)
	if err != nil {
		s.ch <- coloredString{fmt.Sprintf("\ncall %s: %s\n", info.Name, output.Response), Blue}
		return ctx
	}

	formatted, err := sonic.MarshalIndent(response, "  ", "  ")
	if err != nil {
		s.ch <- coloredString{fmt.Sprintf("\ncall %s: %s\n", info.Name, output.Response), Blue}
		return ctx
	}

	s.ch <- coloredString{fmt.Sprintf("\ncall %s result: %s\n", info.Name, string(formatted)), Blue}
	return ctx
}
