package redis

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	defaultRedisKeyPrefix = "doc:"
	_indexName            = "vector_idx"
	contentField          = "content"
	metadataField         = "metadata"
	vectorField           = "content_vector"
	topK                  = 3
)

// RedisVectorStore implements both indexer.Indexer and retriever.Retriever interfaces
type RedisVectorStore struct {
	client    *redis.Client
	embedding embedding.Embedder
	prefix    string
	dimension int
	topK      int
	minScore  float64
}

type Config struct {
	RedisAddr      string
	Embedding      embedding.Embedder
	RedisKeyPrefix string
	Dimension      int
	TopK           int
	MinScore       float64
}

// float64ArrayToFloat32Bytes converts a float64 array to float32 bytes
func float64ArrayToFloat32Bytes(arr []float64) []byte {
	float32Arr := make([]float32, len(arr))
	for i, v := range arr {
		float32Arr[i] = float32(v)
	}

	bytes := make([]byte, len(float32Arr)*4)

	for i, v := range float32Arr {
		binary.LittleEndian.PutUint32(bytes[i*4:], math.Float32bits(v))
	}

	return bytes
}

// NewRedisVectorStore creates a new Redis vector store
func NewRedisVectorStore(ctx context.Context, config *Config) (store *RedisVectorStore, err error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if config.Embedding == nil {
		return nil, fmt.Errorf("embedding cannot be nil")
	}
	if config.Dimension <= 0 {
		return nil, fmt.Errorf("dimension must be positive")
	}
	if config.TopK <= 0 {
		config.TopK = topK
	}

	client := redis.NewClient(&redis.Options{
		Addr: config.RedisAddr,
	})

	// 确保在错误时关闭连接
	defer func() {
		if err != nil {
			client.Close()
		}
	}()

	if err = client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	prefix := defaultRedisKeyPrefix
	if config.RedisKeyPrefix != "" {
		prefix = config.RedisKeyPrefix
	}

	store = &RedisVectorStore{
		client:    client,
		embedding: config.Embedding,
		prefix:    prefix,
		dimension: config.Dimension,
		topK:      config.TopK,
	}

	indexName := fmt.Sprintf("%s:%s", prefix, _indexName)

	// 检查是否存在索引
	exists, err := client.Do(ctx, "FT.INFO", indexName).Result()
	if err != nil {
		if !strings.Contains(err.Error(), "Unknown index name") {
			return nil, fmt.Errorf("failed to check if index exists: %w", err)
		}
		err = nil
	} else if exists != nil {
		return store, nil
	}

	// Create new index
	createIndexArgs := []interface{}{
		"FT.CREATE", indexName,
		"ON", "HASH",
		"PREFIX", "1", prefix,
		"SCHEMA",
		contentField, "TEXT",
		metadataField, "TEXT",
		vectorField, "VECTOR", "FLAT",
		"6",
		"TYPE", "FLOAT32",
		"DIM", config.Dimension,
		"DISTANCE_METRIC", "COSINE",
	}

	if err = client.Do(ctx, createIndexArgs...).Err(); err != nil {
		return nil, fmt.Errorf("failed to create index: %w", err)
	}

	// 验证索引是否创建成功
	if _, err = client.Do(ctx, "FT.INFO", indexName).Result(); err != nil {
		return nil, fmt.Errorf("failed to verify index creation: %w", err)
	}

	return store, nil
}

