# Tempo

Distributed tracing backend for request tracing across services.

## Overview

Tempo provides:
- Distributed tracing storage
- Multiple trace formats (Jaeger, Zipkin, OpenTelemetry)
- Trace querying and correlation
- Integration with Grafana
- Service dependency discovery

## Configuration

### Charts

- `Chart.yaml` - Chart metadata
- `values.yaml` - Base Tempo configuration
- `values-dev.yaml` - Development overrides (3 days retention, 5Gi storage)
- `values-prod.yaml` - Production overrides (7 days retention, 10Gi storage)

### Trace Collection

Traces collected via:
- **OpenTelemetry Collector** - OTLP protocol ingestion
- **Jaeger Agent** - DaemonSet forwarding traces
- **Zipkin Receiver** - Alternative trace format

## Usage

```bash
# Deploy Tempo
helm install tempo . -f values-prod.yaml

# Query traces
kubectl port-forward svc/tempo 3100:3100
```

## Instrumentation

Add tracing to Go services:

```go
import "go.opentelemetry.io/otel"

otel.Tracer("my-service").Start(ctx, "operation-name")
```

## Retention

- Dev: 3 days
- Staging: 7 days
- Prod: 7 days

## Service Graph

Tempo automatically builds service dependency graph from traces, visible in Grafana.
