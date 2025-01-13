package agent

import (
	"context"
	"os"

	"github.com/cloudwego/eino-ext/components/embedding/ark"
	"github.com/cloudwego/eino/components/prompt"
	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"
)

type UserMessage struct {
	ID string `json:"id"`

	Content string            `json:"content"`
	History []*schema.Message `json:"history"`
}

func NewGraph(ctx context.Context) (*compose.Graph[*UserMessage, *schema.Message], error) {
	graph := compose.NewGraph[*UserMessage, *schema.Message]()

	graph.AddLambdaNode("convert", compose.InvokableLambda(func(ctx context.Context, msg *UserMessage) (map[string]any, error) {
		return map[string]any{
			"id":      msg.ID,
			"content": msg.Content,
			"history": msg.History,
		}, nil
	}))

	template := prompt.FromMessages(
		schema.FString,
		schema.MessagesPlaceholder("history", true),
		schema.UserMessage("{content}"),
	)

	embedding, err := ark.NewEmbedder(ctx, &ark.EmbeddingConfig{
		Model:  os.Getenv("ARK_EMBEDDING_MODEL"),
		APIKey: os.Getenv("ARK_API_KEY"),
	})
	if err != nil {
		return nil, err
	}

	retrieverNode := Retriever(ctx, embedding)

	graph.AddRetrieverNode("retriever", retrieverNode)

	graph.AddChatTemplateNode("template", template)

	graph.AddLambdaNode("react_agent", compose.StreamableLambda(func(ctx context.Context, msgs []*schema.Message) (*schema.StreamReader[*schema.Message], error) {
		model := os.Getenv("ARK_CHAT_MODEL")
		apiKey := os.Getenv("ARK_API_KEY")

		agent, err := NewAgent(ctx, `You are a helpful assistant.`, model, apiKey)
		if err != nil {
			return nil, err
		}

		return agent.Stream(ctx, msgs)
	}))

	return graph, nil
}
