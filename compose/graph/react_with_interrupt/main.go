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
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	clc "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/coze-dev/cozeloop-go"
)

func main() {
	cozeloopApiToken := os.Getenv("COZELOOP_API_TOKEN")
	cozeloopWorkspaceID := os.Getenv("COZELOOP_WORKSPACE_ID") // use cozeloop trace, from https://loop.coze.cn/open/docs/cozeloop/go-sdk#4a8c980e

	ctx := context.Background()
	var handlers []callbacks.Handler
	if cozeloopApiToken != "" && cozeloopWorkspaceID != "" {
		client, err := cozeloop.NewClient(
			cozeloop.WithAPIToken(cozeloopApiToken),
			cozeloop.WithWorkspaceID(cozeloopWorkspaceID),
		)
		if err != nil {
			panic(err)
		}
		defer client.Close(ctx)
		handlers = append(handlers, clc.NewLoopHandler(client))
	}
	callbacks.AppendGlobalHandlers(handlers...)

	err := compose.RegisterSerializableType[myState]("state")
	if err != nil {
		log.Fatalf("RegisterSerializableType failed: %v", err)
	}
	runner, err := composeGraph[map[string]any, *schema.Message](
		ctx,
		newChatTemplate(ctx),
		newChatModel(ctx),
		newToolsNode(ctx),
		newCheckPointStore(ctx),
	)
	if err != nil {
		log.Fatal(err)
	}

	var history []*schema.Message

	for {
		result, err := runner.Invoke(ctx, map[string]any{"name": "Megumin", "location": "Beijing"}, compose.WithCheckPointID("1"), compose.WithStateModifier(func(ctx context.Context, path compose.NodePath, state any) error {
			state.(*myState).history = history
			return nil
		}))
		if err == nil {
			fmt.Printf("final result: %s", result.Content)
			break
		}

		info, ok := compose.ExtractInterruptInfo(err)
		if !ok {
			log.Fatal(err)
		}

		history = info.State.(*myState).history
		for i, tc := range history[len(history)-1].ToolCalls {
			fmt.Printf("will call tool: %s, arguments: %s\n", tc.Function.Name, tc.Function.Arguments)
			fmt.Print("Are the arguments as expected? (y/n): ")
			var response string
			_, _ = fmt.Scanln(&response)

			if strings.ToLower(response) == "n" {
				fmt.Print("Please enter the modified arguments: ")
				scanner := bufio.NewScanner(os.Stdin)
				var newArguments string
				if scanner.Scan() {
					newArguments = scanner.Text()
				}

				// Update the tool call arguments
				history[len(history)-1].ToolCalls[i].Function.Arguments = newArguments
				fmt.Printf("Updated arguments to: %s\n", newArguments)
			}
		}
	}
}

func newChatTemplate(_ context.Context) prompt.ChatTemplate {
	return prompt.FromMessages(schema.FString,
		schema.SystemMessage("You are a helpful assistant. If the user asks about the booking, call the \"BookTicket\" tool to book ticket."),
		schema.UserMessage("I'm {name}. Help me book a ticket to {location}"),
	)
}

func newChatModel(ctx context.Context) model.ChatModel {
	cm, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   os.Getenv("OPENAI_MODEL_NAME"),
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
	})
	if err != nil {
		log.Fatal(err)
	}

	tools := getTools()
	var toolsInfo []*schema.ToolInfo
	for _, t := range tools {
		info, err := t.Info(ctx)
		if err != nil {
			log.Fatal(err)
		}
		toolsInfo = append(toolsInfo, info)
	}

	err = cm.BindTools(toolsInfo)
	if err != nil {
		log.Fatal(err)
	}
	return cm
}

type bookInput struct {
	Location             string `json:"location"`
	PassengerName        string `json:"passenger_name"`
	PassengerPhoneNumber string `json:"passenger_phone_number"`
}

func newToolsNode(ctx context.Context) *compose.ToolsNode {
	tools := getTools()

	tn, err := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{Tools: tools})
	if err != nil {
		log.Fatal(err)
	}
	return tn
}

func newCheckPointStore(ctx context.Context) compose.CheckPointStore {
	return &myStore{buf: make(map[string][]byte)}
}

type myState struct {
	history []*schema.Message
}

func composeGraph[I, O any](ctx context.Context, tpl prompt.ChatTemplate, cm model.ChatModel, tn *compose.ToolsNode, store compose.CheckPointStore) (compose.Runnable[I, O], error) {
	g := compose.NewGraph[I, O](compose.WithGenLocalState(func(ctx context.Context) *myState {
		return &myState{}
	}))
	err := g.AddChatTemplateNode(
		"ChatTemplate",
		tpl,
	)
	if err != nil {
		return nil, err
	}
	err = g.AddChatModelNode(
		"ChatModel",
		cm,
		compose.WithStatePreHandler(func(ctx context.Context, in []*schema.Message, state *myState) ([]*schema.Message, error) {
			state.history = append(state.history, in...)
			return state.history, nil
		}),
		compose.WithStatePostHandler(func(ctx context.Context, out *schema.Message, state *myState) (*schema.Message, error) {
			state.history = append(state.history, out)
			return out, nil
		}),
	)
	if err != nil {
		return nil, err
	}
	err = g.AddToolsNode("ToolsNode", tn, compose.WithStatePreHandler(func(ctx context.Context, in *schema.Message, state *myState) (*schema.Message, error) {
		return state.history[len(state.history)-1], nil
	}))
	if err != nil {
		return nil, err
	}

	err = g.AddEdge(compose.START, "ChatTemplate")
	if err != nil {
		return nil, err
	}
	err = g.AddEdge("ChatTemplate", "ChatModel")
	if err != nil {
		return nil, err
	}
	err = g.AddEdge("ToolsNode", "ChatModel")
	if err != nil {
		return nil, err
	}
	err = g.AddBranch("ChatModel", compose.NewGraphBranch(func(ctx context.Context, in *schema.Message) (endNode string, err error) {
		if len(in.ToolCalls) > 0 {
			return "ToolsNode", nil
		}
		return compose.END, nil
	}, map[string]bool{"ToolsNode": true, compose.END: true}))
	if err != nil {
		return nil, err
	}
	return g.Compile(
		ctx,
		compose.WithCheckPointStore(store),
		compose.WithInterruptBeforeNodes([]string{"ToolsNode"}),
	)
}

func getTools() []tool.BaseTool {
	getWeather, err := utils.InferTool("BookTicket", "this tool can book ticket of the specific location", func(ctx context.Context, input bookInput) (output string, err error) {
		return "success", nil
	})
	if err != nil {
		log.Fatal(err)
	}

	return []tool.BaseTool{
		getWeather,
	}
}

type myStore struct {
	buf map[string][]byte
}

func (m *myStore) Get(ctx context.Context, checkPointID string) ([]byte, bool, error) {
	data, ok := m.buf[checkPointID]
	return data, ok, nil
}

func (m *myStore) Set(ctx context.Context, checkPointID string, checkPoint []byte) error {
	m.buf[checkPointID] = checkPoint
	return nil
}
