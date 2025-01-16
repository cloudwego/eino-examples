package knowledgeindexing

import (
	"context"

	"github.com/cloudwego/eino-ext/components/document/loader/file"
	"github.com/cloudwego/eino/components/document"
)

func defaultFileLoaderConfig(ctx context.Context) (*file.FileLoaderConfig, error) {
	config := &file.FileLoaderConfig{}
	return config, nil
}

func NewFileLoader(ctx context.Context, config *file.FileLoaderConfig) (ldr document.Loader, err error) {
	if config == nil {
		config, err = defaultFileLoaderConfig(ctx)
		if err != nil {
			return nil, err
		}
	}
	ldr, err = file.NewFileLoader(ctx, config)
	if err != nil {
		return nil, err
	}
	return ldr, nil
}
