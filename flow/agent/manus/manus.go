/*
 * Copyright 2024 CloudWeGo Authors
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
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/cloudwego/eino-ext/callbacks/langfuse"
	"github.com/cloudwego/eino-ext/components/model/openai"
	"github.com/cloudwego/eino-ext/components/tool/browseruse"
	"github.com/cloudwego/eino-ext/components/tool/commandline"
	"github.com/cloudwego/eino-ext/components/tool/commandline/sandbox"
	"github.com/cloudwego/eino-ext/components/tool/duckduckgo/ddgsearch"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

var (
	langfuseHost      string
	langfusePublicKey string
	langfuseSecretKey string

	openaiAPIKey  string
	openaiModel   string
	openaiBaseURL string

	input string
)

func init() {
	langfuseHost = os.Getenv("LANGFUSE_HOST")
	langfusePublicKey = os.Getenv("LANGFUSE_PUBLIC_KEY")
	langfuseSecretKey = os.Getenv("LANGFUSE_SECRET_KEY")

	openaiAPIKey = os.Getenv("OPENAI_API_KEY")
	openaiModel = os.Getenv("OPENAI_MODEL")
	openaiBaseURL = os.Getenv("OPENAI_BASE_URL")

	input = "what is eino?"
}

func main() {
	ctx := context.Background()

	// init tools
	sb := newSandbox(ctx)
	defer sb.Cleanup(ctx)
	commandlineTools := newCommandLineTools(ctx, sb)
	browserTool := newBrowserTool(ctx)
	defer browserTool.Cleanup()

	// init chat model
	cm := newChatModel(ctx)
	cm = bindTools(ctx, cm, append(commandlineTools, browserTool))

	// init and register callback handlers for logging and tracing
	var handlers []callbacks.Handler
	if len(langfuseHost) > 0 {
		langfuseHandler, flusher := newLangfuseHandler()
		handlers = append(handlers, langfuseHandler)
		defer flusher()
	}
	handlers = append(handlers, newLogHandler())
	callbacks.AppendGlobalHandlers(handlers...)

	// compose graph
	agent := composeAgent(ctx, cm, browserTool, commandlineTools)

	// init langfuse trace
	ctx = langfuse.SetTrace(ctx, langfuse.WithID(uuid.New().String()))

	var userInput string
	for {
		result, err := agent.Invoke(ctx, input,
			compose.WithCheckPointID("1"),
			compose.WithStateModifier(func(ctx context.Context, path compose.NodePath, s any) error {
				s.(*state).UserInput = userInput
				return nil
			}),
			compose.WithRuntimeMaxSteps(20),
		)
		info, ok := compose.ExtractInterruptInfo(err)
		if ok {
			s := info.State.(*state)
			fmt.Printf("ChatModel Output: %s\n", s.History[len(s.History)-1].Content)
			fmt.Print("Do you want to continue? (y/n): ")
			reader := bufio.NewReader(os.Stdin)
			response, _ := reader.ReadString('\n')
			response = strings.TrimSpace(response)

			if strings.ToLower(response) == "y" {
				fmt.Print("Please enter your query: ")
				userInput, _ = reader.ReadString('\n')
				userInput = strings.TrimSpace(userInput)
			} else {
				userInput = ""
			}
			continue
		}
		if err != nil {
			log.Printf("agent run error: %v", err)
			return
		}
		fmt.Printf("[FinalResult]: %s", result)
		break
	}
}

type state struct {
	History   []*schema.Message
	UserInput string
}

const (
	NodeKeyInputConvert  = "InputConverter"
	NodeKeyChatModel     = "ChatModel"
	NodeKeyToolsNode     = "ToolsNode"
	NodeKeyHuman         = "Human"
	NodeKeyOutputConvert = "OutputConverter"
)

func composeAgent(ctx context.Context,
	cm model.BaseChatModel,
	browserTool *browseruse.Tool,
	tools []tool.BaseTool,
) compose.Runnable[string, string] {
	err := compose.RegisterSerializableType[state]("my state")
	if err != nil {
		log.Fatal(err)
	}
	err = compose.RegisterSerializableType[schema.ChatMessagePartType]("cmpt")
	if err != nil {
		log.Fatal(err)
	}
	err = compose.RegisterSerializableType[schema.ChatMessageImageURL]("cmiu")
	if err != nil {
		log.Fatal(err)
	}
	err = compose.RegisterSerializableType[schema.ChatMessageAudioURL]("cnau")
	if err != nil {
		log.Fatal(err)
	}
	err = compose.RegisterSerializableType[schema.ChatMessageVideoURL]("cmvu")
	if err != nil {
		log.Fatal(err)
	}
	err = compose.RegisterSerializableType[schema.ChatMessageFileURL]("cmfu")
	if err != nil {
		log.Fatal(err)
	}
	err = compose.RegisterSerializableType[schema.ImageURLDetail]("iud")
	if err != nil {
		log.Fatal(err)
	}

	g := compose.NewGraph[string, string](compose.WithGenLocalState(func(ctx context.Context) *state {
		return &state{History: []*schema.Message{}}
	}))

	// register nodes
	err = g.AddLambdaNode(NodeKeyInputConvert, compose.InvokableLambda(func(ctx context.Context, input string) (output []*schema.Message, err error) {
		return []*schema.Message{
			schema.SystemMessage(systemPrompt),
			schema.UserMessage(input),
		}, nil
	}), compose.WithNodeName(NodeKeyInputConvert))
	if err != nil {
		log.Fatal(err)
	}

	err = g.AddChatModelNode(
		NodeKeyChatModel,
		cm,
		compose.WithNodeName(NodeKeyChatModel),
		// append other node's output to History and load History to llm input
		compose.WithStatePreHandler(func(ctx context.Context, in []*schema.Message, state *state) ([]*schema.Message, error) {
			state.History = append(state.History, in...)
			return state.History, nil
		}),
		compose.WithStatePostHandler(func(ctx context.Context, out *schema.Message, state *state) (*schema.Message, error) {
			state.History = append(state.History, out)
			return out, nil
		}),
	)
	if err != nil {
		log.Fatal(err)
	}

	toolsNode, err := compose.NewToolNode(ctx, &compose.ToolsNodeConfig{Tools: append(tools, browserTool)})
	if err != nil {
		log.Fatal(err)
	}
	err = g.AddToolsNode(
		NodeKeyToolsNode,
		toolsNode,
		compose.WithNodeName(NodeKeyToolsNode),
		compose.WithStatePostHandler(appendNextPrompt(ctx, browserTool)),
	)
	if err != nil {
		log.Fatal(err)
	}

	err = g.AddLambdaNode(NodeKeyHuman, compose.InvokableLambda(func(ctx context.Context, input *schema.Message) (output []*schema.Message, err error) {
		return []*schema.Message{input}, nil
	}), compose.WithNodeName(NodeKeyHuman),
		compose.WithStatePostHandler(func(ctx context.Context, in []*schema.Message, state *state) ([]*schema.Message, error) {
			if len(state.UserInput) > 0 {
				return []*schema.Message{schema.UserMessage(state.UserInput)}, nil
			}
			return in, nil
		}))
	if err != nil {
		log.Fatal(err)
	}

	err = g.AddLambdaNode(NodeKeyOutputConvert, compose.InvokableLambda(func(ctx context.Context, input []*schema.Message) (output string, err error) {
		return input[len(input)-1].Content, nil
	}))
	if err != nil {
		log.Fatal(err)
	}

	// compose graph
	err = g.AddEdge(compose.START, NodeKeyInputConvert)
	if err != nil {
		log.Fatal(err)
	}
	err = g.AddEdge(NodeKeyInputConvert, NodeKeyChatModel)
	if err != nil {
		log.Fatal(err)
	}
	err = g.AddBranch(NodeKeyChatModel, compose.NewGraphBranch(func(ctx context.Context, in *schema.Message) (endNode string, err error) {
		if len(in.ToolCalls) > 0 {
			return NodeKeyToolsNode, nil
		}
		return NodeKeyHuman, nil
	}, map[string]bool{
		NodeKeyToolsNode: true,
		NodeKeyHuman:     true,
	}))
	if err != nil {
		log.Fatal(err)
	}
	err = g.AddBranch(NodeKeyHuman, compose.NewGraphBranch(func(ctx context.Context, in []*schema.Message) (endNode string, err error) {
		if in[len(in)-1].Role == schema.User {
			return NodeKeyChatModel, nil
		}
		return NodeKeyOutputConvert, nil
	}, map[string]bool{
		NodeKeyChatModel:     true,
		NodeKeyOutputConvert: true,
	}))
	err = g.AddEdge(NodeKeyToolsNode, NodeKeyChatModel)
	if err != nil {
		log.Fatal(err)
	}
	err = g.AddEdge(NodeKeyOutputConvert, compose.END)
	if err != nil {
		log.Fatal(err)
	}

	runner, err := g.Compile(ctx, compose.WithCheckPointStore(newInMemoryStore()), compose.WithInterruptBeforeNodes([]string{NodeKeyHuman}))
	if err != nil {
		log.Fatal(err)
	}

	return runner
}

func bindTools(ctx context.Context, cm model.ToolCallingChatModel, tools []tool.BaseTool) model.ToolCallingChatModel {
	infos := make([]*schema.ToolInfo, 0, len(tools))
	for _, t := range tools {
		info, err := t.Info(ctx)
		if err != nil {
			log.Fatal("get tool info of fail: ", err)
		}
		infos = append(infos, info)
	}

	ncm, err := cm.WithTools(infos)
	if err != nil {
		log.Fatal("bind tools fail: ", err)
	}
	return ncm
}

func appendNextPrompt(ctx context.Context, browserTool *browseruse.Tool) func(ctx context.Context, toolsNodeOutput []*schema.Message, state *state) ([]*schema.Message, error) {
	info, err := browserTool.Info(ctx)
	if err != nil {
		log.Fatal("get browser tool info fail: ", err)
	}
	return func(ctx context.Context, toolsNodeOutput []*schema.Message, state *state) ([]*schema.Message, error) {
		// append next prompt step prompt
		// if call browser tool -> get browser state and append
		// else append common prompt
		if len(state.History) == 0 {
			return toolsNodeOutput, nil
		}

		llmToolCallMessage := state.History[len(state.History)-1]
		for _, tc := range llmToolCallMessage.ToolCalls {
			if tc.Function.Name == info.Name {
				bState, err := browserTool.GetCurrentState()
				if err != nil {
					return nil, fmt.Errorf("failed to get browser tool state: %w", err)
				}
				bPrompt, err := formatBrowserToolPrompt(ctx, bState)
				if err != nil {
					return nil, fmt.Errorf("failed to format browser tool prompt: %w", err)
				}
				return append(toolsNodeOutput, bPrompt), nil
			}
		}

		return append(toolsNodeOutput, schema.UserMessage(nextStepPrompt)), nil
	}
}

func formatBrowserToolPrompt(ctx context.Context, bs *browseruse.BrowserState) (*schema.Message, error) {
	if bs == nil {
		return nil, fmt.Errorf("browser state is nil")
	}
	messages, err := schema.UserMessage(browserNextStepPrompt).Format(ctx, map[string]any{
		"url_placeholder":           bs.URL,
		"tabs_placeholder":          fmt.Sprintf("%+v", bs.Tabs),
		"content_above_placeholder": bs.ScrollInfo.PixelsAbove,
		"content_below_placeholder": bs.ScrollInfo.PixelsBelow,
		"results_placeholder":       "",
	}, schema.FString)
	if err != nil {
		return nil, err
	}
	message := messages[0]
	if len(bs.Screenshot) > 0 {
		message = schema.UserMessage("")
		message.MultiContent = append(message.MultiContent,
			schema.ChatMessagePart{
				Type: schema.ChatMessagePartTypeText,
				Text: messages[0].Content,
			},
			schema.ChatMessagePart{
				Type: schema.ChatMessagePartTypeText,
				Text: "Current browser screenshot:",
			},
			schema.ChatMessagePart{
				Type: schema.ChatMessagePartTypeImageURL,
				ImageURL: &schema.ChatMessageImageURL{
					URL: "data:image/png;base64," + bs.Screenshot,
				},
			})
	}
	return message, nil
}

func newLangfuseHandler() (*langfuse.CallbackHandler, func()) {
	return langfuse.NewLangfuseHandler(&langfuse.Config{
		Host:      langfuseHost,
		PublicKey: langfusePublicKey,
		SecretKey: langfuseSecretKey,
	})
}

func newChatModel(ctx context.Context) model.ToolCallingChatModel {
	var cm model.ToolCallingChatModel
	var err error
	var temp float32 = 0
	cm, err = openai.NewChatModel(ctx, &openai.ChatModelConfig{
		APIKey:      openaiAPIKey,
		BaseURL:     openaiBaseURL,
		Model:       openaiModel,
		Temperature: &temp,
	})
	if err != nil {
		log.Fatal(err)
	}
	return cm
}

func newSandbox(ctx context.Context) *sandbox.DockerSandbox {
	sb, err := sandbox.NewDockerSandbox(ctx, &sandbox.Config{
		Image:          "python:3.9-slim",
		HostName:       "sandbox",
		WorkDir:        "/workspace",
		MemoryLimit:    512 * 1024 * 1024,
		CPULimit:       1.0,
		NetworkEnabled: false,
		Timeout:        time.Second * 30,
	})
	if err != nil {
		log.Fatal(err)
	}
	err = sb.Create(ctx)
	if err != nil {
		log.Fatal(err)
	}
	return sb
}

func newCommandLineTools(ctx context.Context, sb commandline.Operator) []tool.BaseTool {
	et, err := commandline.NewStrReplaceEditor(ctx, &commandline.EditorConfig{Operator: sb})
	if err != nil {
		log.Fatal(err)
	}
	pt, err := commandline.NewPyExecutor(ctx, &commandline.PyExecutorConfig{Command: "python3", Operator: sb})
	if err != nil {
		log.Fatal(err)
	}
	return []tool.BaseTool{et, pt}
}

func newBrowserTool(ctx context.Context) *browseruse.Tool {
	ddgs, err := ddgsearch.New(&ddgsearch.Config{Timeout: time.Second * 30})
	if err != nil {
		log.Fatal(err)
	}

	t, err := browseruse.NewBrowserUseTool(ctx, &browseruse.Config{
		Headless:           false,
		DisableSecurity:    false,
		ExtraChromiumArgs:  nil,
		ChromeInstancePath: "",
		ProxyServer:        "",
		DDGSearchTool:      ddgs,
		ExtractChatModel:   newChatModel(ctx),
		Logf:               log.Printf,
	})
	if err != nil {
		log.Fatal(err)
	}
	return t
}

func newInMemoryStore() *inMemoryStore {
	return &inMemoryStore{m: make(map[string][]byte)}
}

type inMemoryStore struct {
	m map[string][]byte
}

func (i *inMemoryStore) Get(ctx context.Context, checkPointID string) ([]byte, bool, error) {
	data, ok := i.m[checkPointID]
	return data, ok, nil
}

func (i *inMemoryStore) Set(ctx context.Context, checkPointID string, checkPoint []byte) error {
	i.m[checkPointID] = checkPoint
	return nil
}
