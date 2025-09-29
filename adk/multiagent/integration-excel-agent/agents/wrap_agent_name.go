package agents

import (
	"context"

	"github.com/cloudwego/eino/adk"
)

func WrapAgentWithName(agent adk.Agent, name string) adk.Agent {
	return &wrapWithName{agent, name}
}

type wrapWithName struct {
	adk.Agent
	name string
}

func (w *wrapWithName) Name(ctx context.Context) string {
	return w.name
}
