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

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt"

	"github.com/cloudwego/eino-examples/adk/common/prints"
	"github.com/cloudwego/eino-examples/adk/multiagent/plan-execute-replan/agent"
	"github.com/cloudwego/eino-examples/adk/multiagent/plan-execute-replan/trace"
)

func main() {
	ctx := context.Background()

	traceCloseFn, client := trace.AppendCozeLoopCallbackIfConfigured(ctx)
	defer traceCloseFn(ctx)

	planAgent, err := agent.NewPlanner(ctx)
	if err != nil {
		log.Fatalf("agent.NewPlanner failed, err: %v", err)
	}

	executeAgent, err := agent.NewExecutor(ctx)
	if err != nil {
		log.Fatalf("agent.NewExecutor failed, err: %v", err)
	}

	replanAgent, err := agent.NewReplanAgent(ctx)
	if err != nil {
		log.Fatalf("agent.NewReplanAgent failed, err: %v", err)
	}

	entryAgent, err := prebuilt.NewPlanExecuteAgent(ctx, &prebuilt.PlanExecuteConfig{
		Planner:       planAgent,
		Executor:      executeAgent,
		Replanner:     replanAgent,
		MaxIterations: 20,
	})
	if err != nil {
		log.Fatalf("NewPlanExecuteAgent failed, err: %v", err)
	}

	r := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent: entryAgent,
	})

	query := `Plan a 3-day trip to Beijing in Next Month. I need flights from New York, hotel recommendations, and must-see attractions.
Today is 2025-09-09.`
	ctx, finishFn := trace.StartRootSpan(client, ctx, query)
	iter := r.Query(ctx, query)
	var lastMessage adk.Message
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		prints.Event(event)

		if event.Output != nil {
			lastMessage, _, err = adk.GetMessage(event)
		}
	}

	finishFn(ctx, lastMessage)
}
