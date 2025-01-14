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

package index_graph

import (
	"context"

	"github.com/cloudwego/eino-ext/components/document/loader/file"
	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/markdown"
	"github.com/cloudwego/eino/components/document"
	"github.com/cloudwego/eino/compose"
)

type IndexGraphBuildConfig struct {
	LoadFileKeyOfLoader                   *file.FileLoaderConfig
	RedisVectorStoreKeyOfIndexer          *RedisVectorStoreConfig
	SplitDocumentKeyOfDocumentTransformer *markdown.HeaderConfig
}
type BuildConfig struct {
	IndexGraph *IndexGraphBuildConfig
}

func BuildIndexGraph(ctx context.Context, config BuildConfig) (r compose.Runnable[document.Source, []string], err error) {
	const (
		LoadFile         = "LoadFile"
		RedisVectorStore = "RedisVectorStore"
		SplitDocument    = "SplitDocument"
	)
	g := compose.NewGraph[document.Source, []string]()
	loadFileKeyOfLoader, err := NewLoadFileConfig(ctx, config.IndexGraph.LoadFileKeyOfLoader)
	if err != nil {
		return nil, err
	}
	_ = g.AddLoaderNode(LoadFile, loadFileKeyOfLoader)
	redisVectorStoreKeyOfIndexer, err := NewRedisVectorStore(ctx, config.IndexGraph.RedisVectorStoreKeyOfIndexer)
	if err != nil {
		return nil, err
	}
	_ = g.AddIndexerNode(RedisVectorStore, redisVectorStoreKeyOfIndexer)
	splitDocumentKeyOfDocumentTransformer, err := NewSplitDocument(ctx, config.IndexGraph.SplitDocumentKeyOfDocumentTransformer)
	if err != nil {
		return nil, err
	}
	_ = g.AddDocumentTransformerNode(SplitDocument, splitDocumentKeyOfDocumentTransformer)
	_ = g.AddEdge(compose.START, LoadFile)
	_ = g.AddEdge(RedisVectorStore, compose.END)
	_ = g.AddEdge(LoadFile, SplitDocument)
	_ = g.AddEdge(SplitDocument, RedisVectorStore)
	r, err = g.Compile(ctx, compose.WithGraphName("IndexGraph"), compose.WithNodeTriggerMode(compose.AllPredecessor))
	if err != nil {
		return nil, err
	}
	return r, nil
}
