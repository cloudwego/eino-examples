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
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

const (
	demoBudgetCNY           = 1200.0
	maxAutoBookingOptionCNY = 300.0
)

func main() {
	ctx := context.Background()

	workspaceDir, err := prepareWorkspace()
	if err != nil {
		log.Fatal(err)
	}

	planPath := filepath.Join(workspaceDir, "travel_plan.md")
	_ = os.Remove(planPath)

	agent, err := newTravelPlannerAgent(ctx, planPath)
	if err != nil {
		log.Fatal(err)
	}

	runner := adk.NewTypedRunner[*schema.AgenticMessage](adk.TypedRunnerConfig[*schema.AgenticMessage]{
		Agent:           agent,
		EnableStreaming: true,
	})

	fmt.Println("正在运行 Agentic 旅行规划示例...")
	fmt.Printf("workspace: %s\n", workspaceDir)

	iter := runner.Run(ctx, []*schema.AgenticMessage{userRequest(planPath)},
		adk.WithChatModelOptions(agenticRunOptions()),
	)
	if err := printEvents(iter); err != nil {
		log.Fatal(err)
	}
	if err := validateTravelPlan(planPath); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\n完成。旅行计划路径: %s\n", planPath)
}

func prepareWorkspace() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("resolve current source file")
	}

	workspaceDir := filepath.Join(filepath.Dir(file), "workspace")
	if err := os.MkdirAll(workspaceDir, 0o755); err != nil {
		return "", fmt.Errorf("create workspace: %w", err)
	}
	return workspaceDir, nil
}

func validateTravelPlan(planPath string) error {
	info, err := os.Stat(planPath)
	if err != nil {
		return fmt.Errorf("travel plan was not generated: %w", err)
	}
	if info.IsDir() {
		return fmt.Errorf("travel plan path is a directory: %s", planPath)
	}

	content, err := os.ReadFile(planPath)
	if err != nil {
		return fmt.Errorf("read generated travel plan: %w", err)
	}
	if strings.TrimSpace(string(content)) == "" {
		return fmt.Errorf("generated travel plan is empty: %s", planPath)
	}

	return nil
}
