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

package chatmodel

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/agenticark"
	"github.com/cloudwego/eino-ext/components/model/agenticopenai"
	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	einomodel "github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
	arkModel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

// NewModel creates a model matching M. MESSAGE_KIND selects M at the caller
// boundary; MODEL_TYPE still selects the provider (OpenAI by default, Ark when
// MODEL_TYPE=ark).
func NewModel[M adk.MessageType](ctx context.Context) (einomodel.BaseModel[M], error) {
	var zero M
	switch any(zero).(type) {
	case *schema.AgenticMessage:
		cm, err := newAgenticModel(ctx)
		if err != nil {
			return nil, err
		}
		return any(cm).(einomodel.BaseModel[M]), nil
	default:
		cm, err := newChatModel(ctx)
		if err != nil {
			return nil, err
		}
		return any(cm).(einomodel.BaseModel[M]), nil
	}
}

func newChatModel(ctx context.Context) (einomodel.ToolCallingChatModel, error) {
	modelType := strings.ToLower(os.Getenv("MODEL_TYPE"))
	if modelType == "ark" {
		return ark.NewChatModel(ctx, &ark.ChatModelConfig{
			APIKey:  os.Getenv("ARK_API_KEY"),
			Model:   firstEnv("ARK_MODEL", "ARK_MODEL_ID"),
			BaseURL: os.Getenv("ARK_BASE_URL"),
			Thinking: &arkModel.Thinking{
				Type: arkModel.ThinkingTypeDisabled,
			},
		})
	}

	return openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   firstEnv("OPENAI_MODEL", "OPENAI_MODEL_ID"),
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
		ByAzure: os.Getenv("OPENAI_BY_AZURE") == "true",
	})
}

func newAgenticModel(ctx context.Context) (einomodel.AgenticModel, error) {
	modelType := strings.ToLower(os.Getenv("MODEL_TYPE"))
	timeout := 3 * time.Minute
	if modelType == "ark" {
		return agenticark.New(ctx, &agenticark.Config{
			APIKey:  os.Getenv("ARK_API_KEY"),
			Model:   firstEnv("ARK_MODEL", "ARK_MODEL_ID"),
			BaseURL: os.Getenv("ARK_BASE_URL"),
			Timeout: &timeout,
		})
	}

	return agenticopenai.New(ctx, &agenticopenai.Config{
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   firstEnv("OPENAI_MODEL", "OPENAI_MODEL_ID"),
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
		ByAzure: os.Getenv("OPENAI_BY_AZURE") == "true",
		Timeout: &timeout,
	})
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}
