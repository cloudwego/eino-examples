/*
 * Copyright 2024 CloudWeGo Authors
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
	"io"
	"os"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/multiagent/host"
	"github.com/cloudwego/eino/schema"
	"github.com/ollama/ollama/api"
)

// search journal: user ask a question, this specialist load today's journal and ground its answer onto it.

func loadJournal(date string) (string, error) {
	filePath := fmt.Sprintf("journal_%s.txt", date)

	// open the file
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// read the file content
	content, err := io.ReadAll(file)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

func newAnswerWithJournalSpecialist(ctx context.Context, baseURL, model string) (*host.Specialist, error) {
	chatModel, err := ollama.NewChatModel(ctx, &ollama.ChatModelConfig{
		BaseURL: baseURL,
		Model:   model,

		Options: &api.Options{
			Temperature: 0.000001,
		},
	})
	if err != nil {
		return nil, err
	}

	// create a graph: load journal and user query -> chat template -> chat model -> answer

	graph := compose.NewGraph[[]*schema.Message, *schema.Message]()

	if err = graph.AddLambdaNode("journal_loader", compose.InvokableLambda(func(ctx context.Context, input []*schema.Message) (string, error) {
		now := time.Now()
		dateStr := now.Format("2006-01-02")

		return loadJournal(dateStr)
	}), compose.WithOutputKey("journal")); err != nil {
		return nil, err
	}

	if err = graph.AddLambdaNode("query_extractor", compose.InvokableLambda(func(ctx context.Context, input []*schema.Message) (string, error) {
		return input[len(input)-1].Content, nil
	}), compose.WithOutputKey("query")); err != nil {
		return nil, err
	}

	systemTpl := `Answer user's query based on journal content: {journal}'`
	chatTpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage(systemTpl),
		schema.UserMessage("{query}"),
	)
	if err = graph.AddChatTemplateNode("template", chatTpl); err != nil {
		return nil, err
	}

	if err = graph.AddChatModelNode("model", chatModel); err != nil {
		return nil, err
	}

	if err = graph.AddEdge("journal_loader", "template"); err != nil {
		return nil, err
	}

	if err = graph.AddEdge("query_extractor", "template"); err != nil {
		return nil, err
	}

	if err = graph.AddEdge("template", "model"); err != nil {
		return nil, err
	}

	if err = graph.AddEdge(compose.START, "journal_loader"); err != nil {
		return nil, err
	}

	if err = graph.AddEdge(compose.START, "query_extractor"); err != nil {
		return nil, err
	}

	if err = graph.AddEdge("model", compose.END); err != nil {
		return nil, err
	}

	r, err := graph.Compile(ctx)
	if err != nil {
		return nil, err
	}

	return &host.Specialist{
		AgentMeta: host.AgentMeta{
			Name:        "answer_with_journal",
			IntendedUse: "load journal content and answer user's question with it",
		},
		Invokable: func(ctx context.Context, input []*schema.Message, opts ...agent.AgentOption) (*schema.Message, error) {
			return r.Invoke(ctx, input, agent.GetComposeOptions(opts...)...)
		},
	}, nil
}
