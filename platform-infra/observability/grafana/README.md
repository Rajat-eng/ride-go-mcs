# Grafana

Dashboarding and visualization platform for observability data.

## Overview

Grafana provides:
- Unified dashboards for metrics, logs, and traces
- Pre-configured data sources (Prometheus, Loki, Tempo)
- Alerting with notifications
- User management and RBAC
- Plugin ecosystem

## Configuration

### Charts

- `Chart.yaml` - Chart metadata
- `values.yaml` - Base Grafana configuration
- `values-dev.yaml` - Development overrides (50m/64Mi resources)
- `values-prod.yaml` - Production overrides (100m/128Mi resources)

### Data Sources

Pre-configured:
- **Prometheus** → `http://prometheus:9090` (metrics)
- **Loki** → `http://loki:3100` (logs)
- **Tempo** → Receives traces from OTel Collector (traces)

## Usage

```bash
# Deploy Grafana
helm install grafana . -f values-prod.yaml

# Access UI
kubectl port-forward svc/grafana 3000:3000
# Visit http://localhost:3000 (admin/admin123)
```

## Dashboards

Import dashboards from Grafana Lab:
- **Kubernetes Cluster** - Node/pod resources
- **Prometheus Stats** - Scrape metrics
- **Loki Dashboard** - Log overview
- **Traces** - Tempo traces

## Alerts

Define alert rules and configure notification channels:
- Email, Slack, PagerDuty, etc.
- Dynamic alert routing
- Alert grouping and silencing
