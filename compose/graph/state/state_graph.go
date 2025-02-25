/*
 * Copyright 2024 CloudWeGo Authors
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
	"errors"
	"io"
	"runtime/debug"
	"strings"
	"unicode/utf8"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/internal/logs"
)

func main() {
	ctx := context.Background()

	const (
		nodeOfL1 = "invokable"
		nodeOfL2 = "streamable"
		nodeOfL3 = "transformable"
	)

	type testState struct {
		ms []string
	}

	gen := func(ctx context.Context) *testState {
		return &testState{}
	}

	sg := compose.NewGraph[string, string](compose.WithGenLocalState(gen))

	l1 := compose.InvokableLambda(func(ctx context.Context, in string) (out string, err error) {
		return "InvokableLambda: " + in, nil
	})

	l1StateToInput := func(ctx context.Context, in string, state *testState) (string, error) {
		state.ms = append(state.ms, in)
		return in, nil
	}

	l1StateToOutput := func(ctx context.Context, out string, state *testState) (string, error) {
		state.ms = append(state.ms, out)
		return out, nil
	}

	_ = sg.AddLambdaNode(nodeOfL1, l1,
		compose.WithStatePreHandler(l1StateToInput), compose.WithStatePostHandler(l1StateToOutput))

	l2 := compose.StreamableLambda(func(ctx context.Context, input string) (output *schema.StreamReader[string], err error) {
		outStr := "StreamableLambda: " + input

		sr, sw := schema.Pipe[string](utf8.RuneCountInString(outStr))

		// nolint: byted_goroutine_recover
		go func() {
			for _, field := range strings.Fields(outStr) {
				sw.Send(field+" ", nil)
			}
			sw.Close()
		}()

		return sr, nil
	})

	l2StateToOutput := func(ctx context.Context, out string, state *testState) (string, error) {
		state.ms = append(state.ms, out)
		return out, nil
	}

	_ = sg.AddLambdaNode(nodeOfL2, l2, compose.WithStatePostHandler(l2StateToOutput))

	l3 := compose.TransformableLambda(func(ctx context.Context, input *schema.StreamReader[string]) (
		output *schema.StreamReader[string], err error) {

		prefix := "TransformableLambda: "
		sr, sw := schema.Pipe[string](20)

		go func() {

			defer func() {
				if err := recover(); err != nil {
					logs.Errorf("panic occurs: %v\nStack Trace:\n%s", err, string(debug.Stack()))
				}
			}()

			for _, field := range strings.Fields(prefix) {
				sw.Send(field+" ", nil)
			}

			for {
				chunk, err := input.Recv()
				if err != nil {
					if err == io.EOF {
						break
					}
					// TODO: how to trace this kind of error in the goroutine of processing sw
					sw.Send(chunk, err)
					break
				}

				sw.Send(chunk, nil)

			}
			sw.Close()
		}()

		return sr, nil
	})

	l3StateToOutput := func(ctx context.Context, out string, state *testState) (string, error) {
		state.ms = append(state.ms, out)
		logs.Infof("state result: ")
		for idx, m := range state.ms {
			logs.Infof("    %vth: %v", idx, m)
		}
		return out, nil
	}

	_ = sg.AddLambdaNode(nodeOfL3, l3, compose.WithStatePostHandler(l3StateToOutput))

	_ = sg.AddEdge(compose.START, nodeOfL1)

	_ = sg.AddEdge(nodeOfL1, nodeOfL2)

	_ = sg.AddEdge(nodeOfL2, nodeOfL3)

	_ = sg.AddEdge(nodeOfL3, compose.END)

	run, err := sg.Compile(ctx)
	if err != nil {
		logs.Errorf("sg.Compile failed, err=%v", err)
		return
	}

	out, err := run.Invoke(ctx, "how are you")
	if err != nil {
		logs.Errorf("run.Invoke failed, err=%v", err)
		return
	}
	logs.Infof("invoke result: %v", out)

	stream, err := run.Stream(ctx, "how are you")
	if err != nil {
		logs.Errorf("run.Stream failed, err=%v", err)
		return
	}

	for {

		chunk, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			logs.Infof("stream.Recv() failed, err=%v", err)
			break
		}

		logs.Tokenf("%v", chunk)
	}
	stream.Close()

	sr, sw := schema.Pipe[string](1)
	sw.Send("how are you", nil)
	sw.Close()

	stream, err = run.Transform(ctx, sr)
	if err != nil {
		logs.Infof("run.Transform failed, err=%v", err)
		return
	}

	for {

		chunk, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			logs.Infof("stream.Recv() failed, err=%v", err)
			break
		}

		logs.Infof("%v", chunk)
	}
	stream.Close()
}
