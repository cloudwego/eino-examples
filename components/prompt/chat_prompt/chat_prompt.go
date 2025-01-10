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

	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/internal/logs"
)

func main() {

	systemTpl := `你是情绪助手，你的任务是根据用户的输入，生成一段赞美的话，语句优美，韵律强。
用户姓名：{user_name}
用户年龄：{user_age}
用户性别：{user_gender}
用户喜好：{user_hobby}`

	chatTpl := prompt.FromMessages(schema.FString,
		schema.SystemMessage(systemTpl),
		schema.MessagesPlaceholder("message_histories", true),
		schema.UserMessage("{user_query}"),
	)

	msgList, err := chatTpl.Format(context.Background(), map[string]any{
		"user_name":   "张三",
		"user_age":    "18",
		"user_gender": "男",
		"user_hobby":  "打篮球、打游戏",
		"message_histories": []*schema.Message{ // => value of "messages_histories" will be rendered into chatTpl slot.
			schema.UserMessage("我喜欢打羽毛球"),
			schema.AssistantMessage("xxxxxxxx", nil),
		},
		"user_query": "请为我赋诗一首",
	})
	if err != nil {
		logs.Errorf("Format failed, err=%v", err)
		return
	}

	logs.Infof("Rendered Messages:")
	for _, msg := range msgList {
		logs.Infof("- %v", msg)
	}
}
