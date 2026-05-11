/*
 * Copyright 2026 CloudWeGo Authors
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
	"log"
	"os"
	"time"

	"github.com/bytedance/sonic"
	clc "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	"github.com/cloudwego/eino-ext/components/model/agenticark"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/eino/utils/callbacks"
	"github.com/coze-dev/cozeloop-go"
	"github.com/volcengine/volcengine-go-sdk/service/arkruntime/model/responses"
)

func main() {
	ctx := context.Background()

	cli, err := cozeloop.NewClient(
		cozeloop.WithAPIToken(os.Getenv("COZLOOP_API_TOKEN")),
		cozeloop.WithWorkspaceID(os.Getenv("COZLOOP_WORKSPACE_ID")),
	)
	if err != nil {
		panic(err)
	}
	defer cli.Close(ctx)

	am, err := agenticark.New(ctx, &agenticark.Config{
		Model:  os.Getenv("ARK_MODEL_ID"),
		APIKey: os.Getenv("ARK_API_KEY"),
		ServerTools: []*agenticark.ServerToolConfig{
			{
				WebSearch: &responses.ToolWebSearch{
					Type: responses.ToolType_web_search,
				},
			},
		},
	})
	if err != nil {
		log.Fatalf("failed to create agentic model, err=%v", err)
	}

	config := &AgentConfig{
		Model: am,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: []tool.BaseTool{
				&SummarizeNewsTool{},
				&GetUserLocationTool{},
			},
		},
		ToolReturnDirectly: map[string]struct{}{
			summarizeNewsToolName: {},
		},
	}

	r, err := NewAgent(ctx, config)
	if err != nil {
		log.Fatalf("failed to create agent, err=%v", err)
	}

	input := []*schema.AgenticMessage{
		schema.SystemAgenticMessage("You are a news assistant that helps users search for recent news. " +
			"Before using the `summarize_news` tool, " +
			"You MUST use the `get_user_location` tool to get the user's country. " +
			"You MUST use the `summarize_news` tool to summarize the news and return the result."),
		schema.UserAgenticMessage("What news has been happening in the last three days?"),
	}

	cb := callbacks.NewHandlerHelper().AgenticModel(&callbacks.AgenticModelCallbackHandler{
		OnEndWithStreamOutput: newAgenticModelCallback().OnEndWithStreamOutput,
	}).Handler()

	toolInfos, err := genToolInfos(ctx, config.ToolsConfig)
	if err != nil {
		log.Fatalf("failed to generate tool infos, err=%v", err)
	}

	sr, err := r.Stream(ctx, input,
		WithComposeOptions(
			compose.WithChatModelOption(model.WithTools(toolInfos)),
			compose.WithCallbacks(cb, clc.NewLoopHandler(cli)),
			compose.WithCallbacks(cb),
		))
	if err != nil {
		log.Fatalf("failed to stream, err=%v", err)
	}

	var msgs []*schema.AgenticMessage
	for {
		msg, recvErr := sr.Recv()
		if recvErr != nil {
			if recvErr == io.EOF {
				break
			}
			log.Fatalf("failed to recv, err=%v", recvErr)
		}
		msgs = append(msgs, msg)
	}

	msg, err := schema.ConcatAgenticMessages(msgs)
	if err != nil {
		log.Fatalf("failed to concat messages, err=%v", err)
	}

	printHeader("final output", "\033[36m")

	for _, block := range msg.ContentBlocks {
		switch block.Type {
		case schema.ContentBlockTypeFunctionToolResult:
			switch block.FunctionToolResult.Name {
			case summarizeNewsToolName:
				if len(block.FunctionToolResult.Content) == 0 || block.FunctionToolResult.Content[0].Text == nil {
					log.Fatalf("summarize_news returned empty content")
				}

				var res *SummarizeNewsToolInput
				if err = sonic.UnmarshalString(block.FunctionToolResult.Content[0].Text.Text, &res); err != nil {
					log.Fatalf("failed to unmarshal function tool result, err=%v", err)
				}

				b, err := sonic.MarshalIndent(res.News, "", "  ")
				if err != nil {
					log.Fatalf("failed to marshal function tool result, err=%v", err)
				}

				fmt.Println(string(b))
			default:
				fmt.Printf("%s\n", block)
			}

		default:
			fmt.Printf("%s\n", block)
		}
	}

	time.Sleep(5 * time.Second)
}
