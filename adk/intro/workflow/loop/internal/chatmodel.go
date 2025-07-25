package internal

import (
	"context"
	"log"
	"os"

	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
)

func newChatModel() model.ToolCallingChatModel {
	cm, err := openai.NewChatModel(context.Background(), &openai.ChatModelConfig{
		APIKey:  os.Getenv("OPENAI_API_KEY"),
		Model:   os.Getenv("OPENAI_MODEL"),
		BaseURL: os.Getenv("OPENAI_BASE_URL"),
		ByAzure: func() bool {
			if os.Getenv("OPENAI_BY_AZURE") == "true" {
				return true
			}
			return false
		}(),
	})
	if err != nil {
		log.Fatal(err)
	}
	return cm
}

func NewMainAgent() adk.Agent {
	a, err := adk.NewChatModelAgent(context.Background(), &adk.ChatModelAgentConfig{
		Name:        "MainAgent",
		Description: "Main agent that attempts to solve the user's task.",
		Instruction: `You are the main agent responsible for solving the user's task. 
Provide a comprehensive solution based on the given requirements. 
Focus on delivering accurate and complete results.`,
		Model: newChatModel(),
	})
	if err != nil {
		log.Fatal(err)
	}
	return a
}

func NewCritiqueAgent() adk.Agent {
	a, err := adk.NewChatModelAgent(context.Background(), &adk.ChatModelAgentConfig{
		Name:        "CritiqueAgent",
		Description: "Critique agent that reviews the main agent's work and provides feedback.",
		Instruction: `You are a critique agent responsible for reviewing the main agent's work.
Analyze the provided solution for accuracy, completeness, and quality.
If you find issues or areas for improvement, provide specific feedback.
If the work is satisfactory, call the 'exit' tool and provide a final summary response.`,
		Model: newChatModel(),
		// Exit:  nil, // use default exit tool
	})
	if err != nil {
		log.Fatal(err)
	}
	return a
}
