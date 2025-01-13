package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"time"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

// usage:
// go run main.go -model=ep-xxxx -apikey=xxx -role=code_expert 'do you know cloudwego?'

var (
	modelName = flag.String("model", "", "The model to use, eg. ep-xxxx")
	apiKey    = flag.String("apikey", "", "The apikey of the model, eg. xxx")
	role      = flag.String("role", "code_expert", "The role to use, eg. code_expert")
)

func main() {
	flag.Parse()
	if *modelName == "" || *apiKey == "" {
		panic("model and apikey are required, you may get doubao model from: https://console.volcengine.com/ark/region:ark+cn-beijing/model/detail?Id=doubao-pro-32k")
	}

	ctx := context.Background()
	chain, err := NewSimpleLLM(ctx)
	if err != nil {
		panic(err)
	}

	arg1 := flag.Arg(0)
	if arg1 == "" {
		panic("message is required, eg: ./llm -model=ep-xxxx -apikey=xxx 'do you know cloudwego?'")
	}

	runner, err := chain.Compile(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Printf("\n=== START ===\n\n")

	sr, err := runner.Stream(ctx, map[string]any{
		"role": *role,
		"date": time.Now().Format("2006-01-02 15:04:05"),
		"conversations": []*schema.Message{
			schema.UserMessage(arg1),
		},
	})
	if err != nil {
		panic(err)
	}

	for {
		msg, err := sr.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			panic(err)
		}
		fmt.Print(msg.Content)
	}
	fmt.Printf("\n\n=== FINISH ===\n")
}

func NewSimpleLLM(ctx context.Context) (*compose.Chain[map[string]any, *schema.Message], error) {
	chain := compose.NewChain[map[string]any, *schema.Message]()

	// replace with your model: https://console.volcengine.com/ark/region:ark+cn-beijing/model/detail?Id=doubao-pro-32k
	model, err := PrepareModel(ctx, *modelName, *apiKey)
	if err != nil {
		return nil, err
	}

	template, err := PrepareTemplate(ctx)
	if err != nil {
		return nil, err
	}

	chain.AppendChatTemplate(template).AppendChatModel(model)

	return chain, nil
}

func PrepareTemplate(ctx context.Context) (prompt.ChatTemplate, error) {
	promptTemplate := `You are acting as a {role}.
You can only answer questions related to {role}, politely decline questions outside of this scope.
base info: time: {date}.`

	template := prompt.FromMessages(schema.FString, schema.SystemMessage(promptTemplate), schema.MessagesPlaceholder("conversations", false))

	return template, nil
}

func PrepareModel(ctx context.Context, modelName string, apiKey string) (model.ChatModel, error) {
	// 使用 ark 豆包大模型, or openai: openai.NewChatModel at github.com/cloudwego/eino-ext/components/model/openai
	model, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		Model:  modelName,
		APIKey: apiKey,
	})
	if err != nil {
		return nil, err
	}
	return model, nil
}
