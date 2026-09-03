# Production-Grade Go Microservices Platform Build Specification

## Purpose

Build the infrastructure, deployment, testing, observability, and production-hardening layer around an already-completed Go application repository.

The application repo is considered **done**. Do not spend significant effort rewriting application business logic. The goal is to turn the existing application into a production-style, deployable, observable, testable Kubernetes system.

Target architecture:

- ~8 Go microservices
- Docker
- Kubernetes
- Helm
- GitOps with ArgoCD
- Separate configuration repository
- Separate infrastructure/platform repository
- Database migrations
- Kafka/event-driven communication where already present
- Optional Istio service mesh
- Metrics, logs, and distributed tracing
- Integration, contract, E2E, load, stress, and failure testing
- CI/CD
- Rolling/canary deployment and rollback capability

---

# 1. Repository Model

Use a **3-repository model**.

```text
application-repo/
config-repo/
platform-infra/
```

## 1.1 Application Repository

Responsibility:

> What does the software do?

Expected structure:

```text
application-repo/
├── services/
│   ├── payment/
│   │   ├── cmd/
│   │   ├── internal/
│   │   ├── migrations/
│   │   ├── tests/
│   │   ├── go.mod
│   │   └── Dockerfile
│   ├── order/
│   └── user/
│
└── .github/workflows/
```

The real application repository already exists. Preserve its structure unless changes are required for deployment/observability.

Application repo may contain:

- Go source code
- Unit tests
- Integration tests
- DB migration files
- Dockerfiles
- CI definitions
- OpenTelemetry instrumentation
- Health/readiness endpoints
- `/metrics` endpoint if using Prometheus scraping

Do **not** put Kubernetes infrastructure here unless there is a specific reason.

---

# 2. Config Repository

Responsibility:

> What configuration values should the application use in each environment?

Recommended structure:

```text
config-repo/
├── payment/
│   ├── dev.yaml
│   ├── staging.yaml
│   └── prod.yaml
├── order/
│   ├── dev.yaml
│   ├── staging.yaml
│   └── prod.yaml
└── user/
    ├── dev.yaml
    ├── staging.yaml
    └── prod.yaml
```

Example:

```yaml
database:
  host: postgres.internal
  port: 5432
  name: payments

kafka:
  brokers:
    - kafka-1.internal
    - kafka-2.internal
  topic: payment-events

featureFlags:
  newPaymentFlow: true

timeouts:
  downstream: 2s
```

Config repo contains application behavior/configuration such as:

- DB hostname/database name
- Kafka topic names
- feature flags
- downstream URLs
- timeout values
- application-specific settings

Do not commit:

- passwords
- private keys
- API secrets
- tokens
- credentials

Use Kubernetes Secrets, External Secrets, Vault, or a cloud secret manager.

---

# 3. Platform / Infrastructure Repository

Responsibility:

> How should the application and platform be deployed and operated?

Recommended structure:

```text
platform-infra/
├── terraform/
│   ├── networking/
│   ├── eks/
│   ├── rds/
│   └── kafka/
│
├── helm/
│   ├── payment/
│   │   ├── Chart.yaml
│   │   ├── values.yaml
│   │   └── templates/
│   │       ├── deployment.yaml
│   │       ├── service.yaml
│   │       ├── ingress.yaml
│   │       ├── hpa.yaml
│   │       ├── pdb.yaml
│   │       └── migration-job.yaml
│   └── order/
│
├── argocd/
├── istio/
└── observability/
    ├── prometheus/
    ├── grafana/
    ├── loki/
    └── tempo/
```

This repository can be continuously changed. It is **not a one-time setup repo**.

Terraform provisions/changes infrastructure.

Helm defines Kubernetes application deployments.

ArgoCD reconciles Kubernetes with Git.

Istio manages service-mesh capabilities.

Observability components collect and display telemetry.

---

# 4. Config Repo vs Infra Repo

Use this rule:

> If changing the value changes application behavior, it belongs in config.

> If changing it changes how Kubernetes/platform runs the application, it belongs in infrastructure/deployment configuration.

Examples:

| Item | Repository |
|---|---|
| Kafka topic | Config |
| DB hostname | Config |
| Feature flag | Config |
| Downstream timeout | Config |
| Image tag | Infra/GitOps |
| Replica count | Infra/GitOps |
| CPU/memory | Infra/GitOps |
| HPA | Infra/GitOps |
| Kubernetes Deployment | Infra/GitOps |
| Kubernetes Service | Infra/GitOps |
| Ingress | Infra/GitOps |
| Istio | Infra/GitOps |
| Terraform VPC | Infra |
| EKS cluster | Infra |
| Prometheus | Infra/platform |
| Grafana | Infra/platform |