// Store implements the indexer.Indexer interface
func (r *RedisVectorStore) Store(ctx context.Context, docs []*schema.Document, opts ...indexer.Option) ([]string, error) {
	if len(docs) == 0 {
		return []string{}, nil
	}

	ids := make([]string, len(docs))
	pipe := r.client.Pipeline()

	for i, doc := range docs {
		docID := uuid.New().String()
		ids[i] = docID

		vectors, err := r.embedding.EmbedStrings(ctx, []string{doc.Content})
		if err != nil {
			return nil, fmt.Errorf("failed to get embedding for document: %w", err)
		}
		vector := vectors[0]

		if len(vector) != r.dimension {
			return nil, fmt.Errorf("vector dimension mismatch: got %d, want %d", len(vector), r.dimension)
		}

		vectorBytes := float64ArrayToFloat32Bytes(vector)

		// Convert metadata to JSON string
		metadataBytes, err := json.Marshal(doc.MetaData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}

		key := r.prefix + docID
		pipe.HSet(ctx, key, map[string]interface{}{
			contentField:  doc.Content,
			metadataField: string(metadataBytes),
			vectorField:   vectorBytes,
		})
	}

	if _, err := pipe.Exec(ctx); err != nil {
		return nil, fmt.Errorf("failed to store documents: %w", err)
	}

	return ids, nil
}

// Retrieve implements the retriever.Retriever interface
func (r *RedisVectorStore) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	if query == "" {
		return []*schema.Document{}, nil
	}

	vectors, err := r.embedding.EmbedStrings(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding for query: %w", err)
	}
	queryVector := vectors[0]

	if len(queryVector) != r.dimension {
		return nil, fmt.Errorf("query vector dimension mismatch: got %d, want %d", len(queryVector), r.dimension)
	}

	indexName := fmt.Sprintf("%s:%s", r.prefix, _indexName)

	vectorBytes := float64ArrayToFloat32Bytes(queryVector)

	searchArgs := []interface{}{
		"FT.SEARCH", indexName,
		fmt.Sprintf("*=>[KNN %d @%s $BLOB AS distance]", r.topK, vectorField),
		"PARAMS", "2",
		"BLOB", vectorBytes,
		"RETURN", "4", contentField, metadataField, "distance", "id",
		"SORTBY", "distance",
		"DIALECT", "2",
	}

	results, err := r.client.Do(ctx, searchArgs...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to search documents: %w", err)
	}

	var docs []*schema.Document

	resultMap, ok := results.(map[interface{}]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected result type: %T", results)
	}

	totalResults, ok := resultMap["total_results"].(int64)
	if !ok {
		if f, ok := resultMap["total_results"].(float64); ok {
			totalResults = int64(f)
		} else {
			return nil, fmt.Errorf("unexpected total_results type: %T", resultMap["total_results"])
		}
	}

	if totalResults == 0 {
		return []*schema.Document{}, nil
	}

	resultsArray, ok := resultMap["results"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("unexpected results type: %T", resultMap["results"])
	}

	for _, result := range resultsArray {
		resultMap, ok := result.(map[interface{}]interface{})
		if !ok {
			continue
		}

		extraAttrs, ok := resultMap["extra_attributes"].(map[interface{}]interface{})
		if !ok {
			continue
		}

		content, _ := extraAttrs["content"].(string)
		distanceStr, _ := extraAttrs["distance"].(string)
		metadataStr, _ := extraAttrs["metadata"].(string)

		var distance float64
		if distanceStr != "" {
			if d, err := strconv.ParseFloat(distanceStr, 64); err == nil {
				distance = d
			}
		}

		id, _ := resultMap["id"].(string)
		if id != "" {
			id = strings.TrimPrefix(id, r.prefix)
		}

		if content != "" {
			doc := &schema.Document{
				ID:      id,
				Content: content,
				MetaData: map[string]interface{}{
					"distance": distance,
				},
			}

			// Parse metadata if exists
			if metadataStr != "" {
				var metadata map[string]interface{}
				if err := json.Unmarshal([]byte(metadataStr), &metadata); err == nil {
					for k, v := range metadata {
						doc.MetaData[k] = v
					}
				}
			}

			score := 1.0
			if distance < 0 {
				score = 1.0 + distance
			} else {
				score = 1.0 - distance
			}

			if score < r.minScore {
				continue
			}

			doc.WithScore(score)

			docs = append(docs, doc)
		}
	}

	return docs, nil
}
