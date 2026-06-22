# Kubernetes Production Shared Contract

This document is the shared production contract that must be referenced by both the app config repo and the platform infra repo.

## Objective

- Keep production ownership clear between config and infra.
- Avoid production drift from duplicate or conflicting docs.
- Standardize promotion, rollback, security, and observability expectations.

## Ownership Boundaries

### App Config Repo Owns

- Business app manifests (Deployment, Service, HPA, PDB).
- Environment overlays (`dev` and `prod`).
- App Ingress intent (host/path mapping for services).
- ExternalSecret references consumed by workloads.
- Production promotion PRs for image updates.

### Platform Infra Repo Owns

- Cloud and cluster provisioning (VPC, EKS, IAM, IRSA, storage classes).
- Platform controllers (Argo CD, ingress controller, External Secrets Operator, cert-manager if used).
- Shared runtime services when platform-managed (Postgres, RabbitMQ, Redis).
- Observability platform (OpenTelemetry collector, Prometheus, Loki, Grafana).
- Production governance (RBAC, policy checks, sync controls, auditability).

## Non-Negotiable Production Rules

1. No direct `kubectl apply` from app source CI into production.
2. Production desired state changes happen only via PR in app config repo or infra repo.
3. All images in production are immutable tags/digests.
4. Secrets never live in Git; use cloud secret manager -> External Secrets -> Kubernetes Secret.
5. Every production workload must have:
   - readiness and liveness probes
   - resource requests and limits
   - rollout strategy configured
6. Production ingress must terminate TLS with platform-managed certificate flow.

## Argo CD Model

- Config repo path for business apps: prod overlay only.
- Infra repo path for platform resources: prod platform overlays only.
- Suggested production behavior:
  - app sync: manual or gated auto-sync
  - infra sync: controlled/manual sync for risky changes

## Promotion Contract (Dev -> Prod)

1. Build once and publish immutable image.
2. Update config repo dev overlay and validate.
3. Promote the same immutable image to prod via PR.
4. Merge only after approvals and policy checks pass.
5. Argo CD reconciles cluster state from Git.

## Rollback Contract

### App Rollback

1. Revert last prod overlay commit in config repo.
2. Sync Argo application.
3. Validate health, error rate, and key user journeys.

### Infra Rollback

1. Revert last infra change commit in infra repo.
2. Reconcile via Argo/Terraform as applicable.
3. Validate cluster add-ons and observability baseline.

## Required PR Checks

### Config Repo Prod PR

- Manifest render succeeds.
- Schema validation succeeds.
- Policy validation succeeds.
- Immutable image reference present.
- Rollback note included.

### Infra Repo Prod PR

- Risk and blast radius documented.
- Backward compatibility impact documented.
- Rollback steps documented.
- Observability impact documented.

## Production Verification Checklist

- Argo applications report `Synced` and `Healthy`.
- No core workload in `CrashLoopBackOff` or prolonged `Pending`.
- Ingress DNS resolves and TLS is valid.
- Error rate/latency/saturation are within agreed SLO thresholds.
- Alerting and dashboards are healthy after stabilization window.

## Reference Instructions

Add this document link in both repos:

- In app config repo docs: link to this contract under production deployment process.
- In platform infra repo docs: link to this contract under integration and runbook sections.

Suggested link title:

- `Kubernetes Production Shared Contract`
