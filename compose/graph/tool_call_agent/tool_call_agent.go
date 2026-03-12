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
	"os"

	clc "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/coze-dev/cozeloop-go"

	"github.com/cloudwego/eino-examples/internal/gptr"
	"github.com/cloudwego/eino-examples/internal/logs"
)

func main() {

	openAIBaseURL := os.Getenv("OPENAI_BASE_URL")
	openAIAPIKey := os.Getenv("OPENAI_API_KEY")
	modelName := os.Getenv("OPENAI_MODEL_NAME")
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

	handlers = append(handlers, &loggerCallbacks{})
	callbacks.AppendGlobalHandlers(handlers...)

	// 1. create an instance of ChatTemplate as 1st Graph Node
	systemTpl := `你是一名房产经纪人，结合用户的薪酬和工作，使用 user_info API，为其提供相关的房产信息。邮箱是必须的`
	chatTpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage(systemTpl),
		schema.MessagesPlaceholder("message_histories", true),
		schema.UserMessage("{user_query}"),
	)

	// 2. create an instance of ChatModel as 2nd Graph Node
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL:     openAIBaseURL,
		APIKey:      openAIAPIKey,
		Model:       modelName,
		Temperature: gptr.Of(float32(0.7)),
	})
	if err != nil {
		logs.Errorf("NewChatModel failed, err=%v", err)
		return
	}

	// 3. create an instance of tool.InvokableTool for Intent recognition and execution
	userInfoTool := utils.NewTool(
		&schema.ToolInfo{
			Name: "user_info",
			Desc: "根据用户的姓名和邮箱，查询用户的公司、职位、薪酬信息",
			ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
				"name": {
					Type: "string",
					Desc: "用户的姓名",
				},
				"email": {
					Type: "string",
					Desc: "用户的邮箱",
				},
			}),
		},
		func(ctx context.Context, input *userInfoRequest) (output *userInfoResponse, err error) {
			return &userInfoResponse{
				Name:     input.Name,
				Email:    input.Email,
				Company:  "Awesome company",
				Position: "CEO",
				Salary:   "9999",
			}, nil
		})

	info, err := userInfoTool.Info(ctx)
	if err != nil {
		logs.Errorf("Get ToolInfo failed, err=%v", err)
		return
	}

	// 4. bind ToolInfo to ChatModel. ToolInfo will remain in effect until the next BindTools.
	err = chatModel.BindForcedTools([]*schema.ToolInfo{info})
	if err != nil {
		logs.Errorf("BindForcedTools failed, err=%v", err)
		return
	}

	// 5. create an instance of ToolsNode as 3rd Graph Node
	toolsNode, err := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{
		Tools: []tool.BaseTool{userInfoTool},
	})
	if err != nil {
		logs.Errorf("NewToolNode failed, err=%v", err)
		return
	}

	const (
		nodeKeyOfTemplate  = "template"
		nodeKeyOfChatModel = "chat_model"
		nodeKeyOfTools     = "tools"
	)

	// 6. create an instance of Graph
	// input type is 1st Graph Node's input type, that is ChatTemplate's input type: map[string]any
	// output type is last Graph Node's output type, that is ToolsNode's output type: []*schema.Message
	g := compose.NewGraph[map[string]any, []*schema.Message]()

	// 7. add ChatTemplate into graph
	_ = g.AddChatTemplateNode(nodeKeyOfTemplate, chatTpl)

	// 8. add ChatModel into graph
	_ = g.AddChatModelNode(nodeKeyOfChatModel, chatModel)

	// 9. add ToolsNode into graph
	_ = g.AddToolsNode(nodeKeyOfTools, toolsNode)

	// 10. add connection between nodes
	_ = g.AddEdge(compose.START, nodeKeyOfTemplate)

	_ = g.AddEdge(nodeKeyOfTemplate, nodeKeyOfChatModel)

	_ = g.AddEdge(nodeKeyOfChatModel, nodeKeyOfTools)

	_ = g.AddEdge(nodeKeyOfTools, compose.END)

	// 9. compile Graph[I, O] to Runnable[I, O]
	r, err := g.Compile(ctx)
	if err != nil {
		logs.Errorf("Compile failed, err=%v", err)
		return
	}

	out, err := r.Invoke(ctx, map[string]any{
		"message_histories": []*schema.Message{},
		"user_query":        "我叫 zhangsan, 邮箱是 zhangsan@bytedance.com, 帮我推荐一处房产",
	})
	if err != nil {
		logs.Errorf("Invoke failed, err=%v", err)
		return
	}
	logs.Infof("Generation: %v Messages", len(out))
	for _, msg := range out {
		logs.Infof("    %v", msg)
	}
}

type userInfoRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

type userInfoResponse struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Company  string `json:"company"`
	Position string `json:"position"`
	Salary   string `json:"salary"`
}

type loggerCallbacks struct{}

func (l *loggerCallbacks) OnStart(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
	logs.Infof("name: %v, type: %v, component: %v, input: %v", info.Name, info.Type, info.Component, input)
	return ctx
}

func (l *loggerCallbacks) OnEnd(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
	logs.Infof("name: %v, type: %v, component: %v, output: %v", info.Name, info.Type, info.Component, output)
	return ctx
}

func (l *loggerCallbacks) OnError(ctx context.Context, info *callbacks.RunInfo, err error) context.Context {
	logs.Infof("name: %v, type: %v, component: %v, error: %v", info.Name, info.Type, info.Component, err)
	return ctx
}

func (l *loggerCallbacks) OnStartWithStreamInput(ctx context.Context, info *callbacks.RunInfo, input *schema.StreamReader[callbacks.CallbackInput]) context.Context {
	return ctx
}

func (l *loggerCallbacks) OnEndWithStreamOutput(ctx context.Context, info *callbacks.RunInfo, output *schema.StreamReader[callbacks.CallbackOutput]) context.Context {
	return ctx
}
