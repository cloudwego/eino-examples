package index_graph

import (
	"context"

	"github.com/cloudwego/eino-ext/components/document/loader/file"
	"github.com/cloudwego/eino/components/document"
)

func defaultLoadFileConfigConfig(ctx context.Context) (*file.FileLoaderConfig, error) {
	config := &file.FileLoaderConfig{}
	return config, nil
}

func NewLoadFileConfig(ctx context.Context, config *file.FileLoaderConfig) (ld document.Loader, err error) {
	if config == nil {
		config, err = defaultLoadFileConfigConfig(ctx)
		if err != nil {
			return nil, err
		}
	}
	ld, err = file.NewFileLoader(ctx, config)
	if err != nil {
		return nil, err
	}
	return ld, nil
}
