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

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/internal/logs"
)

func main() {
	ctx := context.Background()

	sensitiveWords := map[string]string{
		"password": "***",
		"secret":   "***",
		"token":    "***",
		"key":      "***",
	}

	textFilter := compose.TransformableLambda(func(ctx context.Context, input *schema.StreamReader[string]) (*schema.StreamReader[string], error) {
		logs.Infof("Starting text filter transformation...")

		return schema.StreamReaderWithConvert(input, func(chunk string) (string, error) {
			result := chunk
			for word, replacement := range sensitiveWords {
				result = strings.ReplaceAll(
					strings.ToLower(result),
					word,
					replacement,
				)
			}
			return result, nil
		}), nil
	})

	textUppercase := compose.TransformableLambda(func(ctx context.Context, input *schema.StreamReader[string]) (*schema.StreamReader[string], error) {
		logs.Infof("Starting uppercase transformation...")

		return schema.StreamReaderWithConvert(input, func(chunk string) (string, error) {
			return strings.ToUpper(chunk), nil
		}), nil
	})

	graph := compose.NewGraph[string, string]()
	_ = graph.AddLambdaNode("filter", textFilter)
	_ = graph.AddLambdaNode("uppercase", textUppercase)
	_ = graph.AddEdge(compose.START, "filter")
	_ = graph.AddEdge("filter", "uppercase")
	_ = graph.AddEdge("uppercase", compose.END)

	runner, err := graph.Compile(ctx)
	if err != nil {
		logs.Fatalf("Failed to compile graph: %v", err)
	}

	textChunks := []string{
		"User login with password: ",
		"abc123 and token: ",
		"xyz789. The secret ",
		"key is hidden.",
	}

	inputStream := schema.StreamReaderFromArray(textChunks)

	logs.Infof("Input chunks: %v", textChunks)
	logs.Infof("Processing stream with Transform...")

	outputStream, err := runner.Transform(ctx, inputStream)
	if err != nil {
		logs.Fatalf("Failed to transform: %v", err)
	}
	defer outputStream.Close()

	logs.Infof("Output chunks:")
	var allOutput strings.Builder
	for {
		chunk, err := outputStream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			logs.Errorf("Transform error: %v", err)
			break
		}
		logs.Tokenf("[%s]", chunk)
		allOutput.WriteString(chunk)
	}

	logs.Infof("\nFinal output: %s", allOutput.String())
}
