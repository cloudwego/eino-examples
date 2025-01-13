package index_graph

import (
	"context"

	"github.com/cloudwego/eino-examples/agent/redis"
	"github.com/cloudwego/eino/components/indexer"
)

type RedisVectorStoreConfigImpl struct {
	redis.RedisVectorStoreConfigImpl
}

type RedisVectorStoreConfig struct {
	redis.RedisVectorStoreConfig
}

func defaultRedisVectorStoreConfig(ctx context.Context) (*RedisVectorStoreConfig, error) {
	config := &RedisVectorStoreConfig{}
	return config, nil
}

func NewRedisVectorStore(ctx context.Context, config *RedisVectorStoreConfig) (idx indexer.Indexer, err error) {
	if config == nil {
		config, err = defaultRedisVectorStoreConfig(ctx)
		if err != nil {
			return nil, err
		}
	}

	return redis.NewRedisVectorStore(ctx, &config.RedisVectorStoreConfig)
}
