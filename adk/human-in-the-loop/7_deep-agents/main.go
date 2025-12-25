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

	"github.com/cloudwego/eino/adk"

	"github.com/cloudwego/eino-examples/adk/common/prints"
	"github.com/cloudwego/eino-examples/adk/common/store"
	"github.com/cloudwego/eino-examples/adk/common/tool"
)

func main() {
	ctx := context.Background()

	agent, err := NewDataAnalysisDeepAgent(ctx, newRateLimitedModel())
	if err != nil {
		log.Fatalf("failed to create deep agent: %v", err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true,
		Agent:           agent,
		CheckPointStore: store.NewInMemoryStore(),
	})

	query := "Analyze the market trends and provide investment recommendations."

	fmt.Println("\n========================================")
	fmt.Println("User Query:", query)
	fmt.Println("========================================")
	fmt.Println()

	iter := runner.Query(ctx, query, adk.WithCheckPointID("deep-analysis-1"))

	for {
		lastEvent, interrupted := processEvents(iter)
		if !interrupted {
			break
		}

		interruptCtx := lastEvent.Action.Interrupted.InterruptContexts[0]
		interruptID := interruptCtx.ID
		followUpInfo := interruptCtx.Info.(*tool.FollowUpInfo)

		fmt.Println("\n========================================")
		fmt.Println("CLARIFICATION NEEDED")
		fmt.Println("========================================")
		fmt.Println("The agent needs more information to proceed:")
		fmt.Println()
		for i, q := range followUpInfo.Questions {
			fmt.Printf("  %d. %s\n", i+1, q)
		}
		fmt.Println()
		fmt.Println("----------------------------------------")

		scanner := bufio.NewScanner(os.Stdin)
		var answers []string
		for i, q := range followUpInfo.Questions {
			fmt.Printf("Answer for Q%d (%s): ", i+1, truncate(q, 50))
			scanner.Scan()
			answers = append(answers, scanner.Text())
		}

		followUpInfo.UserAnswer = strings.Join(answers, "\n")

		fmt.Println("\n========================================")
		fmt.Println("Resuming with your answers...")
		fmt.Println("========================================")
		fmt.Println()

		iter, err = runner.ResumeWithParams(ctx, "deep-analysis-1", &adk.ResumeParams{
			Targets: map[string]any{
				interruptID: followUpInfo,
			},
		})
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("\n========================================")
	fmt.Println("Analysis completed!")
	fmt.Println("========================================")
}

func processEvents(iter *adk.AsyncIterator[*adk.AgentEvent]) (*adk.AgentEvent, bool) {
	var lastEvent *adk.AgentEvent
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			log.Fatal(event.Err)
		}

		prints.Event(event)
		lastEvent = event
	}

	if lastEvent == nil {
		return nil, false
	}
	if lastEvent.Action != nil && lastEvent.Action.Interrupted != nil {
		return lastEvent, true
	}
	return lastEvent, false
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
