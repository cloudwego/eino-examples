package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"

	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino-ext/components/tool/duckduckgo"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
	"github.com/cloudwego/eino/flow/agent/react"
	"github.com/cloudwego/eino/schema"
)

// usage:
// go run main.go -model=ep-xxxx -apikey=xxx 'do you know cloudwego, and what is the url of cloudwego? search for me please'

var (
	// you can get model from: https://console.volcengine.com/ark/region:ark+cn-beijing/model/detail?Id=doubao-pro-32k
	modelName = flag.String("model", "", "The model to use, eg. ep-xxxx")
	apiKey    = flag.String("apikey", "", "The apikey of the model, eg. xxx")
)

func main() {
	flag.Parse()

	ctx := context.Background()
	reactAgent, err := NewAgent(ctx)
	if err != nil {
		panic(err)
	}

	arg := flag.Arg(0)
	if arg == "" {
		panic("message is required, eg: ./llm -model=ep-xxxx -apikey=xxx 'do you know cloudwego?'")
	}

	sr, err := reactAgent.Stream(ctx, []*schema.Message{
		schema.UserMessage(arg),
	}, agent.WithComposeOptions(compose.WithCallbacks(LogCallback())))
	if err != nil {
		panic(err)
	}

	for {
		msg, err := sr.Recv()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			panic(err)
		}
		fmt.Print(msg.Content)
	}
	fmt.Printf("\n\n=== %sFINISHED%s ===\n\n", green, reset)
}

func NewAgent(ctx context.Context) (*react.Agent, error) {

	// 初始化模型
	model, err := PrepareModel(ctx)
	if err != nil {
		return nil, err
	}

	// 初始化各种 tool
	tools, err := PrepareTools(ctx)
	if err != nil {
		return nil, err
	}

	// 初始化 agent
	agent, err := react.NewAgent(ctx, &react.AgentConfig{
		Model: model,
		ToolsConfig: compose.ToolsNodeConfig{
			Tools: tools,
		},
	})
	if err != nil {
		return nil, err
	}
	return agent, nil
}

func PrepareModel(ctx context.Context) (model.ChatModel, error) {

	// eg. 使用 ark 豆包大模型, or openai: openai.NewChatModel at github.com/cloudwego/eino-ext/components/model/openai
	arkModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		Model:  *modelName, // replace with your model
		APIKey: *apiKey,    // replace with your api key
	})
	if err != nil {
		return nil, err
	}
	return arkModel, nil
}

func PrepareTools(ctx context.Context) ([]tool.BaseTool, error) {
	duckduckgo, err := duckduckgo.NewTool(ctx, &duckduckgo.Config{})
	if err != nil {
		return nil, err
	}
	return []tool.BaseTool{duckduckgo}, nil
}

// log with color
var (
	green = "\033[32m"
	reset = "\033[0m"
)

func LogCallback() callbacks.Handler {
	builder := callbacks.NewHandlerBuilder()
	builder.OnStartFn(func(ctx context.Context, info *callbacks.RunInfo, input callbacks.CallbackInput) context.Context {
		fmt.Printf("%s[view]%s: start [%s:%s:%s]\n", green, reset, info.Component, info.Type, info.Name)
		return ctx
	})
	builder.OnEndFn(func(ctx context.Context, info *callbacks.RunInfo, output callbacks.CallbackOutput) context.Context {
		fmt.Printf("%s[view]%s: end [%s:%s:%s]\n", green, reset, info.Component, info.Type, info.Name)
		return ctx
	})
	return builder.Build()
}
