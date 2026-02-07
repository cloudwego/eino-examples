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
	"log"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/adk/common/model"
	"github.com/cloudwego/eino-examples/adk/common/prints"
	"github.com/cloudwego/eino-examples/adk/common/tool/graphtool"
)

type SummarizeInput struct {
	Document string `json:"document" jsonschema_description:"The document text to summarize"`
	MaxWords int    `json:"max_words" jsonschema_description:"Maximum number of words in the summary (default: 100)"`
}

type SummarizeOutput struct {
	Summary   string   `json:"summary"`
	KeyPoints []string `json:"key_points"`
	WordCount int      `json:"word_count"`
}

func NewSummarizeTool(ctx context.Context) (tool.InvokableTool, error) {
	cm := model.NewChatModel()

	extractKeyPointsPrompt := prompt.FromMessages(schema.FString,
		schema.SystemMessage(`You are an expert at extracting key points from documents.
Extract the main key points from the following document. Return them as a numbered list.
Be concise and focus on the most important information.`),
		schema.UserMessage(`Document:
{document}

Extract the key points:`))

	condenseSummaryPrompt := prompt.FromMessages(schema.FString,
		schema.SystemMessage(`You are an expert summarizer.
Given the key points below, create a coherent summary in approximately {max_words} words.
The summary should flow naturally and capture the essence of the original content.`),
		schema.UserMessage(`Key Points:
{key_points}

Create a summary in approximately {max_words} words:`))

	fullChain := compose.NewChain[*SummarizeInput, *SummarizeOutput]()
	fullChain.
		AppendLambda(compose.InvokableLambda(func(ctx context.Context, input *SummarizeInput) (map[string]any, error) {
			maxWords := input.MaxWords
			if maxWords <= 0 {
				maxWords = 100
			}
			return map[string]any{
				"document":  input.Document,
				"max_words": maxWords,
			}, nil
		})).
		AppendChatTemplate(extractKeyPointsPrompt).
		AppendChatModel(cm).
		AppendLambda(compose.InvokableLambda(func(ctx context.Context, msg *schema.Message) (map[string]any, error) {
			keyPointsContent := msg.Content
			keyPoints := parseKeyPoints(keyPointsContent)
			ctx = context.WithValue(ctx, keyPointsCtxKey, keyPoints)
			return map[string]any{
				"key_points": keyPointsContent,
				"max_words":  100,
			}, nil
		})).
		AppendChatTemplate(condenseSummaryPrompt).
		AppendChatModel(cm).
		AppendLambda(compose.InvokableLambda(func(ctx context.Context, msg *schema.Message) (*SummarizeOutput, error) {
			return &SummarizeOutput{
				Summary:   msg.Content,
				KeyPoints: []string{},
				WordCount: countWords(msg.Content),
			}, nil
		}))

	return graphtool.NewInvokableGraphTool[*SummarizeInput, *SummarizeOutput](
		fullChain,
		"summarize_document",
		"Summarize a document by extracting key points and creating a coherent summary. Returns the summary, key points, and word count.",
	)
}

type ctxKey string

const keyPointsCtxKey ctxKey = "key_points_list"

func parseKeyPoints(content string) []string {
	return []string{content}
}

func countWords(s string) int {
	words := 0
	inWord := false
	for _, r := range s {
		if r == ' ' || r == '\n' || r == '\t' {
			inWord = false
		} else if !inWord {
			inWord = true
			words++
		}
	}
	return words
}

func main() {
	ctx := context.Background()

	summarizeTool, err := NewSummarizeTool(ctx)
	if err != nil {
		log.Fatalf("failed to create summarize tool: %v", err)
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "DocumentAssistant",
		Description: "An assistant that can summarize documents",
		Instruction: `You are a helpful document assistant.
When the user provides a document or asks you to summarize something, use the summarize_document tool.
Always provide the full document text to the tool.`,
		Model: model.NewChatModel(),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{summarizeTool},
			},
		},
	})
	if err != nil {
		log.Fatalf("failed to create agent: %v", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           agent,
	})

	sampleDocument := `Artificial Intelligence (AI) has transformed numerous industries over the past decade. 
In healthcare, AI systems now assist doctors in diagnosing diseases by analyzing medical images with remarkable accuracy. 
Machine learning algorithms can detect patterns in X-rays and MRIs that might escape human observation.

In finance, AI-powered trading systems execute millions of transactions daily, analyzing market trends and making 
split-second decisions. Banks use AI for fraud detection, identifying suspicious patterns in transaction data.

The transportation sector has seen revolutionary changes with autonomous vehicles. Self-driving cars use 
computer vision and deep learning to navigate roads safely. Companies like Tesla, Waymo, and others are 
racing to perfect this technology.

However, AI also raises important ethical concerns. Issues of bias in AI systems, job displacement, 
and privacy concerns require careful consideration. As AI becomes more powerful, society must develop 
frameworks to ensure its responsible use.`

	query := fmt.Sprintf("Please summarize this document in about 50 words:\n\n%s", sampleDocument)

	iter := runner.Query(ctx, query)

	fmt.Println("=== Document Summarization Example ===")
	fmt.Println()

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			log.Fatalf("error: %v", event.Err)
		}
		prints.Event(event)
	}
}
