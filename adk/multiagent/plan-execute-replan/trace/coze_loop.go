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

package trace

import (
	"context"
	"log"
	"os"

	ccb "github.com/cloudwego/eino-ext/callbacks/cozeloop"
	"github.com/cloudwego/eino/callbacks"
	"github.com/coze-dev/cozeloop-go"
)

func AppendCozeLoopCallbackIfConfigured(ctx context.Context) (closeFn func(ctx context.Context), client cozeloop.Client) {
	// setup cozeloop
	// COZELOOP_WORKSPACE_ID=your workspace id
	// COZELOOP_API_TOKEN=your token

	wsID := os.Getenv("COZELOOP_WORKSPACE_ID")
	apiKey := os.Getenv("COZELOOP_API_TOKEN")
	if wsID == "" || apiKey == "" {
		return func(ctx context.Context) {
			return
		}, nil
	}
	client, err := cozeloop.NewClient(
		cozeloop.WithWorkspaceID(wsID),
		cozeloop.WithAPIToken(apiKey),
	)
	if err != nil {
		log.Fatalf("cozeloop.NewClient failed, err: %v", err)
	}

	// init once
	handler := ccb.NewLoopHandler(client)
	callbacks.AppendGlobalHandlers(handler)

	return client.Close, client
}

func StartRootSpan(client cozeloop.Client, ctx context.Context, input any) (
	nCtx context.Context, finishFn func(ctx context.Context, output any)) {

	if client == nil {
		return ctx, func(ctx context.Context, output any) {
			return
		}
	}

	nCtx, span := client.StartSpan(ctx, "plan-execute-replan", "custom")
	span.SetInput(ctx, input)

	return nCtx, func(ctx context.Context, output any) {
		span.SetOutput(ctx, output)
		span.Finish(ctx)
	}
}
