package main

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
)

func main() {

	ctx := context.Background()

	cm, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		Model:  os.Getenv("ARK_MODEL"),
		APIKey: os.Getenv("ARK_API_KEY"),
	})
	if err != nil {
		log.Fatalf("ark.NewChatModel failed: %v\n", err)
	}

	agent1, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "天气Agent",
		Description: "You are a helpful assistant",
		Instruction: "你只能回答天气相关的问题，超过范围内的回复我不知道",
		Model:       cm,
	})

	agent2, _ := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Name:        "BaseAskAgent",
		Description: "You are a helpful assistant",
		Instruction: "You are a helpful assistant",
		Model:       cm,
	})

	agent, err := adk.NewSequentialAgent(ctx, &adk.SequentialAgentConfig{
		Name:        "AskAgent",
		Description: "You are a helpful assistant",
		SubAgents: []adk.Agent{
			adk.AgentWithOptions(ctx, agent1, adk.WithDisallowTransferToParent()),
			agent2},
	})

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: false,
	})

	iter := runner.Run(ctx, agent, []adk.Message{
		schema.UserMessage("introduce china beijing."),
	})

	for {
		ev, hasNext := iter.Next()
		if !hasNext {
			break
		}

		if ev.Output != nil {
			msg, err_ := ev.Output.MessageOutput.GetMessage()
			if err_ != nil {
				log.Printf("GetMessage failed: %v\n", err_)
				continue
			}

			data, _ := json.MarshalIndent(msg, "", "  ")
			log.Printf("agent=%v\n", ev.AgentName)
			log.Printf("    msg: %v\n", string(data))
		}
	}
}
