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

package model

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	cbutils "github.com/cloudwego/eino/utils/callbacks"
	arkModel "github.com/volcengine/volcengine-go-sdk/service/arkruntime/model"
)

func NewChatModel() model.ToolCallingChatModel {
	modelType := strings.ToLower(os.Getenv("MODEL_TYPE"))

	// Create Ark ChatModel when MODEL_TYPE is "ark"
	if modelType == "ark" {
		cm, err := ark.NewChatModel(context.Background(), &ark.ChatModelConfig{
			// Add Ark-specific configuration from environment variables
			APIKey:  os.Getenv("ARK_API_KEY"),
			Model:   os.Getenv("ARK_MODEL"),
			BaseURL: os.Getenv("ARK_BASE_URL"),
			Thinking: &arkModel.Thinking{
				Type: arkModel.ThinkingTypeDisabled,
			},
		})
		if err != nil {
			log.Fatalf("ark.NewChatModel failed: %v", err)
		}
		return cm
	}

	// Create OpenAI ChatModel (default)
	cm, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   os.Getenv("OPENAI_MODEL"),
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
		ByAzure: func() bool {
			return os.Getenv("OPENAI_BY_AZURE") == "true"
		}(),
	})
	if err != nil {
		log.Fatalf("openai.NewChatModel failed: %v", err)
	}
	return cm
}

func GetInputLoggerCallback() callbacks.Handler {
	return cbutils.NewHandlerHelper().ChatModel(&cbutils.ModelCallbackHandler{
		OnStart: func(ctx context.Context, info *callbacks.RunInfo, input *model.CallbackInput) context.Context {
			time.Sleep(20 * time.Second)
			fmt.Printf("\n========================================\n")
			fmt.Printf("[ChatModel Input] Agent: %s\n", info.Name)
			fmt.Printf("========================================\n")
			for i, msg := range input.Messages {
				fmt.Printf("  Message %d [%s]: %s\n", i+1, msg.Role, msg.Content)
				if len(msg.ToolCalls) > 0 {
					fmt.Printf("    Tool Calls: %d\n", len(msg.ToolCalls))
					for j, tc := range msg.ToolCalls {
						fmt.Printf("      %d. %s: %s\n", j+1, tc.Function.Name, tc.Function.Arguments)
					}
				}
			}
			fmt.Printf("========================================\n\n")
			return ctx
		},
	}).Handler()
}
