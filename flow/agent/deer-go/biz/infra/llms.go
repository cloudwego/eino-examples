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

package infra

import (
	"context"

	"github.com/cloudwego/eino-ext/components/model/openai"
	openai3 "github.com/cloudwego/eino-ext/libs/acl/openai"
	"github.com/getkin/kin-openapi/openapi3gen"

	"github.com/cloudwego/eino-examples/flow/agent/deer-go/biz/model"
	"github.com/cloudwego/eino-examples/flow/agent/deer-go/conf"
)

var (
	ChatModel *openai.ChatModel
	PlanModel *openai.ChatModel
)

func InitModel() {
	config := &openai.ChatModelConfig{
		BaseURL: conf.Config.Model.BaseURL,
		APIKey:  conf.Config.Model.APIKey,
		Model:   conf.Config.Model.DefaultModel,
	}
	ChatModel, _ = openai.NewChatModel(context.Background(), config)
	planSchema, _ := openapi3gen.NewSchemaRefForValue(&model.Plan{}, nil)

	planconfig := &openai.ChatModelConfig{
		BaseURL: conf.Config.Model.BaseURL,
		APIKey:  conf.Config.Model.APIKey,
		Model:   conf.Config.Model.DefaultModel,
		ResponseFormat: &openai3.ChatCompletionResponseFormat{
			Type: openai3.ChatCompletionResponseFormatTypeJSONSchema,
			JSONSchema: &openai3.ChatCompletionResponseFormatJSONSchema{
				Name:   "plan",
				Strict: false,
				Schema: planSchema.Value,
			},
		},
	}
	PlanModel, _ = openai.NewChatModel(context.Background(), planconfig)
}
