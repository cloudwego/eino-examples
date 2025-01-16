package einoagent

import (
	"context"
	"os"

	"github.com/cloudwego/eino-ext/components/embedding/ark"
	"github.com/cloudwego/eino/components/embedding"
)

func defaultArkEmbeddingConfig(ctx context.Context) (*ark.EmbeddingConfig, error) {
	config := &ark.EmbeddingConfig{
		Model:  os.Getenv("ARK_EMBEDDING_MODEL"),
		APIKey: os.Getenv("ARK_EMBEDDING_API_KEY"),
	}
	return config, nil
}

func NewArkEmbedding(ctx context.Context, config *ark.EmbeddingConfig) (eb embedding.Embedder, err error) {
	if config == nil {
		config, err = defaultArkEmbeddingConfig(ctx)
		if err != nil {
			return nil, err
		}
	}
	eb, err = ark.NewEmbedder(ctx, config)
	if err != nil {
		return nil, err
	}
	return eb, nil
}
