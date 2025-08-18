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

package main

import (
	"context"

	"github.com/cloudwego/eino/compose"

	"github.com/cloudwego/eino-examples/internal/logs"
)

func main() {
	type calculator struct {
		Add      []int
		Multiply int
	}

	adder := func(ctx context.Context, in []int) (out int, err error) {
		for _, i := range in {
			out += i
		}
		return out, nil
	}

	type mul struct {
		A int
		B int
	}

	multiplier := func(ctx context.Context, m mul) (int, error) {
		return m.A * m.B, nil
	}

	wf := compose.NewWorkflow[calculator, int]()

	wf.AddLambdaNode("adder", compose.InvokableLambda(adder)).
		AddInput(compose.START, compose.FromField("Add"))

	wf.AddLambdaNode("mul", compose.InvokableLambda(multiplier)).
		AddInput("adder", compose.ToField("A")).
		AddInputWithOptions(compose.START, []*compose.FieldMapping{compose.MapFields("Multiply", "B")},
			// use WithNoDirectDependency to declare a 'data-only' dependency,
			// in this case, START node's execution status will not determine whether 'mul' node can execute.
			// START node only passes one field of its output to 'mul' node.
			compose.WithNoDirectDependency())

	wf.End().AddInput("mul")

	runner, err := wf.Compile(context.Background())
	if err != nil {
		logs.Errorf("workflow compile error: %v", err)
		return
	}

	result, err := runner.Invoke(context.Background(), calculator{
		Add:      []int{2, 5},
		Multiply: 3,
	})
	if err != nil {
		logs.Errorf("workflow run err: %v", err)
		return
	}

	logs.Infof("%d", result)
}
