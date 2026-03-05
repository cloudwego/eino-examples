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
	"strings"

	"github.com/cloudwego/eino/compose"

	"github.com/cloudwego/eino-examples/internal/logs"
)

func main() {
	ctx := context.Background()

	summarizer := compose.InvokableLambda(func(ctx context.Context, text string) (string, error) {
		logs.Infof("Summarizing text of length %d", len(text))

		words := strings.Fields(text)
		if len(words) <= 10 {
			return text, nil
		}

		summary := strings.Join(words[:10], " ") + "..."
		return fmt.Sprintf("Summary (%d words → 10): %s", len(words), summary), nil
	})

	wordCounter := compose.InvokableLambda(func(ctx context.Context, text string) (int, error) {
		count := len(strings.Fields(text))
		logs.Infof("Counted %d words", count)
		return count, nil
	})

	graph := compose.NewGraph[string, int]()
	_ = graph.AddLambdaNode("summarizer", summarizer)
	_ = graph.AddLambdaNode("counter", wordCounter)
	_ = graph.AddEdge(compose.START, "summarizer")
	_ = graph.AddEdge("summarizer", "counter")
	_ = graph.AddEdge("counter", compose.END)

	runner, err := graph.Compile(ctx)
	if err != nil {
		logs.Fatalf("Failed to compile graph: %v", err)
	}

	document := `Eino is a powerful LLM application development framework in Golang. 
It provides rich capabilities for building AI applications including atomic components, 
integrated components, component orchestration, and aspect extensions. 
The framework helps developers create well-architected, maintainable, and highly available AI applications.`

	logs.Infof("Input document:\n%s", document)

	result, err := runner.Invoke(ctx, document)
	if err != nil {
		logs.Fatalf("Failed to invoke: %v", err)
	}

	logs.Infof("Final word count: %d", result)
}