Helm charts should normally live in the **infra/GitOps repository**, not the config repository.

---

# 5. Helm Chart

A Helm chart is a directory, not a single "Helm file".

Example:

```text
helm/payment/
├── Chart.yaml
├── values.yaml
└── templates/
    ├── deployment.yaml
    ├── service.yaml
    ├── ingress.yaml
    ├── hpa.yaml
    ├── pdb.yaml
    └── migration-job.yaml
```

## Chart.yaml

```yaml
apiVersion: v2
name: payment-service
description: Helm chart for payment service
type: application
version: 1.0.0
appVersion: "1.0.0"
```

## values.yaml

Example:

```yaml
replicaCount: 3

image:
  repository: registry.company.com/payment-service
  tag: "a83f21"
  pullPolicy: IfNotPresent

service:
  type: ClusterIP
  port: 8080
  targetPort: 8080

resources:
  requests:
    cpu: 250m
    memory: 256Mi
  limits:
    cpu: 500m
    memory: 512Mi

env:
  configMapName: payment-service-config
  secretName: payment-service-secret
```

## Deployment

The deployment template should:

- use the configured image repository/tag
- expose the application port
- inject configuration
- reference secrets
- configure resource requests/limits
- configure liveness/readiness probes
- use sensible security context
- support rolling updates

Example core section:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Release.Name }}

spec:
  replicas: {{ .Values.replicaCount }}

  selector:
    matchLabels:
      app: {{ .Release.Name }}

  template:
    metadata:
      labels:
        app: {{ .Release.Name }}

    spec:
      containers:
        - name: payment-service
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
          imagePullPolicy: {{ .Values.image.pullPolicy }}

          ports:
            - containerPort: {{ .Values.service.targetPort }}

          envFrom:
            - configMapRef:
                name: {{ .Values.env.configMapName }}

            - secretRef:
                name: {{ .Values.env.secretName }}

          resources:
            {{- toYaml .Values.resources | nindent 12 }}
```

## Service

Use a ClusterIP service for internal communication unless external exposure is explicitly required.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: {{ .Release.Name }}
spec:
  type: {{ .Values.service.type }}
  selector:
    app: {{ .Release.Name }}
  ports:
    - port: {{ .Values.service.port }}
      targetPort: {{ .Values.service.targetPort }}
      protocol: TCP
```

## HPA

Example:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: {{ .Release.Name }}
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: {{ .Release.Name }}
  minReplicas: 3
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
```

Add configurable HPA values rather than hard-coding everything.

---

# 6. CI/CD and Image Flow

The application source code should not be deployed directly to Kubernetes.

Use immutable container images.

Example:

```text
Developer
   |
   | git push
   v
Application Repo
   |
   v
CI
   |
   +--> unit tests
   +--> integration tests
   +--> lint/static analysis
   +--> build
   |
   v
Docker Build
   |
   v
Container Registry
   |
   v
payment-service:a83f21
```

Prefer commit SHA/image digest based versioning:

```text
payment-service:a83f21c
```

Avoid using only:

```text
payment-service:latest
```

because immutable versions make rollback and debugging much easier.

---

# 7. GitOps Deployment Flow

Important distinction:

> A Docker image being pushed to the registry does not itself mean Helm runs.

Recommended flow:

```text
Application Repo
       |
       | CI
       v
Docker Image
       |
       v
Container Registry
       |
       | CI updates desired image version
       v
Platform/Infra Git repo
       |
       | Git commit
       v
ArgoCD detects change
       |
       v
ArgoCD reconciles Helm
       |
       v
Kubernetes
```

For example:

```yaml
image:
  repository: registry.company.com/payment-service
  tag: "a83f21"
```

CI can update only the image tag for a release.

The Helm chart itself does not need to change for every image release.

---

# 8. Environment Selection

Environment selection should be explicit.

Example ArgoCD applications:

```text
argocd/
├── payment-dev.yaml
├── payment-staging.yaml
└── payment-prod.yaml
```

Conceptually:

```text
payment-dev
    -> Helm values/dev.yaml

payment-staging
    -> Helm values/staging.yaml

payment-prod
    -> Helm values/prod.yaml
```

The promotion pipeline determines when an image moves through environments.

Do not rebuild the image separately for every environment.

Use the same artifact:

```text
payment:a83f21
       |
       +--> DEV
       |
       +--> STAGING
       |
       +--> PROD
