package tracing

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

const defaultMetricExportInterval = 10 * time.Second

func InitMeter(cfg Config) (func(context.Context) error, error) {
	if strings.TrimSpace(cfg.OTLPEndpoint) == "" {
		return nil, fmt.Errorf("otlp endpoint is required for metrics initialization")
	}

	ctx := context.Background()
	endpoint, insecure := normalizeOTLPEndpoint(cfg.OTLPEndpoint)

	exporterOpts := []otlpmetricgrpc.Option{otlpmetricgrpc.WithEndpoint(endpoint)}
	if insecure {
		exporterOpts = append(exporterOpts, otlpmetricgrpc.WithInsecure())
	}

	exporter, err := otlpmetricgrpc.New(ctx, exporterOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create otlp metric exporter: %w", err)
	}

	res, err := newResource(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create metric resource: %w", err)
	}

	reader := sdkmetric.NewPeriodicReader(
		exporter,
		sdkmetric.WithInterval(defaultMetricExportInterval),
	)

	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(reader),
	)
	otel.SetMeterProvider(meterProvider)

	return meterProvider.Shutdown, nil
}

func GetMeter(name string) metric.Meter {
	return otel.GetMeterProvider().Meter(name)
}

func normalizeOTLPEndpoint(raw string) (string, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", true
	}

	if !strings.Contains(trimmed, "://") {
		return trimmed, true
	}

	u, err := url.Parse(trimmed)
	if err != nil || u.Host == "" {
		return trimmed, true
	}

	switch strings.ToLower(u.Scheme) {
	case "https":
		return u.Host, false
	case "http":
		return u.Host, true
	default:
		return u.Host, true
	}
}
