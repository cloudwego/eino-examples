package agents

import (
	"context"

	"github.com/cloudwego/eino/adk"
)

type GenPDF struct{}

func (g *GenPDF) Name(ctx context.Context) string {
	return "GenPDF"
}

func (g *GenPDF) Description(ctx context.Context) string {
	return "this agent will generate a pdf file from input"
}

func (g *GenPDF) Run(ctx context.Context, input *adk.AgentInput, options ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	return nil
}

type Searcher struct{}

func (s *Searcher) Name(ctx context.Context) string {
	return "Searcher"
}

func (s *Searcher) Description(ctx context.Context) string {
	return "this agent will search content by input"
}

func (s *Searcher) Run(ctx context.Context, input *adk.AgentInput, options ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	return nil
}
