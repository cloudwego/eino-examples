package adk

import (
	"context"

	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt"
)

func NewPnDAgent(ctx context.Context) (adk.Agent, error) {
	planner, err := prebuilt.NewPlanner(ctx, &prebuilt.PlannerConfig{
		ChatModelWithFormattedOutput: nil,
		GenInputFn:                   plannerInputGen,
		NewPlan: func(ctx context.Context) prebuilt.Plan {
			return &Plan{}
		},
	})
	if err != nil {
		return nil, err
	}

	replanner, err := prebuilt.NewReplanner(ctx, &prebuilt.ReplannerConfig{
		ChatModel:   nil,
		PlanTool:    nil,
		RespondTool: nil,
		GenInputFn:  nil,
		NewPlan: func(ctx context.Context) prebuilt.Plan {
			return &Plan{}
		},
	})

	return prebuilt.NewPlanExecuteAgent(ctx, &prebuilt.PlanExecuteConfig{
		Planner:       planner,
		Executor:      nil,
		Replanner:     replanner,
		MaxIterations: 0,
	})
}
