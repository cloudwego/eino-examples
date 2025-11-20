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

	"github.com/cloudwego/eino/adk"

	"github.com/cloudwego/eino-examples/adk/common/prints"
	"github.com/cloudwego/eino-examples/adk/common/store"
	"github.com/cloudwego/eino-examples/adk/common/tool"
)

func main() {
	ctx := context.Background()
	a := NewItineraryAgent()
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           a,
		CheckPointStore: store.NewInMemoryStore(),
	})

	// Start with a vague request that will force the agent to ask for more information.
	iter := runner.Query(ctx, "Plan a 3-day trip to New York City.", adk.WithCheckPointID("1"))

	for {
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
			log.Fatal("last event is nil")
		}

		// If the agent exits, the conversation is over.
		if lastEvent.Action != nil && lastEvent.Action.Exit {
			fmt.Println("\n--- Conversation Finished ---")
			return
		}

		if lastEvent.Action == nil || lastEvent.Action.Interrupted == nil {
			fmt.Println("\n--- Conversation Finished ---")
			return
		}

		// Handle the follow-up interrupt
		interruptCtx := lastEvent.Action.Interrupted.InterruptContexts[0]
		fuInfo := interruptCtx.Info.(*tool.FollowUpInfo)

		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print("\nYour answer: ")
		scanner.Scan()
		fmt.Println()
		nInput := scanner.Text()

		fuInfo.UserAnswer = nInput

		var err error
		iter, err = runner.ResumeWithParams(ctx, "1", &adk.ResumeParams{
			Targets: map[string]any{
				interruptCtx.ID: fuInfo,
			},
		})
		if err != nil {
			log.Fatal(err)
		}
	}
}
