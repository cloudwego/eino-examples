package main

import (
	"context"
	"log"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt"

	"github.com/cloudwego/eino-examples/adk/common/prints"
	"github.com/cloudwego/eino-examples/adk/multiagent/plan-execute-replan/agent"
	"github.com/cloudwego/eino-examples/adk/multiagent/plan-execute-replan/store"
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
		Agent:           entryAgent,
		CheckPointStore: store.NewInMemoryStore(),
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
