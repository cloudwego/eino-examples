package eino_agent

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-examples/agent/redis"
	"github.com/cloudwego/eino-ext/components/embedding/ark"
	"github.com/cloudwego/eino/components/retriever"
)

type EinoRetrieverImpl struct {
	redis.RedisVectorStoreConfigImpl
}

type EinoRetrieverConfig struct {
	redis.RedisVectorStoreConfig
}

func defaultEinoRetrieverConfig(ctx context.Context) (*EinoRetrieverConfig, error) {
	embedding, err := ark.NewEmbedder(ctx, &ark.EmbeddingConfig{
		Model:  os.Getenv("ARK_EMBEDDING_MODEL"),
		APIKey: os.Getenv("ARK_API_KEY"),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create embedding: %w", err)
	}

	config := &EinoRetrieverConfig{
		RedisVectorStoreConfig: redis.RedisVectorStoreConfig{
			RedisAddr:      "127.0.0.1:6379",
			Embedding:      embedding,
			RedisKeyPrefix: "eino:doc:",
			Dimension:      4096,
			TopK:           3,
			MinScore:       0.65,
		},
	}
	return config, nil
}

func NewEinoRetriever(ctx context.Context, config *EinoRetrieverConfig) (rt retriever.Retriever, err error) {
	if config == nil {
		config, err = defaultEinoRetrieverConfig(ctx)
		if err != nil {
			return nil, err
		}
	}

	return redis.NewRedisVectorStore(ctx, &config.RedisVectorStoreConfig)
}
