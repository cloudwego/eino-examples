package infra

import (
	"context"
	"fmt"
	"os"

	"github.com/RanFeng/ilog"
	"github.com/cloudwego/eino-ext/callbacks/apmplus"
	"github.com/cloudwego/eino/callbacks"
)

func InitAPMPlusCallback(ctx context.Context) (callbacks.Handler, func(ctx context.Context) error) {
	if os.Getenv("APMPLUS_APP_KEY") == "" {
		return nil, nil
	}
	region := os.Getenv("APMPLUS_REGION")
	if region == "" {
		region = "cn-beijing"
	}
	ilog.EventInfo(ctx, "Init APMPlus callback, watch at: https://console.volcengine.com/apmplus-server", region)

	cbh, shutdown, err := apmplus.NewApmplusHandler(&apmplus.Config{
		Host:        fmt.Sprintf("apmplus-%s.volces.com:4317", region),
		AppKey:      os.Getenv("APMPLUS_APP_KEY"),
		ServiceName: "deer-go",
		Release:     "release/v0.0.1",
	})
	if err != nil {
		ilog.EventError(ctx, err, "init apmplus callback failed")
		return nil, nil
	}
	callbacks.AppendGlobalHandlers(cbh)
	ilog.EventInfo(ctx, "Init APMPlus Callback success")
	return cbh, shutdown
}
