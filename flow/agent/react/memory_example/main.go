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
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/flow/agent/react/memory_example/memory"
)

func main() {
	ctx := context.Background()

	apiKey := os.Getenv("OPENAI_API_KEY")
	modelName := os.Getenv("OPENAI_MODEL")
	baseURL := os.Getenv("OPENAI_BASE_URL")
	isAzure := os.Getenv("OPENAI_BY_AZURE") == "true"

	model, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{APIKey: apiKey, Model: modelName, BaseURL: baseURL, ByAzure: isAzure})
	if err != nil {
		panic(err)
	}

	// System prompt is injected at runtime and not persisted.
	sys := "You are a concise assistant. Maintain context across turns."

	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		Model: model,
		MessageModifier: func(_ context.Context, input []*schema.Message) []*schema.Message {
			return append([]*schema.Message{schema.SystemMessage(sys)}, input...)
		},
	})
	if err != nil {
		panic(err)
	}

	// Choose your store: InMemoryStore (default) or RedisStore (see README).
	store := memory.NewInMemoryStore()
	sessionID := "session:demo"

	verifyGobRoundTrip()

	run := func(turn string) {
		// 1) restore prior messages, 2) append new input, 3) call agent, 4) persist with output
		prev, _ := store.Read(ctx, sessionID)
		eff := append(prev, schema.UserMessage(turn))
		msg, err := agent.Generate(ctx, eff)
		if err != nil {
			panic(err)
		}
		fmt.Printf("history_before=%d after=%d\n", len(prev), len(eff)+1)
		fmt.Println(msg.Content)
		_ = store.Write(ctx, sessionID, append(eff, msg))

		hits, _ := store.Query(ctx, sessionID, "AI", 3)
		fmt.Printf("query_hits=%d\n", len(hits))
	}

	run("Hello, summarize AI briefly.")
	run("Add two more details.")
}

func verifyGobRoundTrip() {
	msgs := []*schema.Message{
		schema.UserMessage("a"),
		schema.AssistantMessage("b", nil),
	}
	// Round-trip serialize/deserialize to validate gob setup.
	b, err := memory.EncodeMessages(msgs)
	if err != nil {
		panic(err)
	}
	out, err := memory.DecodeMessages(b)
	if err != nil {
		panic(err)
	}
	fmt.Printf("gob_round_trip=%d\n", len(out))
}
