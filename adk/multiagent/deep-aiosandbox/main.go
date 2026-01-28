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
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino-ext/components/tool/commandline/aiosandbox"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/deep"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"

	"github.com/cloudwego/eino-examples/adk/common/prints"
	"github.com/cloudwego/eino-examples/adk/multiagent/deep/agents"
	"github.com/cloudwego/eino-examples/adk/multiagent/deep/params"
	"github.com/cloudwego/eino-examples/adk/multiagent/deep/tools"
	"github.com/cloudwego/eino-examples/adk/multiagent/deep/utils"
)

func main() {
	// Example queries - choose one:

	// 1. Simple test
	// query := schema.UserMessage("请帮我执行 echo 'Hello from AIO Sandbox' 并创建一个 test.txt 文件写入当前时间")

	// 2. Excel example - create and process Excel file
	query := schema.UserMessage(`请帮我完成以下任务：
1. 创建一个名为 sales.xlsx 的 Excel 文件，包含以下销售数据：
   - 列：产品名称、销售数量、单价、销售额
   - 数据：苹果/100/5/500, 香蕉/200/3/600, 橙子/150/4/600, 葡萄/80/8/640
2. 计算总销售额并添加到最后一行
3. 读取文件内容并展示`)

	ctx := context.Background()

	sandbox, agent, err := newExcelAgentWithAIOSandbox(ctx)
	if err != nil {
		log.Fatal(err)
	}
	defer sandbox.Close(ctx)

	// uuid as task id
	id := uuid.New().String()

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           agent,
		EnableStreaming: true,
	})

	// Use sandbox's work directory
	workdir := "/tmp/" + id

	// Create work directory in sandbox
	_, err = sandbox.RunCommand(ctx, []string{"mkdir", "-p", workdir})
	if err != nil {
		log.Printf("Failed to create work directory: %v", err)
		return
	}

	// Update sandbox work directory
	sandbox.SetWorkDir(workdir)

	ctx = params.InitContextParams(ctx)
	params.AppendContextParams(ctx, map[string]interface{}{
		params.FilePathSessionKey: workdir,
		params.WorkDirSessionKey:  workdir,
		params.TaskIDKey:          id,
	})

	fmt.Printf("Task ID: %s\n", id)
	fmt.Printf("Work Directory: %s\n", workdir)
	fmt.Printf("Query: %s\n\n", query.Content)

	iter := runner.Run(ctx, []*schema.Message{query})

	var (
		lastMessage       adk.Message
		lastMessageStream *schema.StreamReader[adk.Message]
	)

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Output != nil && event.Output.MessageOutput != nil {
			if lastMessageStream != nil {
				lastMessageStream.Close()
			}
			if event.Output.MessageOutput.IsStreaming {
				cpStream := event.Output.MessageOutput.MessageStream.Copy(2)
				event.Output.MessageOutput.MessageStream = cpStream[0]
				lastMessage = nil
				lastMessageStream = cpStream[1]
			} else {
				lastMessage = event.Output.MessageOutput.Message
				lastMessageStream = nil
			}
		}
		prints.Event(event)
	}

	// Print final result
	if lastMessage != nil {
		fmt.Printf("\n=== Final Message ===\n%v\n", lastMessage)
	} else if lastMessageStream != nil {
		msg, _ := schema.ConcatMessageStream(lastMessageStream)
		fmt.Printf("\n=== Final Message ===\n%v\n", msg)
	}

	// List files in work directory to verify
	fmt.Println("\n=== Files in work directory ===")
	output, err := sandbox.RunCommand(ctx, []string{"ls", "-la", workdir})
	if err != nil {
		log.Printf("Failed to list files: %v", err)
	} else {
		fmt.Println(output.Stdout)
	}

	time.Sleep(time.Second * 5)
}

func newExcelAgentWithAIOSandbox(ctx context.Context) (*aiosandbox.AIOSandbox, adk.Agent, error) {
	// Get AIO Sandbox configuration from environment
	baseURL := os.Getenv("AIO_SANDBOX_BASE_URL")
	if baseURL == "" {
		return nil, nil, fmt.Errorf("AIO_SANDBOX_BASE_URL environment variable is required")
	}

	// Token is optional
	token := os.Getenv("AIO_SANDBOX_TOKEN")

	// Create AIO Sandbox operator
	sandbox, err := aiosandbox.NewAIOSandbox(ctx, &aiosandbox.Config{
		BaseURL:     baseURL,
		Token:       token,
		WorkDir:     "/tmp",
		Timeout:     120,
		KeepSession: true, // Keep session for stateful operations
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create AIO sandbox: %w", err)
	}

	fmt.Printf("AIO Sandbox connected, Session ID: %s\n", sandbox.GetSessionID())

	// Create agent with AIO Sandbox as operator
	agent, err := newExcelAgent(ctx, sandbox)
	if err != nil {
		sandbox.Close(ctx)
		return nil, nil, err
	}

	return sandbox, agent, nil
}

func newExcelAgent(ctx context.Context, operator commandline.Operator) (adk.Agent, error) {
	cm, err := utils.NewChatModel(ctx,
		utils.WithMaxTokens(4096),
		utils.WithTemperature(float32(0)),
		utils.WithTopP(float32(0)),
	)
	if err != nil {
		return nil, err
	}

	ca, err := agents.NewCodeAgent(ctx, operator)
	if err != nil {
		return nil, err
	}
	wa, err := agents.NewWebSearchAgent(ctx)
	if err != nil {
		return nil, err
	}

	deepAgent, err := deep.New(ctx, &deep.Config{
		Name:        "ExcelAgent",
		Description: "an agent for excel task with AIO Sandbox",
		ChatModel:   cm,
		SubAgents:   []adk.Agent{ca, wa},
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					tools.NewWrapTool(tools.NewBashTool(operator), nil, nil),
					tools.NewWrapTool(tools.NewReadFileTool(operator), nil, nil),
					tools.NewWrapTool(tools.NewEditFileTool(operator), nil, nil),
					tools.NewWrapTool(tools.NewTreeTool(operator), nil, nil),
				},
			},
		},
		MaxIteration: 100,
	})
	if err != nil {
		return nil, err
	}

	return deepAgent, nil
}

// uploadFileToSandbox uploads a local file to the sandbox
func uploadFileToSandbox(ctx context.Context, sandbox *aiosandbox.AIOSandbox, localPath, remotePath string) error {
	content, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file: %w", err)
	}

	err = sandbox.WriteFile(ctx, remotePath, string(content))
	if err != nil {
		return fmt.Errorf("failed to write to sandbox: %w", err)
	}

	return nil
}
