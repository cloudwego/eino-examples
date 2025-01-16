package knowledgeindexing

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/cloudwego/eino-ext/components/indexer/redis"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	redisCli "github.com/redis/go-redis/v9"
)

var (
	RedisPrefix = "eino:doc:"
	IndexName   = "vector_index"

	ContentField  = "content"
	MetadataField = "metadata"
	VectorField   = "content_vector"
)

func defaultRedisIndexerConfig(ctx context.Context) (*redis.IndexerConfig, error) {
	redisAddr := os.Getenv("REDIS_ADDR")
	redisClient := redisCli.NewClient(&redisCli.Options{
		Addr: redisAddr,
	})

	config := &redis.IndexerConfig{
		Client:    redisClient,
		KeyPrefix: RedisPrefix,
		BatchSize: 5,
		DocumentToHashes: func(ctx context.Context, doc *schema.Document) (*redis.Hashes, error) {
			key := doc.ID
			if doc.ID == "" {
				key = uuid.New().String()
			}

			metadataBytes, err := json.Marshal(doc.MetaData)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal metadata: %w", err)
			}

			return &redis.Hashes{
				Key: key,
				Field2Value: map[string]redis.FieldValue{
					ContentField:  {Value: doc.Content, EmbedKey: VectorField},
					MetadataField: {Value: metadataBytes},
				},
			}, nil
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