```

Only environment-specific configuration/deployment state changes.

---

# 9. Environment Promotion

Recommended flow:

```text
Developer
   |
   v
CI
   |
   +--> build/test
   |
   v
payment:a83f21
   |
   v
DEV
   |
   +--> smoke
   +--> integration
   +--> contract
   +--> E2E
   |
   v
STAGING
   |
   +--> E2E
   +--> load
   +--> stress
   +--> security
   +--> migration validation
   |
   v
PROD
```

For image promotion:

```text
DEV values:
image.tag: a83f21

STAGING values:
image.tag: a83f21

PROD values:
image.tag: a83f21
```

This provides traceability and prevents environment-specific rebuilds.

---

# 10. Database Migrations

Migration files should live with the application because they represent application-owned schema evolution.

Example:

```text
payment/
├── cmd/
│   ├── server/
│   └── migrate/
├── internal/
├── migrations/
│   ├── 001_initial.sql
│   ├── 002_add_status.sql
│   └── 003_add_created_at.sql
└── Dockerfile
```

Use a migration tool such as:

- golang-migrate
- Flyway
- Liquibase

Do not run migrations blindly from every application pod.

Avoid:

```go
func main() {
    migrateDatabase()
    startServer()
}
```

If there are 10 pods, all 10 could attempt migration.

Instead:

```text
Migration = controlled one-off operation
Application = stateless server
```

## Migration Job

Helm can define a Kubernetes Job:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: {{ .Release.Name }}-migration
spec:
  backoffLimit: 3

  template:
    spec:
      restartPolicy: Never

      containers:
        - name: migration
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"

          command:
            - "/app/migrate"

          envFrom:
            - configMapRef:
                name: {{ .Values.env.configMapName }}

            - secretRef:
                name: {{ .Values.env.secretName }}
```

For production, use a controlled deployment sequence so:

```text
Migration succeeds
       |
       v
Application rollout
```

rather than letting migration and application deployment race.

---

# 11. Backward-Compatible DB Migration Strategy

Use:

```text
EXPAND
   |
   v
MIGRATE
   |
   v
DEPLOY
   |
   v
CONTRACT
```

Example:

Current DB:

```text
payments
--------
id
amount
```

New desired DB:

```text
payments
--------
id
amount
status
```

First add `status` in a backward-compatible way.

Then deploy code that uses `status`.

Only later remove obsolete schema elements.

Avoid deployments where the new application requires a DB column that does not yet exist.

---

# 12. Observability

Observability belongs primarily in the platform/infrastructure layer, but application instrumentation belongs in the application repo.

Recommended stack:

```text
Go Services
    |
    +---- Metrics -----> Prometheus
    |
    +---- Logs --------> Loki
    |
    +---- Traces ------> OpenTelemetry ---> Tempo
                                               |
                                               v
                                            Grafana
```

Application should provide/emit:

```text
/health
/ready
/metrics
```

and structured logs.

Use OpenTelemetry for distributed tracing.

Track at minimum:

- request rate
- error rate
- P50 latency
- P95 latency
- P99 latency
- CPU
- memory
- pod restarts
- DB latency
- Kafka consumer lag

Example distributed trace:

```text
API Gateway
    |
    | trace-id
    v
Order Service
    |
    | trace-id
    v
Payment Service
    |
    | trace-id
    v
Kafka
    |
    | trace-id
    v
Notification Service
```

Create Grafana dashboards that are actually useful for diagnosing production issues.

---

# 13. Istio

Istio is optional and should be added after the basic Kubernetes deployment works.

Use it to demonstrate:

- service-to-service traffic management
- retries
- timeouts
- circuit breaking
- traffic splitting
- canary releases
- service-to-service security
- telemetry

Do not add Istio merely because Kubernetes is being used.

Basic flow:

```text
Gateway
   |
   v
Istio
   |
   +--> Service A
   |
   +--> Service B
```

---

# 14. Testing Strategy

Use a test pyramid:

```text
              E2E
               ^
            Contract
               ^
          Integration
               ^
              Unit
```

Tests should run at the earliest useful stage.

## CI

Run:

- unit tests
- fast integration tests
- static analysis
- lint
- security scanning where appropriate
- build

## DEV

After deployment:

1. Smoke tests
2. Integration tests
3. Contract tests
4. E2E tests
5. Observability checks

## STAGING

Run:

1. Full E2E
2. Load tests
3. Stress tests
4. Security tests
5. Migration validation
6. Failure/recovery tests

