# Observability Stack

Production-grade monitoring, logging, and tracing platform.

## Components

- **Prometheus** (v2.45.0) - Metrics collection and alerting
- **Grafana** (v10.0.0) - Dashboards and visualization
- **Loki** (v2.9.0) - Log aggregation and querying
- **OpenTelemetry Collector** (v0.88.0) - OTLP receiver for traces, metrics, logs

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Observability Stack                      │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Services (Pod logs/metrics/traces)                         │
│              ↓ OTLP (traces/metrics/logs)                  │
│         ┌────────────────────────┐                         │
│         │ OTel Collector (0.88.0) │                         │
│         └───┬────────┬────────┬───┘                         │
│             ↓        ↓        ↓                             │
│  ┌──────────────┐ ┌────────┐ ┌─────────────┐               │
│  │Prometheus    │ │ Loki   │ │ Tempo       │               │
│  │(metrics)     │ │(logs)  │ │ (traces)    │               │
│  └────┬─────────┘ └───┬────┘ └────┬────────┘                    │
│       │            │             │                          │
│       └────────────┬─────────────┘                          │
│                    ↓                                        │
│              ┌──────────┐                                   │
│              │ Grafana  │──→ Dashboards, Alerts            │
│              └──────────┘                                   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

## Deployment

```bash
# Deploy entire observability stack for dev
helm install prometheus ./prometheus -f ./prometheus/values-dev.yaml
helm install grafana ./grafana -f ./grafana/values-dev.yaml
helm install loki ./loki -f ./loki/values-dev.yaml
helm install otel-collector ./otel-collector -f ./otel-collector/values-dev.yaml

# Or use ArgoCD for GitOps deployment
kubectl apply -f argocd/observability-apps-dev.yaml
```

## Access

```bash
# Grafana (dashboards)
kubectl port-forward svc/grafana 3000:3000
# Visit http://localhost:3000 (default: admin/admin123)

# Prometheus (metrics)
kubectl port-forward svc/prometheus 9090:9090
# Visit http://localhost:9090

# Loki (logs)
kubectl port-forward svc/loki 3100:3100

# OpenTelemetry Collector (traces/metrics/logs receiver)
kubectl port-forward svc/otel-collector 4317:4317
# OTLP gRPC receiver endpoint: localhost:4317
```

## Data Sources in Grafana

Pre-configured data sources: (metrics)
- **Loki** → `http://loki:3100` (logs)
- **Tempo** → `http://tempo:4317` (traces from OTel Collector)
- **Tempo** → `http://tempo:3100`

## Instrumentation

### Traces & Metrics

Send via OpenTelemetry to the Collector:
```
OTEL_EXPORTER_OTLP_ENDPOINT=http://otel-collector:4317
OTEL_EXPORTER_OTLP_PROTOCOL=grpc
```

### Logs

Collected automatically from pod stdout/stderr via Loki

### Native Prometheus Metrics

Scrape targets with `/metrics` endpoint (collected by OTel Prometheus receiver)

## Retention Policies

| Component | Dev | Staging | Prod |
|-----------|-----|---------|------|
| Prometheus | 7 days | 15 days | 15 days |
| Grafana | Indefinite | Indefinite | Indefinite |
| Loki | 3 days | 7 days | 7 days |
| OTel Collector | Stateless | Stateless | Stateless |

## Storage

All components use local persistent volumes by default. For production, consider:
- **S3-compatible storage** for metrics/logs (MinIO, AWS S3)
- **Distributed storage** for high availability
- **Automated backups** for Grafana dashboards
