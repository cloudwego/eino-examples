package eino_graph

import (
	"context"

	"github.com/cloudwego/eino/compose"
)

type GxBuildConfig struct {
	XxxKeyOfChatModel *ConfConfig
}

type BuildConfig struct {
	Gx *GxBuildConfig
}

func BuildGx(ctx context.Context, config BuildConfig) (*compose.Graph[any, any], error) {
	var err error
	const Xxx = "Xxx"
	g := compose.NewGraph[any, any]()
	xxxKeyOfChatModel, err := NewConf(ctx, config.Gx.XxxKeyOfChatModel)
	if err != nil {
		return nil, err
	}
	_ = g.AddChatModelNode(Xxx, xxxKeyOfChatModel)
	_ = g.AddEdge(compose.START, Xxx)
	_ = g.AddEdge(Xxx, compose.END)
	_, err = g.Compile(ctx, compose.WithGraphName("Gx"), compose.WithNodeTriggerMode(compose.AllPredecessor))
	if err != nil {
		return nil, err
	}
	return g, err
}
