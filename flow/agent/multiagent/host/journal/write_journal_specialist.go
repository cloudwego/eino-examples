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
	"os"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ollama"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/multiagent/host"
	"github.com/cloudwego/eino/schema"
	"github.com/ollama/ollama/api"
)

// create a specialist who can append text to the right local journal file

func appendJournal(text string) error {
	// open or create the journal file for today
	// get today's date
	now := time.Now()
	dateStr := now.Format("2006-01-02")

	filePath, err := getJournalFilePath(dateStr) // Assume this function returns the path to today's journal file
	if err != nil {
		return err
	}

	file, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	// append text to file
	_, err = file.WriteString(text + "\n")
	if err != nil {
		return err
	}

	return nil
}

func getJournalFilePath(dateStr string) (string, error) {
	// generate the unique file path for today's journal file
	filePath := fmt.Sprintf("journal_%s.txt", dateStr)

	// find the file path for today's journal file
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// if not found, create the journal file with the file path
		file, err := os.Create(filePath)
		if err != nil {
			return "", err
		}
		file.Close()
	}

	// return the file path
	return filePath, nil
}

func newWriteJournalSpecialist(ctx context.Context, baseURL, model string) (*host.Specialist, error) {
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

	// use a chat model to rewrite user query to journal entry
	// for example, the user query might be:
	//
	// write: I got up at 7:00 in the morning.
	//
	// should be rewritten to:
	//
	// I got up at 7:00 in the morning.
	chain := compose.NewChain[[]*schema.Message, *schema.Message]()
	chain.AppendLambda(compose.InvokableLambda(func(ctx context.Context, input []*schema.Message) ([]*schema.Message, error) {
		systemMsg := &schema.Message{
			Role:    schema.System,
			Content: "You are responsible for preparing the user query for insertion into journal. The user's query is expected to contain the actual text the user want to write to journal, as well as convey the intention that this query should be written to journal. You job is to remove that intention from the user query, while preserving as much as possible the user's original query, and output ONLY the text to be written into journal",
		}
		return append([]*schema.Message{systemMsg}, input...), nil
	})).
		AppendChatModel(chatModel).
		AppendLambda(compose.InvokableLambda(func(ctx context.Context, input *schema.Message) (*schema.Message, error) {
			err := appendJournal(input.Content)
			if err != nil {
				return nil, err
			}
			return &schema.Message{
				Role:    schema.Assistant,
				Content: "Journal written successfully: " + input.Content,
			}, nil
		}))

	r, err := chain.Compile(ctx)
	if err != nil {
		return nil, err
	}

	return &host.Specialist{
		AgentMeta: host.AgentMeta{
			Name:        "write_journal",
			IntendedUse: "treat the user query as a sentence of a journal entry, append it to the right journal file",
		},
		Invokable: func(ctx context.Context, input []*schema.Message, opts ...agent.AgentOption) (*schema.Message, error) {
			return r.Invoke(ctx, input, agent.GetComposeOptions(opts...)...)
		},
	}, nil
}
