package agent

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/cloudwego/eino-examples/agent/mem"
	"github.com/cloudwego/eino/callbacks"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/flow/agent"
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

func RunAgent(ctx context.Context, id string, msg string) (*schema.StreamReader[*schema.Message], error) {

	reactAgent, err := NewAgent(ctx, "You are an expert in golang.", os.Getenv("ARK_CHAT_MODEL"), os.Getenv("ARK_API_KEY"))
	if err != nil {
		return nil, err
	}

	conversation := memory.GetConversation(id, true)
	userMsg := schema.UserMessage(msg)
	conversation.Append(userMsg)

	msgs := conversation.GetMessages()

	sr, err := reactAgent.Stream(ctx, msgs, agent.WithComposeOptions(compose.WithCallbacks(cbHandler)))
	if err != nil {
		return nil, err
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
