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
	"fmt"
	"os"
	"time"

	"github.com/cloudwego/eino-ext/components/model/agenticark"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model/responses"
)

const (
	defaultBaseURL  = "https://ark.cn-beijing.volces.com/api/v3"
	defaultImageURL = "https://data.tibetcn.com/pictures/the-west-lake-map.jpg"
)

type appConfig struct {
	apiKey string

	model    string
	baseURL  string
	imageURL string
}

func loadConfig() (*appConfig, error) {
	apiKey := os.Getenv("ARK_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ARK_API_KEY is required")
	}

	modelID := firstNonEmpty(os.Getenv("ARK_MODEL_ID"), os.Getenv("ARK_MODEL"))
	if modelID == "" {
		return nil, fmt.Errorf("ARK_MODEL_ID or ARK_MODEL is required")
	}

	return &appConfig{
		apiKey:   apiKey,
		model:    modelID,
		baseURL:  firstNonEmpty(os.Getenv("ARK_BASE_URL"), defaultBaseURL),
		imageURL: defaultImageURL,
	}, nil
}

func newAgenticArkModel(ctx context.Context, cfg *appConfig, modelID string, opts ...model.Option) (model.AgenticModel, error) {
	timeout := 2 * time.Minute
	base, err := agenticark.New(ctx, &agenticark.Config{
		APIKey:  cfg.apiKey,
		BaseURL: cfg.baseURL,
		Model:   modelID,
		Timeout: &timeout,
		Thinking: &responses.ResponsesThinking{
			Type: responses.ThinkingType_disabled.Enum(),
		},
	})
	if err != nil {
		return nil, err
	}
	if len(opts) == 0 {
		return base, nil
	}
	return &modelWithOptions{base: base, opts: opts}, nil
}

type modelWithOptions struct {
	base model.AgenticModel
	opts []model.Option
}

func (m *modelWithOptions) Generate(ctx context.Context, input []*schema.AgenticMessage, opts ...model.Option) (*schema.AgenticMessage, error) {
	return m.base.Generate(ctx, input, appendModelOptions(m.opts, opts)...)
}

func (m *modelWithOptions) Stream(ctx context.Context, input []*schema.AgenticMessage, opts ...model.Option) (*schema.StreamReader[*schema.AgenticMessage], error) {
	return m.base.Stream(ctx, input, appendModelOptions(m.opts, opts)...)
}

func appendModelOptions(left, right []model.Option) []model.Option {
	if len(left) == 0 {
		return right
	}
	combined := make([]model.Option, 0, len(left)+len(right))
	combined = append(combined, left...)
	combined = append(combined, right...)
	return combined
}

func deepThinkingOptions() []model.Option {
	return []model.Option{
		agenticark.WithThinking(&responses.ResponsesThinking{
			Type: responses.ThinkingType_enabled.Enum(),
		}),
		agenticark.WithReasoning(&responses.ResponsesReasoning{
			Effort: responses.ReasoningEffort_high,
		}),
	}
}

func sessionCacheOption() model.Option {
	expireAt := time.Now().Add(30 * time.Minute).Unix()
	return agenticark.WithCache(&agenticark.CacheOption{
		SessionCache: &agenticark.SessionCacheConfig{
			EnableCache: true,
			ExpireAtSec: expireAt,
		},
	})
}

func previousResponseCacheOption(responseID string) model.Option {
	expireAt := time.Now().Add(30 * time.Minute).Unix()
	return agenticark.WithCache(&agenticark.CacheOption{
		HeadPreviousResponseID: &responseID,
		SessionCache: &agenticark.SessionCacheConfig{
			EnableCache: true,
			ExpireAtSec: expireAt,
		},
	})
}

func ptrOf[T any](v T) *T {
	return &v
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
