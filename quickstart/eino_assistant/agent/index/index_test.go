package index

import (
	"context"
	"math/rand/v2"
	"testing"

	"github.com/cloudwego/eino/components/embedding"

	"github.com/cloudwego/eino-examples/agent/redis"
)

func NewMockEmbedder(num int, dim int) *MockEmbedder {
	embedding := make([][]float64, num)
	for i := 0; i < num; i++ {
		embedding[i] = make([]float64, dim)
		for j := 0; j < dim; j++ {
			embedding[i][j] = rand.Float64()
		}
	}

	return &MockEmbedder{embedding: embedding}
}

type MockEmbedder struct {
	embedding [][]float64
}

func (m *MockEmbedder) EmbedStrings(ctx context.Context, texts []string, opts ...embedding.Option) ([][]float64, error) {
	embeddings := make([][]float64, len(texts))
	for i := 0; i < len(texts); i++ {
		idx := rand.IntN(len(m.embedding))
		embeddings[i] = m.embedding[idx]
	}

	return embeddings, nil
}

func TestIndexMarkdown(t *testing.T) {
	ctx := context.Background()

	redisIndexer, err := redis.NewRedisVectorStore(ctx, &redis.Config{
		RedisAddr:      "localhost:6379",
		RedisKeyPrefix: "eino-test:markdown",
		Dimension:      1536,
		TopK:           3,
		MinScore:       0.5,
		Embedding:      NewMockEmbedder(4, 1536),
	})
	if err != nil {
		t.Fatalf("new redis vector store failed: %v", err)
	}

	err = IndexMarkdownFiles(ctx, "./eino-docs", redisIndexer)
	if err != nil {
		t.Fatalf("index markdown failed: %v", err)
	}
}
