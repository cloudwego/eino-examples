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
	"errors"
	"flag"
	"fmt"
	"io"
	"sync"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/tool/duckduckgo/v2"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/hertz-contrib/sse"
)

var (
	modelName = flag.String("model", "", "The model to use, eg. ep-xxxx")
	apiKey    = flag.String("apikey", "", "The apikey of the model, eg. xxx")
	prompt    = flag.String("prompt", "you are a helpful assistant", "The system prompt to use")
)

func main() {
	flag.Parse()
	if *modelName == "" || *apiKey == "" {
		panic("model and apikey are required, you may get doubao model from: https://console.volcengine.com/ark/region:ark+cn-beijing/model/detail?Id=doubao-pro-32k")
	}

	h := server.Default()
	memory := &SimpleMemory{conversations: make(map[string]*Conversation)}

	h.GET("/chat", func(ctx context.Context, c *app.RequestContext) {
		id := c.Query("id")
		if id == "" {
			c.JSON(consts.StatusBadRequest, map[string]string{"error": "missing id, it's required for saving conversation, example: /chat?id=123"})
			return
		}

		msgString := c.Query("msg")
		if msgString == "" {
			c.JSON(consts.StatusBadRequest, map[string]string{"error": "missing msg, it's required for saving conversation, example: /chat?id=123&msg=hello"})
			return
		}

		conv := memory.GetOrCreateConversation(id)
		msg := schema.UserMessage(msgString)
		conv.Append(msg)

		msgs := conv.GetMessages()

		agent, err := NewAgent(ctx)
		if err != nil {
			c.JSON(consts.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		sr, err := agent.Stream(ctx, append([]*schema.Message{schema.SystemMessage(*prompt)}, msgs...))
		if err != nil {
			c.JSON(consts.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}

		c.SetStatusCode(consts.StatusOK)
		c.Response.Header.Set("Content-Type", "text/event-stream")
		c.Response.Header.Set("Cache-Control", "no-cache")
		c.Response.Header.Set("Connection", "keep-alive")

		s := sse.NewStream(c)
		fullMsgs := make([]*schema.Message, 0)

		defer func() {
			sr.Close()
			c.Flush()

			if err != nil && !errors.Is(err, io.EOF) {
				c.AbortWithStatusJSON(consts.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}

			fullMsg, err := schema.ConcatMessages(fullMsgs)
			if err != nil {
				fmt.Println("error concatenating messages: ", err.Error())
				return
			}
			conv.Append(fullMsg)
		}()

		for {
			var chunk *schema.Message
			chunk, err = sr.Recv()
			if errors.Is(err, io.EOF) {
				break
			}
			if err != nil {
				fmt.Println("error receiving chunk: ", err.Error())
				return
			}
			fullMsgs = append(fullMsgs, chunk)
			err = s.Publish(&sse.Event{
				Data: []byte(chunk.Content),
			})
			if err != nil {
				fmt.Println("error publishing event: ", err.Error())
				return
			}
		}
	})

	h.Spin()
}

func NewAgent(ctx context.Context) (*react.Agent, error) {

	// 初始化模型
	m, err := PrepareModel(ctx)
	if err != nil {
		return nil, err
	}

	// 初始化各种 tool
	tools, err := PrepareTools(ctx)
	if err != nil {
		return nil, err
	}

	// 初始化 agent
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		Model: m,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: tools,
		},
	})
	if err != nil {
		return nil, err
	}
	return agent, nil
}

func PrepareModel(ctx context.Context) (model.ChatModel, error) {

	// eg. 使用 ark 豆包大模型, or openai: openai.NewChatModel at github.com/cloudwego/eino-ext/components/model/openai
	arkModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		Model:  *modelName, // replace with your model
		APIKey: *apiKey,    // replace with your api key
	})
	if err != nil {
		return nil, err
	}
	return arkModel, nil
}

func PrepareTools(ctx context.Context) ([]tool.BaseTool, error) {
	ddg, err := duckduckgo.NewTextSearchTool(ctx, &duckduckgo.Config{})
	if err != nil {
		return nil, err
	}
	return []tool.BaseTool{ddg}, nil
}

// simple memory can store messages of each conversation
type SimpleMemory struct {
	mu            sync.Mutex
	conversations map[string]*Conversation
}

func (m *SimpleMemory) GetOrCreateConversation(id string) *Conversation {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.conversations[id]; !ok {
		m.conversations[id] = &Conversation{
			ID:       id,
			Messages: make([]*schema.Message, 0),
		}
	}

	return m.conversations[id]
}

type Conversation struct {
	mu sync.Mutex

	ID       string
	Messages []*schema.Message
}

func (c *Conversation) Append(msg *schema.Message) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.Messages = append(c.Messages, msg)
}

func (c *Conversation) GetMessages() []*schema.Message {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.Messages
}
