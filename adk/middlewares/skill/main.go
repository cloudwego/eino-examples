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
	"path/filepath"

	"github.com/cloudwego/eino-ext/adk/backend/local"
	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/middlewares/filesystem"
	"github.com/cloudwego/eino/adk/middlewares/skill"

	"github.com/cloudwego/eino-examples/adk/common/prints"
)

func main() {
	ctx := context.Background()
	pwd, _ := os.Getwd()
	workDir := filepath.Join(pwd, "adk", "middlewares", "skill", "workdir")
	skillsDir := filepath.Join(workDir, "skills")

	cm, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		APIKey: os.Getenv("ARK_API_KEY"),
		Model:  os.Getenv("ARK_MODEL"),
	})
	if err != nil {
		log.Fatal(err)
	}

	be, err := local.NewBackend(ctx, &local.Config{})
	if err != nil {
		log.Fatal(err)
	}
	fsm, err := filesystem.NewMiddleware(ctx, &filesystem.Config{
		Backend:                          be,
		WithoutLargeToolResultOffloading: true,
	})
	if err != nil {
		log.Fatal(err)
	}

	skillBackend, err := skill.NewLocalBackend(&skill.LocalBackendConfig{
		BaseDir: skillsDir,
	})
	if err != nil {
		log.Fatalf("Failed to create skill backend: %v", err)
	}

	skillMiddleware, err := skill.New(ctx, &skill.Config{
		Backend: skillBackend,
	})
	if err != nil {
		log.Fatalf("Failed to create skill middleware: %v", err)
	}

	agent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "LogAnalysisAgent",
		Description: "An agent that can analyze logs",
		Instruction: "You are a helpful assistant. Use the 'skill' tool to find capabilities. Use 'exec_command' to run scripts.",
		Model:       cm,
		Middlewares: []adk.AgentMiddleware{skillMiddleware, fsm},
	})
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent: agent,
	})

	input := fmt.Sprintf("Analyze the %s file", filepath.Join(workDir, "test.log"))
	log.Println("User: ", input)

	iterator := runner.Query(ctx, input)
	for {
		event, ok := iterator.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			log.Printf("Error: %v\n", event.Err)
			break
		}

		prints.Event(event)
	}
}
