package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino-examples/agent/redis"
	index_graph "github.com/cloudwego/eino-examples/index_graphx"
	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/markdown"
	"github.com/cloudwego/eino-ext/components/embedding/ark"
	"github.com/cloudwego/eino-ext/devops"
	"github.com/cloudwego/eino/components/document"
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
	dimensions := 4096
	embedding, err := ark.NewEmbedder(ctx, &ark.EmbeddingConfig{
		Model:      os.Getenv("ARK_EMBEDDING_MODEL"),
		APIKey:     os.Getenv("ARK_API_KEY"),
		Dimensions: &dimensions,
	})
	if err != nil {
		return fmt.Errorf("new embedder failed: %w", err)
	}
	runner, err := index_graph.BuildIndexGraph(ctx, index_graph.BuildConfig{
		IndexGraph: &index_graph.IndexGraphBuildConfig{
			RedisVectorStoreKeyOfIndexer: &index_graph.RedisVectorStoreConfig{
				RedisVectorStoreConfig: redis.RedisVectorStoreConfig{
					RedisAddr:      "127.0.0.1:6379",
					RedisKeyPrefix: "eino:doc:",
					Dimension:      4096,
					Embedding:      embedding,
				},
			},
			SplitDocumentKeyOfDocumentTransformer: &markdown.HeaderConfig{
				Headers: map[string]string{
					"#":  "title",
					"##": "subtitle",
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
