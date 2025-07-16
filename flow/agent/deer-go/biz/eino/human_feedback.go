/*
 * Copyright 2025 CloudWeGo Authors
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

package eino

import (
	"context"

	"github.com/cloudwego/eino/compose"

	"github.com/cloudwego/eino-examples/flow/agent/deer-go/biz/consts"
	"github.com/cloudwego/eino-examples/flow/agent/deer-go/biz/model"
)

func routerHuman(ctx context.Context, input string, opts ...any) (output string, err error) {
	err = compose.ProcessState[*model.State](ctx, func(_ context.Context, state *model.State) error {
		defer func() {
			output = state.Goto
			state.InterruptFeedback = ""
		}()
		state.Goto = consts.ResearchTeam
		if !state.AutoAcceptedPlan {
			switch state.InterruptFeedback {
			case consts.AcceptPlan:
				return nil
			case consts.EditPlan:
				state.Goto = consts.Planner
				return nil
			default:
				return compose.InterruptAndRerun
			}
		}
		state.Goto = consts.ResearchTeam
		return nil
	})
	return output, err
}

func NewHumanNode[I, O any](ctx context.Context) *compose.Graph[I, O] {
	cag := compose.NewGraph[I, O]()
	_ = cag.AddLambdaNode("router", compose.InvokableLambdaWithOption(routerHuman))

	_ = cag.AddEdge(compose.START, "router")
	_ = cag.AddEdge("router", compose.END)

	return cag
}