---

# 15. Smoke Testing

Immediately after deployment, verify:

```text
GET /health
GET /ready
```

Then test a critical workflow.

Example:

```text
POST /payment
     |
     v
Payment Service
     |
     v
Database
     |
     v
Kafka
     |
     v
Consumer
```

Smoke tests should be fast.

Catch:

- pods not starting
- missing config
- DB connection failures
- Kafka failures
- wrong image
- service discovery failures
- readiness failures

---

# 16. Integration Testing

Verify communication between services:

```text
Order
  |
  v
Payment
  |
  v
Database
  |
  v
Kafka
  |
  v
Notification
```

Test:

- service-to-service calls
- DB access
- Kafka publishing
- Kafka consumption
- error handling
- timeouts

---

# 17. Contract Testing

With ~8 microservices, contract testing is important.

Example:

```text
Order Service
      |
      | POST /payments
      v
Payment Service
```

Consumer expects:

```json
{
  "orderId": "123",
  "amount": 500
}
```

Contract tests should catch breaking changes before production.

---

# 18. E2E Testing

Test actual business workflows.

Example:

```text
Create Order
     |
     v
Payment
     |
     v
Order Confirmed
     |
     v
Kafka Event
     |
     v
Notification
```

E2E tests should validate the complete user/business journey.

---

# 19. Load and Stress Testing

Run performance tests in staging.

Example:

```text
500 RPS
   |
1000 RPS
   |
2000 RPS
   |
3000 RPS
```

Measure:

- throughput
- P50
- P95
- P99
- error rate
- CPU
- memory
- DB utilization
- Kafka lag
- pod scaling

Do not simply record the maximum RPS.

Investigate the bottleneck.

Example target report:

```text
500 RPS   -> healthy
1000 RPS  -> healthy
2000 RPS  -> elevated latency
3000 RPS  -> failure threshold
```

The values above are examples; measure the actual system.

---

# 20. Failure Testing

Deliberately break components.

Examples:

```text
Kill a pod
      |
      v
Does Kubernetes recreate it?
Does traffic continue?
```

Dependency failure:

```text
Payment -> DB unavailable
```

Check:

- timeout
- retry behavior
- circuit breaker
- error response
- alerting
- recovery

Other scenarios:

- Kafka latency
- Kafka unavailable
- DB latency
- high request traffic
- pod crashes
- node failure where feasible

Use observability to understand the failure rather than merely confirming that the service failed.

---

# 21. Production Deployment

Use rolling deployment initially.

Potential advanced strategy:

```text
Old version
    |
    +--> 90% traffic
    |
New version
    |
    +--> 10% traffic
```

Then increase:

```text
10% -> 25% -> 50% -> 100%
```

Use Istio if implementing traffic-based canary releases.

Monitor:

- error rate
- latency
- saturation
- restarts
- business metrics

If metrics degrade, rollback.

---

# 22. Rollback

Image versions should be immutable.

Example:

```text
Current:
payment:a83f21

Previous:
payment:72cd11
```

Rollback should restore the previous desired state:

```text
payment:a83f21
       |
       | rollback
       v
payment:72cd11
```

With GitOps, reverting the deployment configuration allows ArgoCD to reconcile Kubernetes back to the previous version.

---

# 23. Infrastructure Responsibilities

Terraform should be responsible for infrastructure such as:

```text
VPC / networking
Kubernetes cluster
Node pools
Database infrastructure
Kafka infrastructure
Load balancers
IAM
Storage
```

Helm should primarily describe Kubernetes workloads:

```text
Deployment
Service
HPA
PDB
Ingress
ConfigMap references
Secret references
Istio resources
Migration Job
```

Do not confuse:

```text
Terraform
"What infrastructure exists?"

Helm
"How does the application run in Kubernetes?"

ArgoCD
"Make Kubernetes match Git."

Config
"What values does the application use?"
```

---

# 24. Final Architecture

```text
                         Developer
                             |
                          git push
                             |
                             v
                    Application Repo
                             |
                             v
                            CI
                   /         |                        Tests      Build     Security
                             |
                             v
                       Docker Image
                             |
                             v
                    Container Registry
                             |
                             | image version
                             v
                    Platform/Infra Repo
                             |
              +--------------+--------------+
              |              |              |
           Helm           ArgoCD         Terraform
              |              |              |
              +--------------+              |
                             |              |
                             v              v
                        Kubernetes      Cloud Infra
                             |
          +------------------+------------------+
          |                  |                  |
          v                  v                  v
      Service A          Service B          Service C
          |                  |                  |
          +------------------+------------------+
                             |
                    Config / Secrets
                             |
          +------------------+------------------+
          |                  |                  |
          v                  v                  v
      Prometheus           Loki              Tempo
          |                  |                  |
          +------------------+------------------+
                             |
                           Grafana
```

