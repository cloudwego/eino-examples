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

package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"code.byted.org/flow/eino/components/embedding"
	"github.com/cloudwego/eino-examples/quickstart/eino_assistant/eino/knowledgeindexing"
	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/markdown"
	"github.com/cloudwego/eino-ext/components/embedding/ark"
	indexRedis "github.com/cloudwego/eino-ext/components/indexer/redis"
	"github.com/cloudwego/eino-ext/devops"
	"github.com/cloudwego/eino/components/document"
	"github.com/redis/go-redis/v9"
)

func init() {
	if os.Getenv("EINO_DEBUG") == "true" {
		err := devops.Init(context.Background())
		if err != nil {
			log.Printf("[eino dev] init failed, err=%v", err)
		}
	}
}

func main() {
	ctx := context.Background()

	err := indexMarkdownFiles(ctx, "./eino-docs")
	if err != nil {
		panic(err)
	}

	fmt.Println("index success")
}

func indexMarkdownFiles(ctx context.Context, dir string) error {
	// 初始化 embedding, 使用 ark 的 embedding 服务
	// 请查看 README.md 获取相关信息
	embedding, err := ark.NewEmbedder(ctx, &ark.EmbeddingConfig{
		Model:  os.Getenv("ARK_EMBEDDING_MODEL"),
		APIKey: os.Getenv("ARK_API_KEY"),
	})
	if err != nil {
		return fmt.Errorf("new embedder failed: %w", err)
	}

	redisCli := redis.NewClient(&redis.Options{
		Addr: "127.0.0.1:6379",
	})
	runner, err := knowledgeindexing.BuildKnowledgeIndexing(ctx, &knowledgeindexing.BuildConfig{
		KnowledgeIndexing: &knowledgeindexing.KnowledgeIndexingBuildConfig{
			RedisIndexerKeyOfIndexer: &indexRedis.IndexerConfig{
				Client:    redisCli,
				KeyPrefix: "eino:doc:",
				Embedding: embedding,
			},
			MarkdownSplitterKeyOfDocumentTransformer: &markdown.HeaderConfig{
				Headers: map[string]string{
					"#": "title",
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("build index graph failed: %w", err)
	}

	// 遍历 dir 下的所有 markdown 文件
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk dir failed: %w", err)
		}
		if d.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".md") {
			fmt.Printf("[skip] not a markdown file: %s\n", path)
			return nil
		}

		fmt.Printf("[start] indexing file: %s\n", path)

		ids, err := runner.Invoke(ctx, document.Source{URI: path})
		if err != nil {
			return fmt.Errorf("invoke index graph failed: %w", err)
		}

		fmt.Printf("[done] indexing file: %s, len of parts: %d\n", path, len(ids))

		return nil
	})

	return err
}

type RedisVectorStoreConfig struct {
	RedisKeyPrefix string
	IndexName      string
	Embedding      embedding.Embedder
	Dimension      int
	MinScore       float64
	RedisAddr      string
}

func initVectorIndex(ctx context.Context, config *RedisVectorStoreConfig) (err error) {
	if config.Embedding == nil {
		return fmt.Errorf("embedding cannot be nil")
	}
	if config.Dimension <= 0 {
		return fmt.Errorf("dimension must be positive")
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
		return fmt.Errorf("failed to connect to Redis: %w", err)
	}

	indexName := fmt.Sprintf("%s:%s", config.RedisKeyPrefix, config.IndexName)

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
		"PREFIX", "1", config.RedisKeyPrefix,
		"SCHEMA",
		"content", "TEXT",
		"metadata", "TEXT",
		"vector", "VECTOR", "FLAT",
		"6",
		"TYPE", "FLOAT32",
		"DIM", config.Dimension,
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