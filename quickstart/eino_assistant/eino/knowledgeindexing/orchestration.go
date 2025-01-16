package knowledgeindexing

import (
	"context"

	"github.com/cloudwego/eino-ext/components/document/loader/file"
	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/markdown"
	"github.com/cloudwego/eino-ext/components/indexer/redis"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/compose"
)

type KnowledgeIndexingBuildConfig struct {
	FileLoaderKeyOfLoader                    *file.FileLoaderConfig
	MarkdownSplitterKeyOfDocumentTransformer *markdown.HeaderConfig
	RedisIndexerKeyOfIndexer                 *redis.IndexerConfig
}

type BuildConfig struct {
	KnowledgeIndexing *KnowledgeIndexingBuildConfig
}

func BuildKnowledgeIndexing(ctx context.Context, config *BuildConfig) (r compose.Runnable[document.Source, []string], err error) {
	const (
		FileLoader       = "FileLoader"
		MarkdownSplitter = "MarkdownSplitter"
		RedisIndexer     = "RedisIndexer"
	)
	g := compose.NewGraph[document.Source, []string]()
	fileLoaderKeyOfLoader, err := NewFileLoader(ctx, config.KnowledgeIndexing.FileLoaderKeyOfLoader)
	if err != nil {
		return nil, err
	}
	_ = g.AddLoaderNode(FileLoader, fileLoaderKeyOfLoader)
	markdownSplitterKeyOfDocumentTransformer, err := NewMarkdownSplitter(ctx, config.KnowledgeIndexing.MarkdownSplitterKeyOfDocumentTransformer)
	if err != nil {
		return nil, err
	}
	_ = g.AddDocumentTransformerNode(MarkdownSplitter, markdownSplitterKeyOfDocumentTransformer)
	redisIndexerKeyOfIndexer, err := NewRedisIndexer(ctx, config.KnowledgeIndexing.RedisIndexerKeyOfIndexer)
	if err != nil {
		return nil, err
	}
	_ = g.AddIndexerNode(RedisIndexer, redisIndexerKeyOfIndexer)
	_ = g.AddEdge(compose.START, FileLoader)
	_ = g.AddEdge(RedisIndexer, compose.END)
	_ = g.AddEdge(FileLoader, MarkdownSplitter)
	_ = g.AddEdge(MarkdownSplitter, RedisIndexer)
	r, err = g.Compile(ctx, compose.WithGraphName("KnowledgeIndexing"), compose.WithNodeTriggerMode(compose.AnyPredecessor))
	if err != nil {
		return nil, err
	}
	return r, err
}
