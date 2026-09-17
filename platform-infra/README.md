# Platform Infrastructure Repository

Production-grade infrastructure-as-code and deployment configuration for ride-go microservices.

## Directory Structure

```
platform-infra/
├── terraform/              # Infrastructure modules
│   ├── modules/           # Reusable terraform modules
│   │   ├── vpc/          # VPC, subnets, gateways
│   │   ├── eks/          # EKS cluster & node groups
│   │   ├── rds/          # PostgreSQL databases
│   │   ├── redis/        # ElastiCache Redis
│   │   └── rabbitmq/     # Amazon MQ RabbitMQ
│   └── environments/      # Environment-specific configs
│       ├── dev/
│       ├── staging/
│       └── prod/
├── helm/                   # Kubernetes deployment charts (10 services)
├── argocd/                 # GitOps application configurations
├── observability/         # Monitoring & logging stack
└── istio/                 # Service mesh configs
```

## Helm Charts

Complete production-ready charts for all services:

- **api-gateway** - HTTP entry point with routing & auth
- **chat-service** - Real-time messaging
- **driver-service** - Driver management & location tracking
- **dlq-worker** - Dead letter queue processor
- **login-service** - Authentication & user sessions
- **payment-service** - Payment processing
- **trip-service** - Trip lifecycle management
- **ws-gateway** - WebSocket gateway for real-time events
- **ride-service** - Ride management
- **user-service** - User profile management

### Chart Features

Each chart includes:
- Deployment with health probes (readiness/liveness)
- ClusterIP Service for internal DNS
- Horizontal Pod Autoscaling (dev: disabled, staging/prod: enabled)
- Pod Disruption Budgets (prod: minAvailable=2, others: 1)
- Environment-specific ConfigMap and Secret integration
- Resource limits: dev (50m/64Mi req, 250m/256Mi limit), staging (100m/128Mi), prod (250m/256Mi)

### Deployment

```bash
# Deploy to dev
helm install <service-name> ./helm/<service-name> -f ./helm/<service-name>/values-dev.yaml

# Deploy to prod
helm install <service-name> ./helm/<service-name> -f ./helm/<service-name>/values-prod.yaml
```

## Terraform

### Modules

**vpc/main.tf**
- Public/private subnets per AZ
- Internet gateway & NAT gateway
- Route tables for pod traffic

**eks/main.tf**
- EKS cluster with v1.30
- Managed node groups with auto-scaling
- IAM roles and security groups

**rds/main.tf**
- PostgreSQL multi-AZ (prod) or single-AZ (dev/staging)
- Automatic storage autoscaling (up to 100GB)
- Encryption at rest

**redis/main.tf**
- ElastiCache Redis cluster
- Configurable node count and type
- In-VPC deployment

**rabbitmq/main.tf**
- Amazon MQ RabbitMQ broker
- AMQP 5672 & management console 15672
- High availability options

### Environment Configuration

Each environment (dev, staging, prod) includes:
- `terraform.tfvars` - Environment-specific defaults
- `variables.tf` - Input variable contracts
- `main.tf` - Module wiring
- `outputs.tf` - Resource identifiers for ArgoCD

**Cost Optimization**: Resource creation toggles default to `false` in dev to minimize costs.

## ArgoCD

GitOps application manifests for continuous deployment.

Each service + environment combination has an Application resource that:
- Points to corresponding Helm chart
- Uses environment-specific values files
- Enables auto-sync after merge to main
- Manages secrets via Sealed Secrets or External Secrets

## Observability

Production monitoring stack:
- **Prometheus** - Metrics collection
- **Grafana** - Dashboards
- **Loki** - Log aggregation
- **AlertManager** - Alerting

## Quick Start

1. **Provision Infrastructure**
   ```bash
   cd terraform/environments/prod
   terraform init
   terraform apply -var-file="terraform.tfvars"
   ```

2. **Deploy Services**
   ```bash
   # Via ArgoCD (recommended)
   kubectl apply -f argocd/app-api-gateway-prod.yaml
   
   # Or via Helm directly
   helm install api-gateway ./helm/api-gateway -f ./helm/api-gateway/values-prod.yaml
   ```

3. **Monitor**
   - Prometheus: `http://prometheus:9090`
   - Grafana: `http://grafana:3000`
   - Loki: `http://loki:3100`

## Environment Promotion

1. Code changes merge to `main` branch
2. Container image built and tagged with commit SHA
3. Promotion: update image tag in `values-{env}.yaml`
4. ArgoCD detects change and auto-syncs

This maintains:
- **Same image** across environments (no rebuilds)
- **Different config** via Helm values
- **Audit trail** through Git commits
