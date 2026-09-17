# Helm Charts

Complete Helm chart implementations for all ride-go services.

Each chart includes:
- Deployment with probes (readiness/liveness)
- Service (ClusterIP)
- Ingress (optional, disabled by default)
- HPA (horizontal pod autoscaling)
- PDB (pod disruption budget)
- ConfigMap and Secret integration
- Environment-specific values files (dev, staging, prod)

## Services

- api-gateway
- chat-service
- driver-service
- dlq-worker
- login-service
- payment-service
- trip-service
- ws-gateway

## Usage

Deploy to an environment with:

```bash
helm install <release-name> ./<service-name> -f ./<service-name>/values-<env>.yaml
```

Example:
```bash
helm install api-gateway ./api-gateway -f ./api-gateway/values-prod.yaml
```

## Chart Structure

Each service chart follows this layout:

```
service-name/
├── Chart.yaml                 # Chart metadata
├── values.yaml               # Base values (all envs inherit)
├── values-dev.yaml           # Dev overrides
├── values-staging.yaml       # Staging overrides
├── values-prod.yaml          # Prod overrides
└── templates/
    ├── _helpers.tpl          # Helm template macros
    ├── deployment.yaml       # Pod deployment
    ├── service.yaml          # Kubernetes service
    ├── ingress.yaml          # Ingress (optional)
    ├── hpa.yaml             # Horizontal pod autoscaler
    ├── pdb.yaml             # Pod disruption budget
    ├── configmap.yaml       # Configuration
    └── secret.yaml          # Secrets
```

## Environment Promotion

Each environment (dev, staging, prod) has a `values-<env>.yaml` that overrides base values:

- **Dev**: 1 replica, minimal resources, HPA disabled
- **Staging**: 2 replicas, moderate resources, HPA enabled
- **Prod**: 3 replicas, higher resources, PDB minAvailable increased

To deploy: merge base values.yaml with environment-specific overrides.
