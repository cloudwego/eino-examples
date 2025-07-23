package main

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudwego/eino/adk"

	"github.com/cloudwego/eino-examples/adk/internal/prints"
	"github.com/cloudwego/eino-examples/adk/intro/workflow/loop/internal"
)

func main() {
	ctx := context.Background()

	a, err := adk.NewLoopAgent(ctx, &adk.LoopAgentConfig{
		Name:          "ReflectionAgent",
		Description:   "Reflection agent with main and critique agent for iterative task solving.",
		SubAgents:     []adk.Agent{internal.NewMainAgent(), internal.NewCritiqueAgent()},
		MaxIterations: 20,
	})
	if err != nil {
		log.Fatal(err)
	}

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: true, // you can disable streaming here
		Agent:           a,
	})

	iter := runner.Query(ctx,
		"tell me all the possible architectures of multi-modal embeddings, and analyze their pros and cons")
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
	}
}
