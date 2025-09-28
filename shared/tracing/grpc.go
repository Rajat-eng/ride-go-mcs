package tracing

import (
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"google.golang.org/grpc"
	"google.golang.org/grpc/stats"
)

// it sets vaious options for grpc server to initialize--> currently tracing options
func WithTracingInterceptors() []grpc.ServerOption {
	return []grpc.ServerOption{
		grpc.StatsHandler(newServerHandler()),
	}
}

// for grpx client initilaize it with various dial options to set up connections with server
// currnelt with tracing options and collect data in otel
func DialOptionsWithTracing() []grpc.DialOption {
	return []grpc.DialOption{
		grpc.WithStatsHandler(newClientHandler()),
	}
}

func newClientHandler() stats.Handler {
	return otelgrpc.NewClientHandler(otelgrpc.WithTracerProvider(otel.GetTracerProvider()))
}

func newServerHandler() stats.Handler {
	return otelgrpc.NewServerHandler(otelgrpc.WithTracerProvider(otel.GetTracerProvider()))
}
