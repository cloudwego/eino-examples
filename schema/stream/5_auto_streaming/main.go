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
	"io"
	"strings"
	"time"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/internal/logs"
)

type Document struct {
	Title   string
	Content string
}

type ProcessedDocument struct {
	Title     string
	WordCount int
	Preview   string
}

func main() {
	ctx := context.Background()

	parseMetadata := compose.InvokableLambda(func(ctx context.Context, doc *Document) (*Document, error) {
		logs.Infof("[Node 1 - Invoke] Parsing document metadata: %q", doc.Title)
		return doc, nil
	})

	generateChunks := compose.StreamableLambda(func(ctx context.Context, doc *Document) (*schema.StreamReader[string], error) {
		logs.Infof("[Node 2 - Stream] Generating content chunks for: %q", doc.Title)

		words := strings.Fields(doc.Content)
		sr, sw := schema.Pipe[string](1)

		go func() {
			defer sw.Close()
			chunkSize := 3
			for i := 0; i < len(words); i += chunkSize {
				end := i + chunkSize
				if end > len(words) {
					end = len(words)
				}
				chunk := strings.Join(words[i:end], " ")
				time.Sleep(50 * time.Millisecond)
				sw.Send(chunk, nil)
				logs.Infof("[Node 2 - Stream] Emitted chunk: %q", chunk)
			}
		}()

		return sr, nil
	})

	aggregateContent := compose.CollectableLambda(func(ctx context.Context, chunks *schema.StreamReader[string]) (*ProcessedDocument, error) {
		logs.Infof("[Node 3 - Collect] Aggregating content chunks...")

		var allChunks []string
		for {
			chunk, err := chunks.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				return nil, err
			}
			allChunks = append(allChunks, chunk)
		}

		fullContent := strings.Join(allChunks, " ")
		wordCount := len(strings.Fields(fullContent))

		preview := fullContent
		if len(preview) > 50 {
			preview = preview[:50] + "..."
		}

		result := &ProcessedDocument{
			Title:     "Processed",
			WordCount: wordCount,
			Preview:   preview,
		}

		logs.Infof("[Node 3 - Collect] Aggregation complete: %d words", wordCount)
		return result, nil
	})

	graph := compose.NewGraph[*Document, *ProcessedDocument]()
	_ = graph.AddLambdaNode("parse", parseMetadata)
	_ = graph.AddLambdaNode("generate", generateChunks)
	_ = graph.AddLambdaNode("aggregate", aggregateContent)
	_ = graph.AddEdge(compose.START, "parse")
	_ = graph.AddEdge("parse", "generate")
	_ = graph.AddEdge("generate", "aggregate")
	_ = graph.AddEdge("aggregate", compose.END)

	runner, err := graph.Compile(ctx)
	if err != nil {
		logs.Fatalf("Failed to compile graph: %v", err)
	}

	doc := &Document{
		Title: "Eino Framework Guide",
		Content: `Eino is a powerful LLM application development framework in Golang. 
It provides rich capabilities for building AI applications including atomic components, 
integrated components, component orchestration, and aspect extensions. 
The framework helps developers create well-architected, maintainable, and highly available AI applications.`,
	}

	logs.Infof("Input document: %q", doc.Title)
	logs.Infof("Content length: %d characters", len(doc.Content))
	logs.Infof("")
	logs.Infof("Pipeline: Invoke → Stream → Collect")
	logs.Infof("Eino auto-converts between stream/non-stream at node boundaries")
	logs.Infof("")

	result, err := runner.Invoke(ctx, doc)
	if err != nil {
		logs.Fatalf("Failed to process document: %v", err)
	}

	logs.Infof("")
	logs.Infof("Result: %s", fmt.Sprintf("Title=%q, Words=%d, Preview=%q",
		result.Title, result.WordCount, result.Preview))
}
