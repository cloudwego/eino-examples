package einoagent

import (
	"context"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/retriever/redis"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	redisCli "github.com/redis/go-redis/v9"
)

var (
	RedisPrefix = "eino:doc:"
	IndexName   = "vector_index"

	ContentField  = "content"
	MetadataField = "metadata"
	VectorField   = "content_vector"
)

func defaultRedisRetrieverConfig(ctx context.Context) (*redis.RetrieverConfig, error) {
	redisAddr := os.Getenv("REDIS_ADDR")
	redisClient := redisCli.NewClient(&redisCli.Options{
		Addr:     redisAddr,
		Protocol: 2,
	})

	config := &redis.RetrieverConfig{
		Client:       redisClient,
		Index:        fmt.Sprintf("%s:%s", RedisPrefix, IndexName),
		Dialect:      2,
		ReturnFields: []string{ContentField, MetadataField},
		TopK:         4,
		DocumentConverter: func(ctx context.Context, doc redisCli.Document) (*schema.Document, error) {
			resp := &schema.Document{
				ID:       doc.ID,
				Content:  "",
				MetaData: map[string]any{},
			}
			for field, val := range doc.Fields {
				if field == ContentField {
					resp.Content = val
				} else if field == MetadataField {
					resp.MetaData[field] = val
				}
			}

			return resp, nil
		},
	}
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
