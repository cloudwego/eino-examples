package main

import (
	"context"
	"fmt"
	"log"
	"testing"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"

	adk2 "github.com/cloudwego/eino-examples/adk"
)

func TestSupervisor(t *testing.T) {
	sv, err := buildSupervisor(context.Background())
	if err != nil {
		log.Fatalf("build superviser failed: %v", err)
	}

	query := "find US and New York state GDP in 2024. what % of US GDP was New York state?"

	iter := sv.Run(context.Background(), &adk.AgentInput{
		Messages: []adk.Message{
			schema.UserMessage(query),
		},
		EnableStreaming: true,
	})

	fmt.Println("\nuser query: ", query)

	for {
		event, hasEvent := iter.Next()
		if !hasEvent {
			break
		}

		adk2.PrintEvent(event)
	}
}