---

# 25. Runtime Architecture

For approximately eight microservices:

```text
                         Clients
                            |
                            v
                     Load Balancer
                            |
                            v
                     Ingress/Gateway
                            |
                            v
                       Istio Gateway
                            |
            +---------------+---------------+
            |               |               |
            v               v               v
       Payment           Order            User
       Service           Service          Service
            |               |               |
            +---------------+---------------+
                            |
                     Internal Services
                            |
                +-----------+-----------+
                |                       |
                v                       v
              Kafka                  Databases
```

---

# 26. Recommended Implementation Timeline

The application repository is already complete.

Target approximately **5–6 weeks at 10–12 hours/week**.

## Week 1 — Container + CI/CD

Deliver:

```text
git push
   |
   v
CI
   |
   +--> tests
   +--> build
   +--> lint
   |
   v
Docker image
   |
   v
Container Registry
```

## Week 2 — Kubernetes + Helm

Implement:

- Kubernetes deployment
- Service
- ConfigMap references
- Secret references
- health probes
- resource requests/limits
- HPA
- PDB
- DEV deployment

Goal:

```text
All services running in DEV Kubernetes.
```

## Week 3 — Config + ArgoCD + DB migration

Implement:

- environment configuration
- ArgoCD applications
- GitOps flow
- image promotion
- migration Job
- safe migration sequence

Goal:

```text
git push
  |
  v
image
  |
  v
GitOps update
  |
  v
ArgoCD
  |
  v
migration
  |
  v
Kubernetes rollout
```

## Week 4 — Observability

Implement:

- Prometheus
- Grafana
- Loki
- OpenTelemetry
- Tempo/Jaeger
- dashboards
- alerts

## Week 5 — Testing + Performance

Implement:

- smoke tests
- integration tests
- contract tests
- E2E tests
- load tests
- stress tests
- performance analysis

## Week 6 — Production Hardening

Implement:

- Istio
- retries/timeouts
- circuit breaking
- canary deployment if appropriate
- failure testing
- rollback
- documentation

---

# 27. Milestone Definition

The project is considered complete when the following flow works:

```text
Developer changes Go code
        |
        v
git push
        |
        v
CI
        |
        +--> tests
        +--> build
        +--> Docker
        |
        v
Container Registry
        |
        v
GitOps image update
        |
        v
ArgoCD
        |
        v
Helm
        |
        v
DB migration
        |
        v
Kubernetes rolling deployment
        |
        v
Health checks
        |
        v
Smoke tests
        |
        v
Integration / Contract / E2E
        |
        v
Observability
        |
        v
Staging load/stress
        |
        v
Production
        |
        v
Canary/Rolling + Monitoring
```

---

# 28. Engineering Principles

1. Keep application logic separate from deployment infrastructure.
2. Keep application configuration separate from deployment templates.
3. Use immutable image versions.
4. Do not use `latest` as the primary deployment version.
5. Do not run DB migrations independently from every application pod.
6. Prefer backward-compatible DB migrations.
7. Promote the same image through environments.
8. Let Git represent desired deployment state.
9. Let ArgoCD reconcile Kubernetes to Git.
10. Fail fast with cheap tests.
11. Run production-like performance tests in staging.
12. Make observability part of the system, not an afterthought.
13. Test failure and recovery, not only happy paths.
14. Keep infrastructure changes version-controlled.
15. Avoid adding complexity such as Istio until the basic Kubernetes deployment is stable.

---

# 29. Expected Final Project Story

The completed project should be explainable as:

> "I built an approximately eight-service Go system and productionized it using Docker, Kubernetes, Helm and GitOps. Application code, environment configuration, and platform/deployment configuration are separated into distinct repositories. CI builds immutable images and promotes the same artifact through DEV, staging, and production. ArgoCD reconciles the desired state from Git, Helm manages Kubernetes workloads, and database migrations run as controlled jobs using backward-compatible schema evolution. The platform includes metrics, logs, distributed tracing, alerting, service-to-service traffic management, autoscaling, and rollback. The system is validated through smoke, integration, contract, E2E, load, stress, and failure testing."

This should be the target architecture and implementation plan.
