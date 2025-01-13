package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/cloudwego/eino-examples/agent/mem"
	eino_agent "github.com/cloudwego/eino-examples/eino_agentx"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

var memory = mem.GetDefaultMemory()

var cbHandler callbacks.Handler

var once sync.Once

func Init() error {
	var err error
	once.Do(func() {
		os.MkdirAll("log", 0755)
		var f *os.File
		f, err = os.OpenFile("log/eino.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return
		}

		cbConfig := &LogCallbackConfig{
			Detail: true,
			Writer: f,
		}
		if os.Getenv("DEBUG") == "true" {
			cbConfig.Debug = true
		}

		cbHandler = LogCallback(cbConfig)
	})
	return err
}

// func Run(ctx context.Context, userMessage *eino_agent.UserMessage) (*schema.StreamReader[*schema.Message], error) {
// 	graph, err := eino_agent.BuildAgentGraph(ctx, eino_agent.BuildConfig{})
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to build agent graph: %w", err)
// 	}

// 	runner, err := graph.Compile(ctx)
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to compile agent graph: %w", err)
// 	}

// 	// conversation := memory.GetConversation(id, true)
// 	// userMsg := schema.UserMessage(msg)
// 	// conversation.Append(userMsg)

// 	// msgs := conversation.GetMessages()

// 	sr, err := runner.Stream(ctx, userMessage, compose.WithCallbacks(cbHandler))
// 	if err != nil {
// 		return nil, fmt.Errorf("failed to stream: %w", err)
// 	}

// 	return sr, nil
// }

func RunAgent(ctx context.Context, id string, msg string) (*schema.StreamReader[*schema.Message], error) {

	runner, err := eino_agent.BuildAgentGraph(ctx, eino_agent.BuildConfig{
		AgentGraph: &eino_agent.AgentGraphBuildConfig{},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to build agent graph: %w", err)
	}

	conversation := memory.GetConversation(id, true)
	userMsg := schema.UserMessage(msg)
	conversation.Append(userMsg)

	userMessage := &eino_agent.UserMessage{
		ID:      id,
		Query:   msg,
		History: conversation.GetMessages(),
	}

	sr, err := runner.Stream(ctx, userMessage, compose.WithCallbacks(cbHandler))
	if err != nil {
		return nil, fmt.Errorf("failed to stream: %w", err)
	}

	srs := sr.Copy(2)

	go func() {
		// for save to memory
		fullMsgs := make([]*schema.Message, 0)

		defer func() {
			srs[1].Close()
			fullMsg, err := schema.ConcatMessages(fullMsgs)
			if err != nil {
				fmt.Println("error concatenating messages: ", err.Error())
			}
			conversation.Append(fullMsg)
		}()

	outter:
		for {
			select {
			case <-ctx.Done():
				fmt.Println("context done", ctx.Err())
				return
			default:
				chunk, err := srs[1].Recv()
				if err != nil {
					if errors.Is(err, io.EOF) {
						break outter
					}
				}

				fullMsgs = append(fullMsgs, chunk)
			}
		}
	}()

	return srs[0], nil
}
