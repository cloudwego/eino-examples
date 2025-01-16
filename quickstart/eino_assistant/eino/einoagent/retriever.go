package eino_agent

import (
	"context"

	"github.com/cloudwego/eino-ext/components/retriever/redis"
	"github.com/cloudwego/eino/components/retriever"
)

func defaultRedisRetrieverConfig(ctx context.Context) (*redis.RetrieverConfig, error) {
	config := &redis.RetrieverConfig{
		ReturnFields: []string{}}
	embeddingCfg11, err := defaultArkEmbeddingConfig(ctx)
	if err != nil {
		return nil, err
	}
	embeddingIns11, err := NewArkEmbedding(ctx, embeddingCfg11)
	if err != nil {
		return nil, err
	}
	config.Embedding = embeddingIns11
	return config, nil
}

func NewRedisRetriever(ctx context.Context, config *redis.RetrieverConfig) (rtr retriever.Retriever, err error) {
	if config == nil {
		config, err = defaultRedisRetrieverConfig(ctx)
		if err != nil {
			return nil, err
		}
	}
	rtr, err = redis.NewRetriever(ctx, config)
	if err != nil {
		return nil, err
	}
	return rtr, nil
}
