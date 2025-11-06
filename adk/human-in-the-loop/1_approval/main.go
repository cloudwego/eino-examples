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
	a := NewTicketBookingAgent()
	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true, // you can disable streaming here
		Agent:           a,
		CheckPointStore: store.NewInMemoryStore(),
	})
	iter := runner.Query(ctx, "book a ticket for Martin, to Beijing, on 2025-12-01, the phone number is 1234567. directly call tool.", adk.WithCheckPointID("1"))

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
	apInfo := lastEvent.Action.Interrupted.InterruptContexts[0].Info.(*tool.ApprovalInfo)
	interruptID := lastEvent.Action.Interrupted.InterruptContexts[0].ID

	for {
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Print("your input here: ")
		scanner.Scan()
		fmt.Println()
		nInput := scanner.Text()
		if strings.ToUpper(nInput) == "Y" {
			apInfo.ApprovalResult = &tool.ApprovalResult{Approved: true}
			break
		} else if strings.ToUpper(nInput) == "N" {
			// Prompt for reason when denying
			fmt.Print("Please provide a reason for denial: ")
			scanner.Scan()
			reason := scanner.Text()
			fmt.Println()
			apInfo.ApprovalResult = &tool.ApprovalResult{Approved: false, DisapproveReason: &reason}
			break
		}

		fmt.Println("invalid input, please input Y or N")
	}

	iter, err := runner.TargetedResume(ctx, "1", map[string]any{
		interruptID: apInfo,
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
