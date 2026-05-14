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

func main() {
	ctx := context.Background()

	workspaceDir, err := prepareWorkspace()
	if err != nil {
		log.Fatal(err)
	}

	reportPath := filepath.Join(workspaceDir, "research_report.md")
	_ = os.Remove(reportPath)

	agent, err := newResearchAssistant(ctx, reportPath)
	if err != nil {
		log.Fatal(err)
	}

	runner := adk.NewTypedRunner[*schema.AgenticMessage](adk.TypedRunnerConfig[*schema.AgenticMessage]{
		Agent:           agent,
		EnableStreaming: true,
	})

	input := schema.UserAgenticMessage(userRequest(reportPath))

	fmt.Println("Running Agentic Research Assistant...")
	fmt.Printf("workspace: %s\n", workspaceDir)
	printAgenticMessage(1, input)

	iter := runner.Run(ctx, []*schema.AgenticMessage{input},
		adk.WithChatModelOptions(agenticRunOptions()),
	)
	messageIndex := 1
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			log.Fatal(event.Err)
		}

		msg, _, err := adk.TypedGetMessage(event)
		if err != nil {
			log.Fatal(err)
		}
		if msg != nil {
			messageIndex++
			printAgenticMessage(messageIndex, msg)
		}
	}

	if err := validateResearchReport(reportPath); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("\nDone. Research report: %s\n", reportPath)
}

func printAgenticMessage(index int, msg *schema.AgenticMessage) {
	if msg == nil {
		return
	}
	fmt.Printf("\n--- AgenticMessage #%d ---\n", index)
	fmt.Print(msg.String())
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

func validateResearchReport(reportPath string) error {
	content, err := os.ReadFile(reportPath)
	if err != nil {
		return fmt.Errorf("research report was not generated: %w", err)
	}
	if strings.TrimSpace(string(content)) == "" {
		return fmt.Errorf("research report is empty: %s", reportPath)
	}
	return nil
}
