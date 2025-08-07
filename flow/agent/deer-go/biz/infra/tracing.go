package infra

import (
	"context"
	"fmt"
	"os"

	"github.com/RanFeng/ilog"
	"github.com/cloudwego/eino-ext/callbacks/apmplus"
	"github.com/cloudwego/eino/callbacks"
	hertzconfig "github.com/cloudwego/hertz/pkg/common/config"
	"github.com/hertz-contrib/obs-opentelemetry/provider"
	hertztracing "github.com/hertz-contrib/obs-opentelemetry/tracing"
)

var EmptyHertzConfigOption = hertzconfig.Option{}

// InitAPMPlusTracing initializes APMPlus observability components for Eino applications
// Creates Eino APMPlus callbacks and optionally initializes Hertz server tracing integration based on the withHertzServer flag.
// Parameters:
//
//	withHertzServer - If true, enables Hertz HTTP server tracing integration
//
// Returns:
//
//	tracer - Hertz server configuration (nil if withHertzServer=false)
//	cfg - Tracing configuration instance (nil if withHertzServer=false)
//	shutdown - cleanup function for eino apmplus callback to flush telemetry data to APMPlus Server
func InitAPMPlusTracing(ctx context.Context, withHertzServer bool) (tracer hertzconfig.Option, cfg *hertztracing.Config, shutdown func(ctx context.Context) error) {
	appKey := os.Getenv("APMPLUS_APP_KEY")
	if appKey == "" {
		return EmptyHertzConfigOption, nil, nil
	}
	region := os.Getenv("APMPLUS_REGION")
	if region == "" {
		region = "cn-beijing"
	}
	_, shutdown = initAPMPlusCallback(ctx, appKey, region)
	if !withHertzServer {
		return EmptyHertzConfigOption, nil, shutdown
	}
	// for hertz server, init hertz tracing integration
	tracer, cfg = initHertzTracing(ctx, appKey, region)
	return tracer, cfg, shutdown
}

// initAPMPlusCallback initializes the Eino APMPlus callback handler.
// It creates trace spans and collects metrics for Eino applications,
// which will be sent to the APMPlus server for observability.
func initAPMPlusCallback(ctx context.Context, appKey, region string) (callbacks.Handler, func(ctx context.Context) error) {
	ilog.EventInfo(ctx, "Init APMPlus callback, watch at: https://console.volcengine.com/apmplus-server", region)
	cbh, shutdown, err := apmplus.NewApmplusHandler(&apmplus.Config{
		Host:        fmt.Sprintf("apmplus-%s.volces.com:4317", region),
		AppKey:      appKey,
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

// initHertzTracing initializes Hertz framework tracing integration
// It creates trace spans and collects metrics for incoming HTTP requests,
// which will be sent to the APMPlus server for observability.
func initHertzTracing(ctx context.Context, appKey, region string) (hertzconfig.Option, *hertztracing.Config) {
	ilog.EventInfo(ctx, "Init Hertz Tracing", region)
	_ = provider.NewOpenTelemetryProvider(
		provider.WithServiceName("deer-go"),
		provider.WithExportEndpoint(fmt.Sprintf("apmplus-%s.volces.com:4317", region)),
		provider.WithInsecure(),
		provider.WithHeaders(map[string]string{"X-ByteAPM-AppKey": appKey}),
	)
	tracer, cfg := hertztracing.NewServerTracer()
	return tracer, cfg

}
