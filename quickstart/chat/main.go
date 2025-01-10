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
	"log"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/internal/logs"
)

func main() {
	openAIAPIKey := os.Getenv("OPENAI_API_KEY")
	openAIBaseURL := os.Getenv("OPENAI_BASE_URL")
	openAIModelName := os.Getenv("OPENAI_MODEL_NAME")

	ctx := context.Background()

	// 创建模板，使用 FString 格式
	template := prompt.FromMessages(schema.FString,
		// 系统消息模板
		schema.SystemMessage("你是一个{role}。你需要用{style}的语气回答问题。你的目标是帮助程序员保持积极乐观的心态，提供技术建议的同时也要关注他们的心理健康，给他们提供足够的情绪价值。"),

		// 插入可选的示例对话
		schema.MessagesPlaceholder("examples", true),

		// 插入必需的对话历史
		schema.MessagesPlaceholder("chat_history", false),

		// 用户消息模板
		schema.UserMessage("问题: {question}"),
	)

	// 使用模板生成消息
	messages, err := template.Format(ctx, map[string]any{
		"role":     "程序员鼓励师",
		"style":    "积极、温暖且专业",
		"question": "我的代码一直报错，感觉好沮丧，该怎么办？",
		// 对话历史（必需的）
		"chat_history": []*schema.Message{
			schema.UserMessage("你好"),
			schema.AssistantMessage("嘿！我是你的程序员鼓励师！记住，每个优秀的程序员都是从 Debug 中成长起来的。有什么我可以帮你的吗？", nil),
		},
		// 示例对话（可选的）
		"examples": []*schema.Message{
			schema.UserMessage("我觉得自己写的代码太烂了"),
			schema.AssistantMessage("每个程序员都经历过这个阶段！重要的是你在不断学习和进步。让我们一起看看代码，我相信通过重构和优化，它会变得更好。记住，Rome wasn't built in a day，代码质量是通过持续改进来提升的。", nil),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	// 创建 OpenAI ChatModel, 假设使用 openai 官方服务。
	chatModel, err := openai.NewChatModel(ctx, &openai.ChatModelConfig{
		BaseURL: openAIBaseURL,
		Model:   openAIModelName, // 使用的模型版本
		APIKey:  openAIAPIKey,    // OpenAI API 密钥

		// 可选的 Azure OpenAI 配置
		ByAzure:    true, // 是否使用 Azure OpenAI
		APIVersion: "2024-06-01",
	})
	if err != nil {
		logs.Errorf("NewChatModel failed, err=%v", err)
		return
	}

	// 使用 Generate 获取完整回复
	response, err := chatModel.Generate(ctx, messages)
	if err != nil {
		logs.Errorf("chatModel.Generate failed, err=%v", err)
		return
	}

	logs.Infof("below is chat model's output:")
	logs.Tokenf("%v", response.Content)
}
