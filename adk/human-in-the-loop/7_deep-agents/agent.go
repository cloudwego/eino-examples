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
	"os"
	"strconv"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	commonModel "github.com/cloudwego/eino-examples/adk/common/model"
	tool2 "github.com/cloudwego/eino-examples/adk/common/tool"
	"github.com/cloudwego/eino-examples/components/tool/middlewares/errorremover"
)

type rateLimitedModel struct {
	m     model.ToolCallingChatModel
	delay time.Duration
}

func (r *rateLimitedModel) WithTools(tools []*schema.ToolInfo) (model.ToolCallingChatModel, error) {
	newM, err := r.m.WithTools(tools)
	if err != nil {
		return nil, err
	}
	return &rateLimitedModel{newM, r.delay}, nil
}

func (r *rateLimitedModel) Generate(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.Message, error) {
	time.Sleep(r.delay)
	return r.m.Generate(ctx, input, opts...)
}

func (r *rateLimitedModel) Stream(ctx context.Context, input []*schema.Message, opts ...model.Option) (*schema.StreamReader[*schema.Message], error) {
	time.Sleep(r.delay)
	return r.m.Stream(ctx, input, opts...)
}

func getRateLimitDelay() time.Duration {
	delayMs := os.Getenv("RATE_LIMIT_DELAY_MS")
	if delayMs == "" {
		return 0
	}
	ms, err := strconv.Atoi(delayMs)
	if err != nil {
		return 0
	}
	return time.Duration(ms) * time.Millisecond
}

func newRateLimitedModel() model.ToolCallingChatModel {
	delay := getRateLimitDelay()
	if delay == 0 {
		return commonModel.NewChatModel()
	}
	return &rateLimitedModel{
		m:     commonModel.NewChatModel(),
		delay: delay,
	}
}

func buildResearchAgent(ctx context.Context, m model.ToolCallingChatModel) (adk.Agent, error) {
	searchTool, err := NewSearchTool(ctx)
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "ResearchAgent",
		Description: "A research agent that can search for information and gather data on various topics.",
		Instruction: `You are a research agent specialized in gathering information.
Use the search tool to find relevant information for the given task.
Provide comprehensive and accurate results.`,
		Model: m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{searchTool},
			},
		},
		MaxIterations: 10,
	})
}

func buildAnalysisAgent(ctx context.Context, m model.ToolCallingChatModel) (adk.Agent, error) {
	analyzeTool, err := NewAnalyzeTool(ctx)
	if err != nil {
		return nil, err
	}

	return adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "AnalysisAgent",
		Description: "An analysis agent that processes data and generates insights.",
		Instruction: `You are an analysis agent specialized in processing data and generating insights.
Use the analyze tool to process data and provide meaningful analysis.
Present your findings clearly and concisely.`,
		Model: m,
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{analyzeTool},
			},
		},
		MaxIterations: 10,
	})
}

func NewDataAnalysisDeepAgent(ctx context.Context, m model.ToolCallingChatModel) (adk.Agent, error) {
	researchAgent, err := buildResearchAgent(ctx, m)
	if err != nil {
		return nil, err
	}

	analysisAgent, err := buildAnalysisAgent(ctx, m)
	if err != nil {
		return nil, err
	}

	followUpTool := tool2.GetFollowUpTool()

	return deep.New(ctx, &deep.Config{
		Name:        "DataAnalysisAgent",
		Description: "A deep agent for comprehensive data analysis tasks that may require clarification from users.",
		Instruction: `You are a data analysis agent that helps users analyze market data and provide insights.

IMPORTANT: Before starting any analysis, you MUST first use the FollowUpTool to ask the user clarifying questions to understand:
1. What specific market sectors or industries they are interested in (e.g., technology, finance, healthcare)
2. What time period they want to analyze (e.g., last quarter, year-to-date, specific dates)
3. What type of analysis they need (e.g., trend analysis, comparison, statistical analysis)
4. Their risk tolerance for investment recommendations (e.g., conservative, moderate, aggressive)

Only after receiving answers from the user should you proceed with the analysis using the ResearchAgent and AnalysisAgent.

Available tools:
- FollowUpTool: Use this FIRST to ask clarifying questions before any analysis
- ResearchAgent: Use to search for market data and information
- AnalysisAgent: Use to analyze data and generate insights`,
		ChatModel: m,
		SubAgents: []adk.Agent{researchAgent, analysisAgent},
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools:               []tool.BaseTool{followUpTool},
				ToolCallMiddlewares: []compose.ToolMiddleware{errorremover.Middleware()}, // Inject the remove_error middleware.
			},
		},
		MaxIteration: 50,
	})
}
