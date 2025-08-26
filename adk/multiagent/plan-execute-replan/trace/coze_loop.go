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
