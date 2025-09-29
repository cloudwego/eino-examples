package agents

import (
	"context"
	"encoding/json"
	"log"
	"runtime/debug"

	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/consts"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/generic"
	"github.com/cloudwego/eino-examples/adk/multiagent/integration-excel-agent/params"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/adk/prebuilt/planexecute"
)

func NewWrite2PlanMDWrapper(a adk.Agent) adk.Agent {
	return &write2PlanMDWrapper{a: a}
}

type write2PlanMDWrapper struct {
	a adk.Agent
}

func (r *write2PlanMDWrapper) Name(ctx context.Context) string {
	return r.a.Name(ctx)
}

func (r *write2PlanMDWrapper) Description(ctx context.Context) string {
	return r.a.Description(ctx)
}

func (r *write2PlanMDWrapper) Run(ctx context.Context, input *adk.AgentInput, options ...adk.AgentRunOption) *adk.AsyncIterator[*adk.AgentEvent] {
	iter := r.a.Run(ctx, input, options...)
	nIter, gen := adk.NewAsyncIteratorPair[*adk.AgentEvent]()

	go func() {
		defer func() {
			if e := recover(); e != nil {
				log.Printf("[write2PlanMDWrapper] exec panic recover:%+v, stack: %s", e, string(debug.Stack()))
			}
			gen.Close()
		}()

		for {
			e, ok := iter.Next()
			if !ok {
				break
			}
			if e.Action != nil && e.Action.Exit {
				err := write2PlanMD(ctx)
				gen.Send(e)
				if err != nil {
					log.Print("write plan failed", err)
					return
				}
				return
			}
			gen.Send(e)
		}

		err := write2PlanMD(ctx)
		if err != nil {
			log.Print("write plan failed", err)
			return
		}
	}()

	return nIter
}

func write2PlanMD(ctx context.Context) error {
	var executedSteps []planexecute.ExecutedStep
	var plan *Plan
	p, ok := consts.GetSessionValue[*Plan](ctx, planexecute.PlanSessionKey)
	if ok {
		plan = p
	}
	es, ok := consts.GetSessionValue[[]planexecute.ExecutedStep](ctx, planexecute.ExecutedStepsSessionKey)
	if ok {
		executedSteps = es
	}

	var plans []*generic.FullPlan
	for i, step := range executedSteps {
		var desc string
		s := &Step{}
		err := json.Unmarshal([]byte(step.Step), s)
		if err == nil {
			desc = s.Desc
		}
		plans = append(plans, &generic.FullPlan{
			TaskID: i + 1,
			Status: generic.PlanStatusDone,
			Desc:   desc,
			ExecResult: &generic.SubmitResult{
				IsSuccess: nil,
				Result:    step.Result,
				Files:     nil, // todo
			},
			EvaluatorExecResult: nil, // todo
		})
	}
	if plan != nil {
		for i, step := range plan.Steps {
			plans = append(plans, &generic.FullPlan{
				TaskID: i + len(executedSteps) + 1,
				Status: generic.PlanStatusTodo,
				Desc:   step.Desc,
			})
		}
	}
	err := generic.Write2PlanMD(params.GetCozeSpaceTaskID(ctx), plans)
	if err != nil {
		return err
	}

	return nil
}
