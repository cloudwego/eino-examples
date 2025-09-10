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
	"fmt"
	"log"
	"time"

	"github.com/cloudwego/eino/adk"

	"github.com/cloudwego/eino-examples/adk/common/prints"
	"github.com/cloudwego/eino-examples/adk/common/trace"
)

func main() {
	ctx := context.Background()

	traceCloseFn, startSpanFn := trace.AppendCozeLoopCallbackIfConfigured(ctx)
	defer traceCloseFn(ctx)

	sv, err := buildSupervisor(ctx)
	if err != nil {
		log.Fatalf("build supervisor failed: %v", err)
	}

	query := "find US and New York state GDP in 2024. what % of US GDP was New York state?"

	runner := adk.NewRunner(ctx, adk.RunnerConfig{
		Agent:           sv,
		EnableStreaming: true,
	})

	ctx, endSpanFn := startSpanFn(ctx, "Supervisor", query)

	iter := runner.Query(ctx, query)

	fmt.Println("\nuser query: ", query)

	var lastMessage adk.Message
	for {
		event, hasEvent := iter.Next()
		if !hasEvent {
			break
		}

		prints.Event(event)

		if event.Output != nil {
			lastMessage, _, err = adk.GetMessage(event)
		}
	}

	endSpanFn(ctx, lastMessage)

	// wait for all span to be ended
	time.Sleep(5 * time.Second)
}
