package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/bytedance/sonic"
	"github.com/cloudwego/eino-ext/components/model/ark"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/components/tool/utils"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

func main() {
	ctx := context.Background()
	chatModel, err := ark.NewChatModel(ctx, &ark.ChatModelConfig{
		Model:  os.Getenv("ARK_MODEL"),
		APIKey: os.Getenv("ARK_API_KEY"),
	})
	if err != nil {
		log.Fatalf("ark.NewChatModel failed: %v\n", err)
	}

	getWeatherTool, err := utils.InferTool("get_weather", "Retrieves the current weather report for a specified city.", getWeatherInfo)
	if err != nil {
		log.Fatalf("utils.InferTool get_weather failed: %v\n", err)
	}

	sayHelloTool, err := utils.InferTool("say_hello", "Provides a simple greeting, optionally addressing the user by name.", sayHello)
	if err != nil {
		log.Fatalf("utils.InferTool say_hello failed: %v\n", err)
	}

	sayGoodbyeTool, err := utils.InferTool("say_goodbye", "Provides a simple farewell message to conclude the conversation.", sayGoodbye)
	if err != nil {
		log.Fatalf("utils.InferTool say_goodbye failed: %v\n", err)
	}

	greetingAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Model: chatModel,
		Instruction: `You are the Greeting Agent. Your ONLY task is to provide a friendly greeting to the user.
Use the 'say_hello' tool to generate the greeting.
If the user provides their name, make sure to pass it to the tool.
Do not engage in any other conversation or tasks.`,
		Name:        "greeting_agent",
		Description: "Handles simple greetings and hellos using the 'say_hello' tool.",
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{sayHelloTool},
			},
		},
	})
	if err != nil {
		log.Fatalf("NewChatModelAgent greeting_agent failed: %v\n", err)
	}

	farewellAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Model: chatModel,
		Instruction: `You are the Farewell Agent. Your ONLY task is to provide a polite goodbye message. "
Use the 'say_goodbye' tool when the user indicates they are leaving or ending the conversation "
(e.g., using words like 'bye', 'goodbye', 'thanks bye', 'see you'). "
Do not perform any other actions.`,
		Name:        "farewell_agent",
		Description: "Handles simple farewells and goodbyes using the 'say_goodbye' tool.",
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{sayGoodbyeTool},
			},
		},
	})
	if err != nil {
		log.Fatalf("NewChatModelAgent farewell_agent failed: %v\n", err)
	}

	weatherAgent, err := adk.NewChatModelAgent(ctx, &adk.ChatModelAgentConfig{
		Model: chatModel,
		Instruction: `You are a helpful weather assistant. "
When the user asks for the weather in a specific city, "
use the 'get_weather' tool to find the information. "
If the tool returns an error, inform the user politely. "
If the tool is successful, present the weather report clearly.`,
		Name:        "weather_agent",
		Description: "Provides weather information for specific cities.",
		ToolsConfig: adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{
				Tools: []tool.BaseTool{getWeatherTool},
			},
		},
	})
	if err != nil {
		log.Fatalf("NewChatModelAgent weather_agent failed: %v\n", err)
	}

	as, err := adk.SetSubAgents(ctx, weatherAgent, []adk.Agent{greetingAgent, farewellAgent})
	if err != nil {
		log.Fatalf("SetSubAgents failed: %v\n", err)
	}

	rn := adk.NewRunner(ctx, adk.RunnerConfig{
		EnableStreaming: false,
	})
	events := rn.Run(ctx, as, []adk.Message{schema.UserMessage("bye")})

	for i := 0; ; i++ {
		event, ok := events.Next()
		if !ok {
			break
		}

		log.Printf("agent: %s, eventIdx=%d\n", event.AgentName, i)

		if event.Err != nil {
			log.Printf("    error: %v\n", event.Err)
			continue
		}
		s, e := sonic.MarshalString(event)
		if e != nil {
			panic(e)
		}
		log.Printf("    event: %s\n", s)
	}
}

func buildSubAgents(ctx context.Context, chatModel model.ToolCallingChatModel) (adk.Agent, error) {

	return nil, nil
}

type City struct {
	Name string `json:"city_name" jsonschema:"description=the name of the city (e.g., \"Beijing\", \"London\")"`
}

func getWeatherInfo(ctx context.Context, city *City) (string, error) {
	return fmt.Sprintf("the weather in %s is good", city.Name), nil
}

type UserName struct {
	Name string `json:"user_name" jsonschema:"description=the name of the user (e.g., \"Alice\", \"Bob\")"`
}

func sayHello(ctx context.Context, user *UserName) (string, error) {
	return fmt.Sprintf("hello, %s", user.Name), nil
}

func sayGoodbye(ctx context.Context, none struct{}) (string, error) {
	return "goodbye", nil
}
