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
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/agenticark"
	"github.com/cloudwego/eino-ext/components/model/agenticopenai"
	"github.com/cloudwego/eino/adk"
	einoModel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	openaiResponses "github.com/openai/openai-go/v3/responses"
	arkResponses "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model/responses"
)

const (
	providerArk    = "ark"
	providerOpenAI = "openai"

	travelRouteToolName = "travel_route"
	tokenLimitReason    = "max_output_tokens"

	defaultRequestTimeout = 3 * time.Minute
)

var retryMaxTokens = []int{4096, 8192, 16384}

func main() {
	log.SetFlags(0)

	provider := selectedProvider()
	initialMaxTokens := 128

	ctx, cancel := context.WithTimeout(context.Background(), defaultRequestTimeout)
	defer cancel()

	agenticModel, inspect, modelID, err := newAgenticModel(ctx, provider)
	if err != nil {
		log.Fatal(err)
	}

	agent, err := newTravelAgent(ctx, agenticModel, inspect)
	if err != nil {
		log.Fatal(err)
	}

	printRunHeader(provider, modelID, initialMaxTokens)

	runner := adk.NewTypedRunner[*schema.AgenticMessage](adk.TypedRunnerConfig[*schema.AgenticMessage]{
		Agent: agent,
	})

	messageIndex := 1
	input := schema.UserAgenticMessage(userPrompt())
	printAgenticMessage(messageIndex, input)

	opts := []einoModel.Option{einoModel.WithMaxTokens(initialMaxTokens)}
	if provider == providerOpenAI {
		opts = append(opts, agenticopenai.WithStore(true))
	}

	iter := runner.Run(ctx,
		[]*schema.AgenticMessage{input},
		adk.WithChatModelOptions(opts),
	)

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if handled := printRetryEvent(event.Err); handled {
			continue
		}
		if event.Err != nil {
			log.Fatalf("agent event error: %v", event.Err)
		}

		msg, _, err := adk.TypedGetMessage(event)
		if err != nil {
			log.Fatalf("read agent output: %v", err)
		}
		if shouldPrintAgenticMessage(msg) {
			messageIndex++
			printAgenticMessage(messageIndex, msg)
		}
	}
}

func printRetryEvent(err error) bool {
	var willRetry *adk.WillRetryError
	if !errors.As(err, &willRetry) {
		return false
	}
	fmt.Printf("[retry-event] attempt=%d reason=%v\n", willRetry.RetryAttempt, willRetry.RejectReason())
	return true
}

func newAgenticModel(ctx context.Context, provider string) (einoModel.AgenticModel, responseInspector, string, error) {
	switch provider {
	case providerArk:
		modelID, err := requiredEnv("ARK_MODEL_ID")
		if err != nil {
			return nil, nil, "", err
		}
		enablePassBackReasoning := false
		m, err := agenticark.New(ctx, &agenticark.Config{
			APIKey:                  os.Getenv("ARK_API_KEY"),
			Model:                   modelID,
			BaseURL:                 os.Getenv("ARK_BASE_URL"),
			Reasoning:               &arkResponses.ResponsesReasoning{Effort: arkResponses.ReasoningEffort_medium},
			EnablePassBackReasoning: &enablePassBackReasoning,
		})
		return m, arkStatusAndReason, modelID, err
	case providerOpenAI:
		apiKey, err := requiredEnv("OPENAI_API_KEY")
		if err != nil {
			return nil, nil, "", err
		}
		modelID, err := requiredEnv("OPENAI_MODEL_ID")
		if err != nil {
			return nil, nil, "", err
		}
		timeout := defaultRequestTimeout
		m, err := agenticopenai.New(ctx, &agenticopenai.Config{
			APIKey:  apiKey,
			Model:   modelID,
			BaseURL: os.Getenv("OPENAI_BASE_URL"),
			Timeout: &timeout,
			Include: []openaiResponses.ResponseIncludable{
				openaiResponses.ResponseIncludableWebSearchCallActionSources,
				openaiResponses.ResponseIncludableReasoningEncryptedContent,
			},
		})
		return m, openAIStatusAndReason, modelID, err
	default:
		return nil, nil, "", fmt.Errorf("unsupported AGENTIC_MODEL_PROVIDER %q, expected %q or %q",
			provider, providerOpenAI, providerArk)
	}
}

func selectedProvider() string {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("AGENTIC_MODEL_PROVIDER")))
	if provider == "" {
		return providerOpenAI
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

func printRunHeader(provider, modelID string, initialMaxTokens int) {
	fmt.Printf("provider=%s model=%s initial_max_output_tokens=%d retry_max_output_tokens=%v\n",
		provider, modelID, initialMaxTokens, retryMaxTokens)
}

func printAgenticMessage(index int, msg *schema.AgenticMessage) {
	if msg == nil {
		return
	}
	fmt.Printf("\n--- AgenticMessage #%d ---\n", index)
	fmt.Print(msg.String())
}

func shouldPrintAgenticMessage(msg *schema.AgenticMessage) bool {
	if msg == nil {
		return false
	}
	for _, block := range msg.ContentBlocks {
		if block == nil {
			continue
		}
		if block.Type != schema.ContentBlockTypeFunctionToolResult {
			return true
		}
	}
	return len(msg.ContentBlocks) == 0
}

func userPrompt() string {
	return strings.TrimSpace(`
请为第一次去杭州的情侣规划一个 3 天 2 晚旅游路线。
请先调用本地旅游路线工具，然后输出一份中文路线，包含：每日安排、交通建议、餐饮建议、预算范围、注意事项。
`)
}
