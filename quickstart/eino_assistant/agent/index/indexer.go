package index

// import (
// 	"context"
// 	"fmt"
// 	"io/fs"
// 	"path/filepath"
// 	"strings"

// 	"github.com/cloudwego/eino-examples/agent/redis"
// 	"github.com/cloudwego/eino-examples/index_graph"
// 	"github.com/cloudwego/eino-ext/components/document/loader/file"
// 	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/markdown"
// 	"github.com/cloudwego/eino/components/document"
// )

// // index markdown files in a directory
// // 1. 使用 document.Loader 来读取文件
// // 2. 使用 transformers 来处理 markdown 文件 (按照 #, ##, ### 分割成块)
// // 3. 使用 indexer 来保存 markdown 文件
// func NewMarkdownIndexer(ctx context.Context, indexer indexer.Indexer) (*compose.Graph[document.Source, []string], error) {
// 	if indexer == nil {
// 		return nil, fmt.Errorf("indexer is required")
// 	}

// 	graph := compose.NewGraph[document.Source, []string]()

// 	loader, err := file.NewFileLoader(ctx, &file.FileLoaderConfig{})
// 	if err != nil {
// 		return nil, err
// 	}

// 	graph.AddLoaderNode("load_file", loader)

// 	transformer, err := markdown.NewHeaderSplitter(ctx, &markdown.HeaderConfig{
// 		Headers: map[string]string{
// 			"#":   "Header1",
// 			"##":  "Header2",
// 			"###": "Header3",
// 		},
// 	})
// 	if err != nil {
// 		return nil, err
// 	}

// 	graph.AddDocumentTransformerNode("transform_file", transformer)

// 	graph.AddIndexerNode("index_file", indexer)

// 	graph.AddEdge(compose.START, "load_file")
// 	graph.AddEdge("load_file", "transform_file")
// 	graph.AddEdge("transform_file", "index_file")

// 	err = graph.AddEdge("index_file", compose.END)
// 	if err != nil {
// 		return nil, err
// 	}

// 	return graph, nil
// }

// // IndexMarkdownFiles 索引 dir 目录下的所有 markdown 文件
// func IndexMarkdownFiles(ctx context.Context, dir string, redisVectorConfig redis.RedisVectorStoreConfig) error {
// 	graph, err := index_graph.BuildIndexGraph(ctx, index_graph.BuildConfig{
// 		IndexGraph: &index_graph.IndexGraphBuildConfig{
// 			LoadFileKeyOfLoader: &file.FileLoaderConfig{},
// 			RedisVectorStoreKeyOfIndexer: &index_graph.RedisVectorStoreConfig{
// 				RedisVectorStoreConfig: redisVectorConfig,
// 			},
// 			SplitDocumentKeyOfDocumentTransformer: &markdown.HeaderConfig{
// 				Headers: map[string]string{
// 					"#":   "title",
// 					"##":  "subtitle",
// 					"###": "section",
// 				},
// 			},
// 		},
// 	})
// 	if err != nil {
// 		return fmt.Errorf("build index graph failed: %w", err)
// 	}

// 	runner, err := graph.Compile(ctx)
// 	if err != nil {
// 		return fmt.Errorf("compile index graph failed: %w", err)
// 	}

// 	// 遍历 dir 下的所有 markdown 文件
// 	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
// 		if err != nil {
// 			return fmt.Errorf("walk dir failed: %w", err)
// 		}
// 		if d.IsDir() {
// 			return nil
// 		}

// 		if !strings.HasSuffix(path, ".md") {
// 			fmt.Printf("[skip] not a markdown file: %s\n", path)
// 			return nil
// 		}

// 		fmt.Printf("[start] indexing file: %s\n", path)

// 		ids, err := runner.Invoke(ctx, document.Source{URI: path})
// 		if err != nil {
// 			return fmt.Errorf("invoke index graph failed: %w", err)
// 		}

// 		fmt.Printf("[done] indexing file: %s, len of parts: %d\n", path, len(ids))

// 		return nil
// 	})

// 	return err
// }
