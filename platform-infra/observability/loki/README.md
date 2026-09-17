# Loki

Log aggregation and querying engine for the observability stack.

## Overview

Loki provides:
- Logs indexed by labels, not full text
- LogQL query language (similar to PromQL)
- Multi-tenant log storage
- Efficient storage via chunk compression
- Integration with Grafana

## Configuration

### Charts

- `Chart.yaml` - Chart metadata
- `values.yaml` - Base Loki configuration
- `values-dev.yaml` - Development overrides (3 days retention, 5Gi storage)
- `values-prod.yaml` - Production overrides (7 days retention, 10Gi storage)

### Log Collection

Logs collected via:
- **Promtail** - DaemonSet scraping pod logs
- **Fluent Bit** - Alternative lightweight collector
- **Kubernetes API** - Pod log access

## Usage

```bash
# Deploy Loki
helm install loki . -f values-prod.yaml

# Query logs
kubectl port-forward svc/loki 3100:3100
```

## LogQL Queries

```logql
# All logs from api-gateway
{pod="api-gateway-*"}

# Error logs
{pod=~".*"} |= "ERROR"

# Request duration parsing
{pod="api-gateway-*"} | json | duration > 1000ms

# Aggregation
sum by (pod) (rate({job="kubernetes-pods"}[5m]))
```

## Retention

- Dev: 3 days
- Staging: 7 days  
- Prod: 7 days
