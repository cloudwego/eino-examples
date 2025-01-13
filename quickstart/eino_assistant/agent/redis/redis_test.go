package redis

import (
	"context"
	"testing"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/schema"
	"github.com/stretchr/testify/assert"
)

// MockEmbedder implements embedding.Embedder interface for testing
type MockEmbedder struct{}

func (m *MockEmbedder) EmbedStrings(ctx context.Context, texts []string, opts ...embedding.Option) ([][]float64, error) {
	// Return mock embeddings with dimension 1536 for testing
	embeddings := make([][]float64, len(texts))
	for i := range texts {
		// Create a vector with 1536 dimensions
		vector := make([]float64, 1536)
		for j := range vector {
			vector[j] = 0.1 // Simple value for testing
		}
		embeddings[i] = vector
	}
	return embeddings, nil
}

func TestRedisVectorStore(t *testing.T) {
	ctx := context.Background()

	// Create Redis vector store with mock embedder
	store, err := NewRedisVectorStore(ctx, &Config{
		RedisAddr: "localhost:6379",
		Embedding: &MockEmbedder{},
		Dimension: 1536,
		TopK:      3,
	})
	assert.NoError(t, err)
	assert.NotNil(t, store)

	// Test storing documents
	docs := []*schema.Document{
		{
			Content: "test document 1",
			MetaData: map[string]interface{}{
				"type": "doc1",
				"tag":  "test",
			},
		},
		{
			Content: "test document 2",
			MetaData: map[string]interface{}{
				"type": "doc2",
				"tag":  "test",
			},
		},
		{
			Content: "test document 3",
			MetaData: map[string]interface{}{
				"type": "doc3",
				"tag":  "test",
			},
		},
	}

	ids, err := store.Store(ctx, docs)
	assert.NoError(t, err)
	assert.Equal(t, len(docs), len(ids))

	// Test retrieving documents
	query := "test query"
	results, err := store.Retrieve(ctx, query)
	assert.NoError(t, err)
	assert.NotEmpty(t, results)

	// Verify returned documents
	for _, doc := range results {
		assert.NotEmpty(t, doc.ID)
		assert.NotEmpty(t, doc.Content)
		assert.NotNil(t, doc.MetaData)
		assert.Contains(t, doc.MetaData, "distance")
		assert.Contains(t, doc.MetaData, "type")
		assert.Contains(t, doc.MetaData, "tag")
		score := doc.Score()
		assert.GreaterOrEqual(t, score, float64(0))
		assert.LessOrEqual(t, score, float64(1))
	}
}

func TestRedisVectorStoreEmptyInput(t *testing.T) {
	ctx := context.Background()

	store, err := NewRedisVectorStore(ctx, &Config{
		RedisAddr: "localhost:6379",
		Embedding: &MockEmbedder{},
		Dimension: 1536,
		TopK:      3,
	})
	assert.NoError(t, err)

	// Test empty documents
	ids, err := store.Store(ctx, []*schema.Document{})
	assert.NoError(t, err)
	assert.Empty(t, ids)

	// Test empty query
	results, err := store.Retrieve(ctx, "")
	assert.NoError(t, err)
	assert.Empty(t, results)
}

func TestRedisVectorStoreInvalidConfig(t *testing.T) {
	ctx := context.Background()

	// Test invalid Redis address
	_, err := NewRedisVectorStore(ctx, &Config{
		RedisAddr: "invalid:6379",
		Embedding: &MockEmbedder{},
		Dimension: 1536,
		TopK:      3,
	})
	assert.Error(t, err)
}
