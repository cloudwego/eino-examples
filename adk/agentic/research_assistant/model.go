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
	"os"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/agenticark"
	"github.com/cloudwego/eino-ext/components/model/agenticopenai"
	einoModel "github.com/cloudwego/eino/components/model"
	openaiResponses "github.com/openai/openai-go/v3/responses"
	arkResponses "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model/responses"
)

const (
	providerARK    = "ark"
	providerOpenAI = "openai"
)

func newAgenticModel(ctx context.Context) (einoModel.AgenticModel, error) {
	provider := selectedAgenticProvider()
	switch provider {
	case providerARK:
		return newARKAgenticModel(ctx)
	case providerOpenAI:
		return newOpenAIAgenticModel(ctx)
	default:
		return nil, fmt.Errorf("unsupported AGENTIC_MODEL_PROVIDER %q, expected %q or %q",
			provider, providerARK, providerOpenAI)
	}
}

func newARKAgenticModel(ctx context.Context) (einoModel.AgenticModel, error) {
	timeout := 3 * time.Minute

	apiKey, err := requiredEnv("ARK_API_KEY")
	if err != nil {
		return nil, err
	}
	model, err := requiredEnv("ARK_MODEL_ID")
	if err != nil {
		return nil, err
	}

	return agenticark.New(ctx, &agenticark.Config{
		APIKey:  apiKey,
		Model:   model,
		BaseURL: os.Getenv("ARK_BASE_URL"),
		Timeout: &timeout,
	})
}

func newOpenAIAgenticModel(ctx context.Context) (einoModel.AgenticModel, error) {
	timeout := 3 * time.Minute

	apiKey, err := requiredEnv("OPENAI_API_KEY")
	if err != nil {
		return nil, err
	}
	model, err := requiredEnv("OPENAI_MODEL_ID")
	if err != nil {
		return nil, err
	}

	return agenticopenai.NewResponsesModel(ctx, &agenticopenai.ResponsesConfig{
		APIKey:  apiKey,
		Model:   model,
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
		Timeout: &timeout,
		Include: []openaiResponses.ResponseIncludable{
			openaiResponses.ResponseIncludableWebSearchCallActionSources,
			openaiResponses.ResponseIncludableReasoningEncryptedContent,
		},
	})
}

func agenticRunOptions() []einoModel.Option {
	switch selectedAgenticProvider() {
	case providerOpenAI:
		return openAIRunOptions()
	default:
		return arkRunOptions()
	}
}

func arkRunOptions() []einoModel.Option {
	return []einoModel.Option{
		einoModel.WithMaxTokens(8192),
		agenticark.WithThinking(&arkResponses.ResponsesThinking{
			Type: arkResponses.ThinkingType_enabled.Enum(),
		}),
		agenticark.WithReasoning(&arkResponses.ResponsesReasoning{
			Effort: arkResponses.ReasoningEffort_high,
		}),
		agenticark.WithMaxToolCalls(6),
		agenticark.WithParallelToolCalls(false),
		agenticark.WithServerTools([]*agenticark.ServerToolConfig{
			{
				WebSearch: &arkResponses.ToolWebSearch{
					Type:       arkResponses.ToolType_web_search,
					Limit:      ptrOf[int64](6),
					MaxKeyword: ptrOf[int32](3),
					Sources: []arkResponses.SourceType_Enum{
						arkResponses.SourceType_search_engine,
						arkResponses.SourceType_toutiao,
					},
				},
			},
		}),
	}
}

func openAIRunOptions() []einoModel.Option {
	return []einoModel.Option{
		einoModel.WithMaxTokens(8192),
		agenticopenai.WithResponsesReasoning(&openaiResponses.ReasoningParam{
			Effort:  openaiResponses.ReasoningEffortLow,
			Summary: openaiResponses.ReasoningSummaryDetailed,
		}),
		agenticopenai.WithResponsesMaxToolCalls(6),
		agenticopenai.WithResponsesParallelToolCalls(false),
		agenticopenai.WithResponsesServerTools([]*agenticopenai.ResponsesServerToolConfig{
			{
				WebSearch: &openaiResponses.WebSearchToolParam{
					Type: openaiResponses.WebSearchToolTypeWebSearch,
				},
			},
		}),
	}
}

func selectedAgenticProvider() string {
	provider := strings.TrimSpace(strings.ToLower(os.Getenv("AGENTIC_MODEL_PROVIDER")))
	if provider == "" {
		return providerARK
	}
	return provider
}

func requiredEnv(name string) (string, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return value, nil
}

func ptrOf[T any](v T) *T {
	return &v
}
