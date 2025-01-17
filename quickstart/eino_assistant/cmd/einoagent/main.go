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
	"log"
	"os"
	"time"

	"github.com/cloudwego/eino-ext/devops"

	"github.com/cloudwego/eino-examples/quickstart/eino_assistant/cmd/einoagent/agent"
	"github.com/cloudwego/eino-examples/quickstart/eino_assistant/cmd/einoagent/task"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/app/server"
)

func init() {
	if os.Getenv("EINO_DEBUG") != "false" {
		err := devops.Init(context.Background())
		if err != nil {
			log.Printf("[eino dev] init failed, err=%v", err)
		}
	}
}

func main() {
	// 获取端口
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// 创建 Hertz 服务器
	h := server.Default(server.WithHostPorts(":" + port))

	h.Use(LogMiddleware())

	// 注册 task 路由组
	taskGroup := h.Group("/task")
	if err := task.BindRoutes(taskGroup); err != nil {
		log.Fatal("failed to bind task routes:", err)
	}

	// 注册 agent 路由组
	agentGroup := h.Group("/agent")
	if err := agent.BindRoutes(agentGroup); err != nil {
		log.Fatal("failed to bind agent routes:", err)
	}

	// Redirect root path to /agent
	h.GET("/", func(ctx context.Context, c *app.RequestContext) {
		c.Redirect(302, []byte("/agent"))
	})

	// 启动服务器
	h.Spin()
}

// LogMiddleware 记录 HTTP 请求日志
func LogMiddleware() app.HandlerFunc {
	return func(ctx context.Context, c *app.RequestContext) {
		start := time.Now()
		path := string(c.Request.URI().Path())
		method := string(c.Request.Method())

		// 处理请求
		c.Next(ctx)

		// 记录请求信息
		latency := time.Since(start)
		statusCode := c.Response.StatusCode()
		log.Printf("[HTTP] %s %s %d %v\n", method, path, statusCode, latency)
	}
}
