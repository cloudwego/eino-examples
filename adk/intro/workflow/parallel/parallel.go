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

	"github.com/cloudwego/eino-examples/adk/common/prints"
	"github.com/cloudwego/eino-examples/adk/common/trace"
	"github.com/cloudwego/eino-examples/adk/intro/workflow/parallel/subagents"
	"github.com/cloudwego/eino/adk"
)

func main() {
	ctx := context.Background()

	traceCloseFn, startSpanFn := trace.AppendCozeLoopCallbackIfConfigured(ctx)
	defer traceCloseFn(ctx)

	a, err := adk.NewParallelAgent(ctx, &adk.ParallelAgentConfig{
		Name:        "DataCollectionAgent",
		Description: "Data Collection Agent could collect data from multiple sources.",
		SubAgents: []adk.Agent{
			subagents.NewStockDataCollectionAgent(),
			subagents.NewNewsDataCollectionAgent(),
			subagents.NewSocialMediaInfoCollectionAgent(),
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	query := "give me today's market research"
	ctx, endSpanFn := startSpanFn(ctx, "layered-supervisor", query)

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true, // you can disable streaming here
		Agent:           a,
	})

	iter := runner.Query(ctx, query)

	var lastMessage adk.Message

	for {
		event, ok := iter.Next()
		if !ok {
			break
		}
		if event.Err != nil {
			fmt.Printf("Error: %v\n", event.Err)
			break
		}

		prints.Event(event)
		if event.Output != nil {
			lastMessage, _, err = adk.GetMessage(event)
		}
	}

	endSpanFn(ctx, lastMessage)
}
