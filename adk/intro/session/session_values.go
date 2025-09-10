package main

import (
	"context"
	"fmt"
	"log"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"

	"github.com/cloudwego/eino-examples/adk/common/model"
	"github.com/cloudwego/eino-examples/adk/common/prints"
)

func main() {
	ctx := context.Background()

	toolA, err := utils.InferTool("tool_a", "set user name", toolAFn)
	if err != nil {
		log.Fatalf("InferTool failed, err: %v", err)
	}

	toolB, err := utils.InferTool("tool_b", "set user age", toolBFn)
	if err != nil {
		log.Fatalf("InferTool failed, err: %v", err)
	}

	a, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "ChatModelAgent",
		Description: "A chat model agent",
		Instruction: "You are a chat model agent, call tool_a first, call tool_b secondly",
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{
					toolA,
					toolB,
				},
			},
		},
		Model: model.NewChatModel(),
	})
	if err != nil {
		log.Fatalf("NewChatModelAgent failed, err: %v", err)
	}

	r := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent: a,
	})

	iter := r.Query(ctx, "my name is Alice, my age is 18")
	for {
		event, ok := iter.Next()
		if !ok {
			break
		}

		prints.Event(event)
	}
}

type ToolAInput struct {
	Name string `json:"input" jsonschema:"description=user's name'"`
}

func toolAFn(ctx context.Context, in *ToolAInput) (string, error) {

	adk.AddSessionValue(ctx, "user-name", in.Name)
	return in.Name, nil
}

type ToolBInput struct {
	Age int `json:"input" jsonschema:"description=user's age'"`
}

func toolBFn(ctx context.Context, in *ToolBInput) (string, error) {
	adk.AddSessionValue(ctx, "user-age", in.Age)
	userName, _ := adk.GetSessionValue(ctx, "user-name")
	return fmt.Sprintf("user-name: %v, user-age: %v", userName, in.Age), nil
}
