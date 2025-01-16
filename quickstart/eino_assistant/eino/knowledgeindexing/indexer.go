package knowledge_indexing

import (
	"context"

	"github.com/cloudwego/eino-ext/components/indexer/redis"
	"github.com/cloudwego/eino/components/indexer"
)

func defaultRedisIndexerConfig(ctx context.Context) (*redis.IndexerConfig, error) {
	config := &redis.IndexerConfig{
		KeyPrefix: "eino_assistant",
		BatchSize: 5}
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

func NewRedisIndexer(ctx context.Context, config *redis.IndexerConfig) (idr indexer.Indexer, err error) {
	if config == nil {
		config, err = defaultRedisIndexerConfig(ctx)
		if err != nil {
			return nil, err
		}
	}
	idr, err = redis.NewIndexer(ctx, config)
	if err != nil {
		return nil, err
	}
	return idr, nil
}
