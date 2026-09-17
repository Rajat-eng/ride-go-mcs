# Prometheus

Metrics collection, storage, and querying for the observability stack.

## Overview

Prometheus scrapes metrics from service endpoints and stores them as time-series data. It provides:
- Metric collection from `/metrics` endpoints
- Multi-dimensional data storage
- Built-in alerting capabilities
- PromQL query language

## Configuration

### Charts

- `Chart.yaml` - Chart metadata
- `values.yaml` - Base Prometheus configuration
- `values-dev.yaml` - Development overrides (7 days retention, 5GB storage)
- `values-prod.yaml` - Production overrides (15 days retention, 10GB storage)

### Key Settings

- **Retention**: How long to keep metrics (dev: 7d, prod: 15d)
- **Storage**: Local persistent volume (dev: 5Gi, prod: 10Gi)
- **Resources**: CPU/memory limits (dev: 100m/256Mi, prod: 250m/512Mi)

## Scrape Targets

Add services to scrape by updating `prometheus.yml`:

```yaml
scrape_configs:
  - job_name: 'api-gateway'
    static_configs:
      - targets: ['api-gateway:8080']
    metrics_path: '/metrics'
  
  - job_name: 'kubernetes-pods'
    kubernetes_sd_configs:
      - role: pod
```

## Usage

```bash
# Deploy Prometheus
helm install prometheus . -f values-prod.yaml

# Access UI
kubectl port-forward svc/prometheus 9090:9090
# Visit http://localhost:9090
```

## Queries

Common PromQL queries:

```promql
# Request rate
rate(http_requests_total[5m])

# Error rate
rate(http_requests_total{status=~"5.."}[5m])

# Latency p99
histogram_quantile(0.99, http_request_duration_seconds_bucket)

# Pod memory usage
container_memory_usage_bytes
```

## Alerting

Define alerts in `prometheus-rules.yaml` and attach to Alertmanager for notifications.
