package tracing

import (
	"net/http"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

func WrapHandler(handler http.Handler, operation string) http.Handler {
	return otelhttp.NewHandler(handler, operation)
}

func WrapHandlerFunc(handler http.HandlerFunc, operation string) http.Handler {
	return WrapHandler(handler, operation)
}
