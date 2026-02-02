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
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/summarization"
	"github.com/cloudwego/eino/schema"
)

func loadMessagesFromJSONFile(fileName string) ([]adk.Message, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("get current working directory failed: %w", err)
	}

	filePath := filepath.Join(wd, "adk", "middlewares", "summarization", fileName)

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("read messages json failed: %w", err)
	}

	var msgs []*schema.Message
	if err := json.Unmarshal(data, &msgs); err != nil {
		return nil, fmt.Errorf("unmarshal into []*schema.Message failed: %w", err)
	}

	return msgs, nil
}

func main() {
	ctx := context.Background()

	adk.SetLanguage(adk.LanguageChinese)

	cm, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey: os.Getenv("ARK_API_KEY"),
		Model:  os.Getenv("ARK_MODEL_ID"),
	})
	if err != nil {
		log.Fatalf("failed to create chat model: %v", err)
	}

	mw, err := summarization.New(ctx, &summarization.Config{
		Model: cm,
		Trigger: &summarization.TriggerCondition{
			ContextTokens: 500,
		},
		TranscriptFilePath: "/adk/middlewares/summarization/history.json",
	})
	if err != nil {
		log.Fatalf("failed to create summarization middleware: %v", err)
	}

	msgs, err := loadMessagesFromJSONFile("history.json")
	if err != nil {
		log.Fatalf("failed to load messages: %v", err)
	}

	state := &adk.ChatModelAgentState{Messages: msgs}

	_, newState, err := mw.BeforeModelRewriteState(ctx, state, &adk.ModelContext{})
	if err != nil {
		log.Fatalf("failed to summarize messages: %v", err)
	}

	output, err := cm.Generate(ctx, newState.Messages)
	if err != nil {
		log.Fatalf("failed to generate messages: %v", err)
	}

	history := append(newState.Messages, output)

	s, err := sonic.MarshalIndent(history, "", "  ")
	if err != nil {
		log.Fatalf("failed to marshal messages: %v", err)
	}

	fmt.Println(string(s))
}
