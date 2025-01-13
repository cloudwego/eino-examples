package index_graph

import (
	"context"

	"github.com/cloudwego/eino-ext/components/document/transformer/splitter/markdown"
	"github.com/cloudwego/eino/components/document"
)

func defaultSplitDocumentConfig(ctx context.Context) (*markdown.HeaderConfig, error) {
	config := &markdown.HeaderConfig{
		Headers: map[string]string{
			"#":   "title",
			"##":  "subtitle",
			"###": "section",
		},
	}
	return config, nil
}

func NewSplitDocument(ctx context.Context, config *markdown.HeaderConfig) (tf document.Transformer, err error) {
	if config == nil {
		config, err = defaultSplitDocumentConfig(ctx)
		if err != nil {
			return nil, err
		}
	}
	tf, err = markdown.NewHeaderSplitter(ctx, config)
	if err != nil {
		return nil, err
	}
	return tf, nil
}
