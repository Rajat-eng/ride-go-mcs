package tracing

import (
	"context"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// RunInSpan wraps a function in a span and applies standard error status handling.
func RunInSpan(ctx context.Context, tracerName, operation string, attrs []attribute.KeyValue, fn func(context.Context, trace.Span) error) error {
	tracer := otel.GetTracerProvider().Tracer(tracerName)
	ctx, span := tracer.Start(ctx, operation, trace.WithAttributes(attrs...))
	defer span.End()

	err := fn(ctx, span)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}
	return err
}

// DBSpanAttrs adds the standard db.system attribute and appends any call-specific attributes.
func DBSpanAttrs(system string, attrs ...attribute.KeyValue) []attribute.KeyValue {
	return append([]attribute.KeyValue{attribute.String("db.system", system)}, attrs...)
}
