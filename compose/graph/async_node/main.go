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
	"errors"
	"io"
	"strings"

	"github.com/cloudwego/eino/compose"
	"github.com/cloudwego/eino/schema"

	"github.com/cloudwego/eino-examples/internal/logs"
)

// This example demonstrates an async lambda node inside an Eino graph.
// Scenario 1: Background report generation (invokable).
// Scenario 2: Live transcription stream (streamable) with conversion.
func main() {
	ctx := context.Background()

	g := compose.NewGraph[string, string]()

	// async_invokable: starts a long-running job and waits on a channel.
	inv := compose.InvokableLambda(func(ctx context.Context, in string) (string, error) {
		notify := make(chan reportResult, 1)
		logs.Infof("async_invokable: start report job with input=%q", in)
		go generateReport(ctx, in, notify)

		logs.Infof("async_invokable: waiting on reportResult notification")
		select {
		case r := <-notify:
			return r.url, r.err
		case <-ctx.Done():
			return "", ctx.Err()
		}
	})

	// async_streamable: consumes a live stream and converts tokens.
	str := compose.StreamableLambda(func(ctx context.Context, in string) (*schema.StreamReader[string], error) {
		logs.Infof("async_streamable: start transcription stream with input=%q", in)
		upstream := transcribeLive(ctx, in)
		converted := schema.StreamReaderWithConvert(upstream, func(tok string) (string, error) {
			if tok == "" {
				return tok, errors.New("empty token")
			}
			return strings.ToUpper(tok), nil
		})
		logs.Infof("async_streamable: waiting on upstream tokens via StreamReaderWithConvert")
		return converted, nil
	})

	_ = g.AddLambdaNode("async_invokable", inv)
	_ = g.AddLambdaNode("async_streamable", str)

	_ = g.AddEdge(compose.START, "async_invokable")
	_ = g.AddEdge("async_invokable", "async_streamable")
	_ = g.AddEdge("async_streamable", compose.END)

	run, err := g.Compile(ctx)
	if err != nil {
		logs.Errorf("compile error: %v", err)
		return
	}

	// Invoke: "generate report" path
	out, err := run.Invoke(ctx, "Quarterly Sales Report")
	if err != nil {
		logs.Errorf("invoke error: %v", err)
	} else {
		logs.Infof("report url: %s", out)
	}

	// Stream: "transcription" path
	stream, err := run.Stream(ctx, "hello world from async node")
	if err != nil {
		logs.Errorf("stream error: %v", err)
		return
	}

	for {
		chunk, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			logs.Tokenf("recv error: %v", err)
			break
		}
		logs.Tokenf("%s", chunk)
	}
	stream.Close()
}
