/*
 * Copyright 2024 CloudWeGo Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package redis

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"

	ri "github.com/cloudwego/eino-ext/components/indexer/redis"
	rr "github.com/cloudwego/eino-ext/components/retriever/redis"
	"github.com/cloudwego/eino/components/embedding"
	"github.com/cloudwego/eino/components/indexer"
	"github.com/cloudwego/eino/components/retriever"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	defaultRedisKeyPrefix = "doc:"
	indexNamePrefix       = "vector_idx"
	contentField          = "content"
	metadataField         = "metadata"
	vectorField           = "content_vector"
	topK                  = 3
)

func NewRedisIndexer(ctx context.Context, config *RedisVectorStoreConfig) (i indexer.Indexer, err error) {
	if config == nil {
		config, err = defaultRedisVectorStoreConfig(ctx)
		if err != nil {
			return nil, err
		}
	}

	client := redis.NewClient(&redis.Options{
		Addr: config.RedisAddr,
	})

	return ri.NewIndexer(ctx, &ri.IndexerConfig{
		Client:    client,
		KeyPrefix: config.RedisKeyPrefix,
		DocumentToHashes: func(ctx context.Context, doc *schema.Document) (*ri.Hashes, error) {
			key := doc.ID
			if doc.ID == "" {
				key = uuid.New().String()
			}

			metadataBytes, err := json.Marshal(doc.MetaData)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal metadata: %w", err)
			}

			return &ri.Hashes{
				Key: key,
				Field2Value: map[string]ri.FieldValue{
					contentField:  {Value: doc.Content, EmbedKey: vectorField},
					metadataField: {Value: metadataBytes},
				},
			}, nil
		},
		BatchSize: 10,
		Embedding: config.Embedding,
	})
}

func NewRedisRetriever(ctx context.Context, config *RedisVectorStoreConfig) (r retriever.Retriever, err error) {
	if config == nil {
		config, err = defaultRedisVectorStoreConfig(ctx)
		if err != nil {
			return nil, err
		}
	}

	client := redis.NewClient(&redis.Options{
		Addr:          config.RedisAddr,
		Protocol:      2,
		UnstableResp3: true,
	})

	return rr.NewRetriever(ctx, &rr.RetrieverConfig{
		Client:            client,
		Index:             fmt.Sprintf("%s:%s", config.RedisKeyPrefix, indexNamePrefix),
		VectorField:       vectorField,
		DistanceThreshold: nil,
		Dialect:           2,
		ReturnFields: []string{
			contentField,
			vectorField,
		},
		DocumentConverter: func(ctx context.Context, doc redis.Document) (*schema.Document, error) {
			resp := &schema.Document{
				ID:       doc.ID,
				Content:  "",
				MetaData: map[string]any{},
			}
			for field, val := range doc.Fields {
				if field == contentField {
					resp.Content = val
				} else if field == vectorField {
					resp.WithDenseVector(rr.Bytes2Vector([]byte(val)))
				} else {
					resp.MetaData[field] = val
				}
			}
			return resp, nil
		},
		TopK:      5,
		Embedding: config.Embedding,
	})
}

type RedisVectorStoreConfig struct {
	RedisAddr      string             `json:"redis_addr"`
	Embedding      embedding.Embedder `json:"-"`
	RedisKeyPrefix string             `json:"redis_key_prefix"`
	Dimension      int                `json:"dimension"`
	TopK           int                `json:"top_k"`
	MinScore       float64            `json:"min_score"`
}

func defaultRedisVectorStoreConfig(ctx context.Context) (*RedisVectorStoreConfig, error) {
	config := &RedisVectorStoreConfig{}
	return config, nil
}

// RedisVectorStoreConfigImpl implements both indexer.Indexer and retriever.Retriever interfaces
type RedisVectorStoreConfigImpl struct {
	config *RedisVectorStoreConfig

	client *redis.Client
	prefix string
}

// NewRedisVectorStore creates a new Redis vector store
func NewRedisVectorStore(ctx context.Context, config *RedisVectorStoreConfig) (store *RedisVectorStoreConfigImpl, err error) {
	if config == nil {
		config, err = defaultRedisVectorStoreConfig(ctx)
		if err != nil {
			return nil, err
		}
	}
	impl := &RedisVectorStoreConfigImpl{config: config}

	err = impl.init(ctx)
	if err != nil {
		return nil, err
	}

	return impl, nil
}

func (impl *RedisVectorStoreConfigImpl) init(ctx context.Context) (err error) {
	if impl.config.Embedding == nil {
		return fmt.Errorf("embedding cannot be nil")
	}
	if impl.config.Dimension <= 0 {
		return fmt.Errorf("dimension must be positive")
	}
	if impl.config.TopK <= 0 {
		impl.config.TopK = topK
	}

	client := redis.NewClient(&redis.Options{
		Addr: impl.config.RedisAddr,
	})

	// 确保在错误时关闭连接
	defer func() {
		if err != nil {
			client.Close()
		}
	}()

	if err = client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	prefix := defaultRedisKeyPrefix
	if impl.config.RedisKeyPrefix != "" {
		prefix = impl.config.RedisKeyPrefix
	}

	impl.client = client
	impl.prefix = prefix

	indexName := fmt.Sprintf("%s:%s", prefix, indexNamePrefix)

	// 检查是否存在索引
	exists, err := client.Do(ctx, "FT.INFO", indexName).Result()
	if err != nil {
		if !strings.Contains(err.Error(), "Unknown index name") {
			return fmt.Errorf("failed to check if index exists: %w", err)
		}
		err = nil
	} else if exists != nil {
		return nil
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
		"DIM", impl.config.Dimension,
		"DISTANCE_METRIC", "COSINE",
	}

	if err = client.Do(ctx, createIndexArgs...).Err(); err != nil {
		return fmt.Errorf("failed to create index: %w", err)
	}

	// 验证索引是否创建成功
	if _, err = client.Do(ctx, "FT.INFO", indexName).Result(); err != nil {
		return fmt.Errorf("failed to verify index creation: %w", err)
	}

	return nil
}

// Store implements the indexer.Indexer interface
func (impl *RedisVectorStoreConfigImpl) Store(ctx context.Context, docs []*schema.Document, opts ...indexer.Option) ([]string, error) {
	if len(docs) == 0 {
		return []string{}, nil
	}

	ids := make([]string, len(docs))
	pipe := impl.client.Pipeline()

	for i, doc := range docs {
		if doc.Content == "" {
			continue
		}

		docID := uuid.New().String()
		ids[i] = docID
		vectors, err := impl.config.Embedding.EmbedStrings(ctx, []string{doc.Content})
		if err != nil {
			fmt.Println("failed to get embedding for document: %w", err)
			continue
			return nil, fmt.Errorf("failed to get embedding for document: %w", err)
		}
		vector := vectors[0]

		if len(vector) != impl.config.Dimension {
			return nil, fmt.Errorf("vector dimension mismatch: got %d, want %d", len(vector), impl.config.Dimension)
		}

		vectorBytes := float64ArrayToFloat32Bytes(vector)

		// Convert metadata to JSON string
		metadataBytes, err := json.Marshal(doc.MetaData)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal metadata: %w", err)
		}

		key := impl.prepareKey(docID)
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
func (impl *RedisVectorStoreConfigImpl) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]*schema.Document, error) {
	if query == "" {
		return []*schema.Document{}, nil
	}

	vectors, err := impl.config.Embedding.EmbedStrings(ctx, []string{query})
	if err != nil {
		return nil, fmt.Errorf("failed to get embedding for query: %w", err)
	}
	queryVector := vectors[0]

	if len(queryVector) != impl.config.Dimension {
		return nil, fmt.Errorf("query vector dimension mismatch: got %d, want %d", len(queryVector), impl.config.Dimension)
	}

	indexName := fmt.Sprintf("%s:%s", impl.prefix, indexNamePrefix)

	vectorBytes := float64ArrayToFloat32Bytes(queryVector)

	searchArgs := []interface{}{
		"FT.SEARCH", indexName,
		fmt.Sprintf("*=>[KNN %d @%s $BLOB AS distance]", impl.config.TopK, vectorField),
		"PARAMS", "2",
		"BLOB", vectorBytes,
		"RETURN", "4", contentField, metadataField, "distance", "id",
		"SORTBY", "distance",
		"DIALECT", "2",
	}

	results, err := impl.client.Do(ctx, searchArgs...).Result()
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
			id = strings.TrimPrefix(id, impl.prefix)
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

			if score < impl.config.MinScore {
				continue
			}

			doc.WithScore(score)

			docs = append(docs, doc)
		}
	}

	return docs, nil
}

func (impl *RedisVectorStoreConfigImpl) prepareKey(docID string) string {
	return impl.prefix + docID
}

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
