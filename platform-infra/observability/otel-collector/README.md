# OpenTelemetry Collector

Central collection point for traces, metrics, and logs from all services.

## Overview

OpenTelemetry Collector provides:
- OTLP receiver for traces, metrics, logs
- Multi-protocol support (gRPC, HTTP)
- Service instrumentation pipeline
- Export to multiple backends (Prometheus, Loki, Tempo)
- Processing and transformation (batch, memory limits, resource attribution)

## Configuration

### Charts

- `Chart.yaml` - Chart metadata
- `values.yaml` - Base OTel configuration
- `values-dev.yaml` - Development overrides (100m/200Mi resources)
- `values-prod.yaml` - Production overrides (200m/400Mi resources)

### Deployment Modes

- **daemonset** (default) - Runs on every node for host/container metrics
- **deployment** - Centralized collector (for scaling)
- **statefulset** - With persistent state

## Usage

```bash
# Deploy OTel Collector
helm install otel-collector . -f values-prod.yaml

# Verify health
kubectl port-forward svc/otel-collector 13133:13133
curl http://localhost:13133/
```

## Receivers

- **OTLP gRPC** (4317) - OpenTelemetry Protocol (traces, metrics, logs)
- **OTLP HTTP** (4318) - OpenTelemetry HTTP Protocol
- **Prometheus** - Scrape Prometheus endpoints

## Exporters

- **OTLP** → Tempo (traces)
- **Prometheus** → Prometheus (metrics)
- **Loki** → Loki (logs)

## Service Instrumentation

Add to your services:

```go
import "go.opentelemetry.io/otel"

tracer := otel.Tracer("service-name")
meter := otel.Meter("service-name")
```

Configure OTLP endpoint:
```
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317
OTEL_EXPORTER_OTLP_PROTOCOL=grpc
```
