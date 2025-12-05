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
	"time"

	clc "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/callbacks"
	"github.com/coze-dev/cozeloop-go"

	"github.com/cloudwego/eino-examples/adk/common/prints"
	"github.com/cloudwego/eino-examples/adk/common/store"
	"github.com/cloudwego/eino-examples/adk/common/tool"
)

func main() {
	ctx := context.Background()

	cozeloopApiToken := os.Getenv("COZELOOP_API_TOKEN")
	cozeloopWorkspaceID := os.Getenv("COZELOOP_WORKSPACE_ID") // use cozeloop trace, from https://loop.coze.cn/open/docs/cozeloop/go-sdk#4a8c980e

	var handlers []callbacks.Handler
	if cozeloopApiToken != "" && cozeloopWorkspaceID != "" {
		client, err := cozeloop.NewClient(
			cozeloop.WithAPIToken(cozeloopApiToken),
			cozeloop.WithWorkspaceID(cozeloopWorkspaceID),
		)
		if err != nil {
			panic(err)
		}
		defer func() {
			time.Sleep(5 * time.Second)
			client.Close(ctx)
		}()
		handlers = append(handlers, clc.NewLoopHandler(client))
	}
	callbacks.AppendGlobalHandlers(handlers...)

	a := NewTicketBookingAgent()
	runner := adk.NewRunner(context.Background(), adk.RunnerConfig{
		EnableStreaming: true, // you can disable streaming here
		Agent:           a,

		// provide a CheckPointStore for eino to persist the execution state of the agent for later resumption.
		// Here we use an in-memory store for simplicity.
		// In the real world, you can use a distributed store like Redis to persist the checkpoints.
		CheckPointStore: store.NewInMemoryStore(),
	})
	iter := runner.Query(context.Background(), "book a ticket for Martin, to Beijing, on 2025-12-01, the phone number is 1234567. directly call tool.", adk.WithCheckPointID("1"))

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
	if lastEvent.Action == nil || lastEvent.Action.Interrupted == nil {
		log.Fatal("last event is not an interrupt event")
	}

	// this interruptID is crucial 'locator' for Eino to know where the interrupt happens,
	// so when resuming later, you have to provide this same `interruptID` along with the approval result back to Eino
	interruptID := lastEvent.Action.Interrupted.InterruptContexts[0].ID

	var apResult *tool.ApprovalResult
	for {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print("your input here: ")
		scanner.Scan()
		fmt.Println()
		nInput := scanner.Text()
		if strings.ToUpper(nInput) == "Y" {
			apResult = &tool.ApprovalResult{Approved: true}
			break
		} else if strings.ToUpper(nInput) == "N" {
			// Prompt for reason when denying
			fmt.Print("Please provide a reason for denial: ")
			scanner.Scan()
			reason := scanner.Text()
			fmt.Println()
			apResult = &tool.ApprovalResult{Approved: false, DisapproveReason: &reason}
			break
		}

		fmt.Println("invalid input, please input Y or N")
	}

	// here we directly resumes right in the same instance where the original `Runner.Query` happened.
	// In the real world, the original `Runner.Run/Query` and the subsequent `Runner.ResumeWithParams`
	// can happen in different processes or machines, as long as you use the same `CheckPointID`,
	// and you provided a distributed `CheckPointStore` when creating the `Runner` instance.
	iter, err := runner.ResumeWithParams(context.Background(), "1", &adk.ResumeParams{
		Targets: map[string]any{
			interruptID: apResult,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		if event.Err != nil {
			log.Fatal(event.Err)
		}

		prints.Event(event)
	}
}
