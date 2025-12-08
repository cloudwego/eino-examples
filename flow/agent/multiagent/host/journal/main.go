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
	"bufio"
	"context"
	"io"
	"os"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/multiagent/host"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/devops/visualize"
)

func main() {

	openAIAPIKey := os.Getenv("OPENAI_API_KEY")
	openAIBaseURL := os.Getenv("OPENAI_BASE_URL")
	openAIModelName := os.Getenv("OPENAI_MODEL_NAME")

	ollamaBaseURL := os.Getenv("OLLAMA_BASE_URL")
	ollamaModel := os.Getenv("OLLAMA_MODEL")

	ctx := context.Background()
	h, err := newHost(ctx, openAIBaseURL, openAIAPIKey, openAIModelName)
	if err != nil {
		panic(err)
	}

	writer, err := newWriteJournalSpecialist(ctx, ollamaBaseURL, ollamaModel)
	if err != nil {
		panic(err)
	}

	reader, err := newReadJournalSpecialist(ctx)
	if err != nil {
		panic(err)
	}

	answerer, err := newAnswerWithJournalSpecialist(ctx, ollamaBaseURL, ollamaModel)
	if err != nil {
		panic(err)
	}

	hostMA, err := host.NewMultiAgent(ctx, &host.MultiAgentConfig{
		Host: *h,
		Specialists: []*host.Specialist{
			writer,
			reader,
			answerer,
		},
	})
	if err != nil {
		panic(err)
	}

	// export the graph via API and render mermaid (non-critical path)
	{
		anyG, opts := hostMA.ExportGraph()
		gen := visualize.NewMermaidGenerator("flow/agent/multiagent/host/journal")
		g := compose.NewGraph[[]*schema.Message, *schema.Message]()
		_ = g.AddGraphNode("host_multiagent", anyG, opts...)
		_ = g.AddEdge(compose.START, "host_multiagent")
		_ = g.AddEdge("host_multiagent", compose.END)
		_, _ = g.Compile(context.Background(), compose.WithGraphCompileCallbacks(gen))
	}

	cb := &logCallback{}

	for { // 多轮对话，除非用户输入了 "exit"，否则一直循环
		println("\n\nYou: ") // 提示轮到用户输入了

		var message string
		scanner := bufio.NewScanner(os.Stdin) // 获取用户在命令行的输入
		for scanner.Scan() {
			message += scanner.Text()
			break
		}

		if err := scanner.Err(); err != nil {
			panic(err)
		}

		if message == "exit" {
			return
		}

		msg := &schema.Message{
			Role:    schema.User,
			Content: message,
		}

		out, err := hostMA.Stream(ctx, []*schema.Message{msg}, host.WithAgentCallbacks(cb))
		if err != nil {
			panic(err)
		}

		println("\nAnswer:")

		for {
			msg, err := out.Recv()
			if err != nil {
				if err == io.EOF {
					break
				}
			}

			print(msg.Content)
		}

		out.Close()
	}
}

type logCallback struct{}

func (l *logCallback) OnHandOff(ctx context.Context, info *host.HandOffInfo) context.Context {
	println("\nHandOff to", info.ToAgentName, "with argument", info.Argument)
	return ctx
}
