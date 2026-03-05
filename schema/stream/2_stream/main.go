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
	"io"
	"strings"
	"time"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/internal/logs"
)

func main() {
	ctx := context.Background()

	textGenerator := compose.StreamableLambda(func(ctx context.Context, prompt string) (*schema.StreamReader[string], error) {
		logs.Infof("Generating text for prompt: %q", prompt)

		response := "Eino is a powerful framework for building LLM applications in Go. " +
			"It provides streaming capabilities that enable real-time token generation."

		words := strings.Fields(response)

		sr, sw := schema.Pipe[string](1)

		go func() {
			defer sw.Close()
			for i, word := range words {
				select {
				case <-ctx.Done():
					return
				default:
					time.Sleep(100 * time.Millisecond)
					if i < len(words)-1 {
						sw.Send(word+" ", nil)
					} else {
						sw.Send(word, nil)
					}
				}
			}
		}()

		return sr, nil
	})

	graph := compose.NewGraph[string, string]()
	_ = graph.AddLambdaNode("generator", textGenerator)
	_ = graph.AddEdge(compose.START, "generator")
	_ = graph.AddEdge("generator", compose.END)

	runner, err := graph.Compile(ctx)
	if err != nil {
		logs.Fatalf("Failed to compile graph: %v", err)
	}

	logs.Infof("Starting stream generation...")

	stream, err := runner.Stream(ctx, "Tell me about Eino")
	if err != nil {
		logs.Fatalf("Failed to start stream: %v", err)
	}
	defer stream.Close()

	logs.Infof("Receiving tokens:")
	for {
		chunk, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logs.Errorf("Stream error: %v", err)
			break
		}
		logs.Tokenf("%s", chunk)
	}

	logs.Infof("\nStream completed")
}
