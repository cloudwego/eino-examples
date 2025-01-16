package knowledgeindexing

import (
	"context"

	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/markdown"
	"github.com/cloudwego/eino/components/document"
)

func defaultMarkdownSplitterConfig(ctx context.Context) (*markdown.HeaderConfig, error) {
	config := &markdown.HeaderConfig{
		TrimHeaders: true}
	return config, nil
}

func NewMarkdownSplitter(ctx context.Context, config *markdown.HeaderConfig) (tfr document.Transformer, err error) {
	if config == nil {
		config, err = defaultMarkdownSplitterConfig(ctx)
		if err != nil {
			return nil, err
		}
	}
	tfr, err = markdown.NewHeaderSplitter(ctx, config)
	if err != nil {
		return nil, err
	}
	return tfr, nil
}
