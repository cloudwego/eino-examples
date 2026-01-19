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
	"strings"
	"time"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/adk/common/model"
	"github.com/cloudwego/eino-examples/adk/common/prints"
	"github.com/cloudwego/eino-examples/adk/common/tool/graphtool"
)

type ResearchInput struct {
	Query string `json:"query" jsonschema_description:"The research topic or question to investigate"`
}

func mockWebSearch(ctx context.Context, query string) (string, error) {
	time.Sleep(100 * time.Millisecond)
	return fmt.Sprintf(`Web Search Results for "%s":
1. Wikipedia: %s is a widely discussed topic with multiple perspectives...
2. News Article: Recent developments in %s show promising trends...
3. Research Paper: A comprehensive study on %s reveals key insights...`, query, query, query, query), nil
}

func mockKnowledgeBaseSearch(ctx context.Context, query string) (string, error) {
	time.Sleep(80 * time.Millisecond)
	return fmt.Sprintf(`Knowledge Base Results for "%s":
- Internal Doc #1: Company guidelines related to %s...
- Internal Doc #2: Best practices for handling %s...
- FAQ Entry: Common questions about %s answered...`, query, query, query, query), nil
}

func mockLocalFileSearch(ctx context.Context, query string) (string, error) {
	time.Sleep(50 * time.Millisecond)
	return fmt.Sprintf(`Local File Results for "%s":
- notes/research_%s.md: Personal notes on %s...
- docs/guide_%s.txt: Step-by-step guide for %s...`, query, strings.ReplaceAll(query, " ", "_"), query, strings.ReplaceAll(query, " ", "_"), query), nil
}

type searchResults struct {
	Query        string
	WebResults   string
	KBResults    string
	LocalResults string
}

func NewResearchTool(ctx context.Context) (tool.StreamableTool, error) {
	cm := model.NewChatModel()

	synthesizePrompt := prompt.FromMessages(schema.FString,
		schema.SystemMessage(`You are a research analyst. Synthesize the following search results from multiple sources into a coherent summary.
Focus on the most relevant and reliable information. Identify any conflicting information across sources.
Be concise but comprehensive. Output the summary directly without any JSON formatting.`),
		schema.UserMessage(`Research Query: {query}

Web Search Results:
{web_results}

Knowledge Base Results:
{kb_results}

Local File Results:
{local_results}

Please synthesize these results into a comprehensive summary:`))

	graph := compose.NewGraph[*ResearchInput, *schema.Message]()

	_ = graph.AddLambdaNode("parallel_search", compose.InvokableLambda(func(ctx context.Context, input *ResearchInput) (*searchResults, error) {
		fmt.Println("  [Graph] Starting parallel searches...")

		type result struct {
			source string
			data   string
			err    error
		}

		resultCh := make(chan result, 3)

		go func() {
			data, err := mockWebSearch(ctx, input.Query)
			resultCh <- result{source: "web", data: data, err: err}
		}()

		go func() {
			data, err := mockKnowledgeBaseSearch(ctx, input.Query)
			resultCh <- result{source: "kb", data: data, err: err}
		}()

		go func() {
			data, err := mockLocalFileSearch(ctx, input.Query)
			resultCh <- result{source: "local", data: data, err: err}
		}()

		results := &searchResults{Query: input.Query}
		for i := 0; i < 3; i++ {
			r := <-resultCh
			if r.err != nil {
				return nil, r.err
			}
			switch r.source {
			case "web":
				results.WebResults = r.data
				fmt.Println("  [Graph] Web search completed")
			case "kb":
				results.KBResults = r.data
				fmt.Println("  [Graph] Knowledge base search completed")
			case "local":
				results.LocalResults = r.data
				fmt.Println("  [Graph] Local file search completed")
			}
		}

		fmt.Println("  [Graph] All searches completed, preparing synthesis...")
		return results, nil
	}))

	_ = graph.AddLambdaNode("prepare_prompt_input", compose.InvokableLambda(func(ctx context.Context, results *searchResults) (map[string]any, error) {
		return map[string]any{
			"query":         results.Query,
			"web_results":   results.WebResults,
			"kb_results":    results.KBResults,
			"local_results": results.LocalResults,
		}, nil
	}))

	_ = graph.AddChatTemplateNode("prepare_prompt", synthesizePrompt)

	_ = graph.AddChatModelNode("synthesize", cm)

	_ = graph.AddEdge(compose.START, "parallel_search")
	_ = graph.AddEdge("parallel_search", "prepare_prompt_input")
	_ = graph.AddEdge("prepare_prompt_input", "prepare_prompt")
	_ = graph.AddEdge("prepare_prompt", "synthesize")
	_ = graph.AddEdge("synthesize", compose.END)

	return graphtool.NewStreamableGraphTool[*ResearchInput, *schema.Message](
		graph,
		"research_topic",
		"Research a topic by querying multiple sources (web, knowledge base, local files) in parallel and synthesizing the results. Returns a streaming summary directly.",
	)
}

func main() {
	ctx := context.Background()

	researchTool, err := NewResearchTool(ctx)
	if err != nil {
		log.Fatalf("failed to create research tool: %v", err)
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "ResearchAssistant",
		Description: "An assistant that can research topics using multiple sources",
		Instruction: `You are a helpful research assistant.
When the user asks about a topic or wants to learn something, use the research_topic tool to gather information from multiple sources.
The tool will stream the research results directly to the user.`,
		Model: model.NewChatModel(),
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{researchTool},
			},
			ReturnDirectly: map[string]bool{
				"research_topic": true,
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

	query := "What are the best practices for building microservices?"

	iter := runner.Query(ctx, query)

	fmt.Println("=== Multi-Source Research Example (using compose.Graph + StreamableGraphTool) ===")
	fmt.Println()
	fmt.Println("This example demonstrates:")
	fmt.Println("1. StreamableGraphTool with compose.Graph")
	fmt.Println("2. Parallel search execution within a graph node")
	fmt.Println("3. Streaming output from ChatModel via ReturnDirectly")
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
