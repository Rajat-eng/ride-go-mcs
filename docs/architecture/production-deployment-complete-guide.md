# Production Deployment Complete Guide

This is the single source of truth for production deployment in this repository.

Shared cross-repo production contract: `docs/architecture/k8s-prod-shared-contract.md`

It combines:

- beginner step-by-step operations
- IAM and entity responsibilities
- senior deployment standards
- compare-and-contrast maturity roadmap
- Helm, Terraform, Argo CD adoption path
- observability placement guidance
- automatic load balancer and DNS automation strategy

## Table Of Contents

1. Deployment Goals
2. Quick Beginner Steps
3. Current Deployment Architecture
4. IAM Roles And Policy Responsibilities
5. What Should Be Done vs What We Did
6. Senior Deployment Standard (How To Operate)
7. Observability Strategy
8. Automatic Load Balancer And DNS Attachment
9. Multi-Environment Model (How Big Teams Work)
10. Step-By-Step Maturity Path From Current Setup
11. Incident Response And Rollback
12. Production Checklist

## 1) Deployment Goals

- predictable deployments
- fewer manual fixes
- fast detection of failure
- fast rollback
- clear ownership of infra vs app changes

## 2) Quick Beginner Steps

Follow these in sequence.

1. Confirm repo and branch:

```bash
git status
git branch
```

2. Confirm GitHub settings:
- `vars.AWS_REGION`
- `secrets.AWS_ROLE_TO_ASSUME`
- `secrets.EKS_CLUSTER_NAME`

3. Confirm DNS resolves:

```bash
dig +short app.rexy.co.in
dig +short api.rexy.co.in
dig +short ws.rexy.co.in
```

4. Confirm cluster health:

```bash
kubectl get nodes
kubectl get pods -n kube-system
```

5. Confirm load balancer controller:

```bash
kubectl get deployment -n kube-system aws-load-balancer-controller
kubectl get pods -n kube-system | grep aws-load-balancer-controller
```

6. Confirm storage prereqs:

```bash
kubectl get storageclass
kubectl get csidrivers.storage.k8s.io
```

7. Trigger CD workflow and wait for completion.

8. If CD fails, download diagnostics artifact and inspect:
- `not_ready_pods.txt`
- `events.txt`
- `pod-logs.txt`
- `pod-describe.txt`

9. Verify rollout:

```bash
kubectl get pods -n default
kubectl get svc -n default
kubectl get ingress -n default
kubectl get endpoints -n default
```

10. Smoke test:

```bash
curl -i https://api.rexy.co.in/auth/signup
curl -I https://app.rexy.co.in
```

## 3) Current Deployment Architecture

Current pipeline behavior (from CD workflow):

1. Build and push images to ECR.
2. Configure EKS access.
3. Install External Secrets CRDs + controller.
4. Render manifests with image replacements.
5. Apply manifests in ordered phases:
- storage and base config
- observability stack
- secret sync resources
- stateful dependencies
- app workloads
- ingress
6. Wait for core rollouts.
7. On failure, collect diagnostics and upload artifact.

## 4) IAM Roles And Policy Responsibilities

## Deploy role (GitHub OIDC assumed role)

Must allow:

- ECR push/pull repository actions
- EKS cluster describe
- kubectl apply authorization via cluster RBAC mapping

## External Secrets IRSA role

Used by External Secrets controller service account.

Must allow:

- `secretsmanager:GetSecretValue`
- `secretsmanager:DescribeSecret`
- `secretsmanager:ListSecretVersionIds`
- `kms:Decrypt` (if CMK used)

## AWS Load Balancer Controller IRSA role

Used by `aws-load-balancer-controller` service account.

Must allow ALB/EC2 permissions, including account-specific additional permissions if required.

## Node/runtime roles

Must support:

- worker/node operations
- CSI-based volume provisioning
- image pull permissions

## 5) What Should Be Done vs What We Did

| Area | Senior Standard | Current State |
|---|---|---|
| Ordered deployment | Platform/deps first, ingress last | Implemented |
| Secret synchronization | Wait for SecretStore/ExternalSecret ready | Implemented |
| Stateful dependencies | Explicit manifests + storage checks | Implemented |
| Observability before app | OTel + Loki + Prometheus + Grafana ready first | Implemented |
| Failure diagnostics | Automatic artifact with logs/events/describes | Implemented |
| Public exposure policy | Keep logs/metrics private | Guidance present, enforceable policy can be improved |
| Environment promotion | dev -> stage -> prod | Not fully implemented |
| Terraform infra codification | VPC/EKS/IAM/DNS in Terraform | Not fully implemented |
| GitOps controller | Argo CD/Flux continuous reconciliation | Not yet implemented |
| Packaging strategy | Helm/Kustomize per environment | Not yet fully implemented |

## 6) Senior Deployment Standard (How To Operate)

Principles:

1. Platform first, app second.
2. Public traffic last.
3. No hidden dependencies.
4. Every deploy must be observable.
5. Rollback plan exists before rollout starts.

Deployment order:

1. cluster + IAM prechecks
2. observability and secret platform
3. stateful runtime dependencies
4. app workloads
5. ingress and DNS validation
6. smoke tests and stabilization window

## 7) Observability Strategy

## Should observability be in same cluster?

Option A: Same cluster (current)
- simpler and cheaper
- acceptable for current maturity

Option B: Separate observability environment
- stronger fault isolation
- more complex and costly

Recommendation now:

- keep in same cluster
- keep Loki/Prometheus private
- optionally expose Grafana internally only
- design configs so moving to separate stack later is easy

## Minimal operating checks

```bash
kubectl get pods -n default | egrep 'otel-collector|loki|prometheus|grafana'
kubectl run prom-query --rm -i --restart=Never --image=curlimages/curl:8.8.0 -- sh -c "curl -s 'http://prometheus:9090/api/v1/targets'"
kubectl run loki-query --rm -i --restart=Never --image=curlimages/curl:8.8.0 -- sh -c "curl -s http://loki:3100/loki/api/v1/labels"
```

## 8) Automatic Load Balancer And DNS Attachment

To avoid manual load balancer and manual DNS work:

1. Keep AWS Load Balancer Controller installed.
2. Manage all external routes via Kubernetes Ingress resources.
3. Let controller create/update ALB automatically.
4. Add ExternalDNS to create/update Route53 records automatically from Ingress/Service.
5. Use ACM-managed certificates and consistent ingress annotations.

Do not expose raw Loki/Prometheus endpoints publicly.

## 9) Multi-Environment Model (How Big Teams Work)

Typical model:

- dev: fast iteration
- stage: integration validation
- prod: customer traffic

Promotion model:

1. Build once, immutable image digest.
2. Deploy same digest to stage.
3. Run integration and SLO checks.
4. Approve promotion to prod.

GitOps model:

- CI builds image and updates env manifest values
- Argo CD syncs desired state
- drift automatically detected and reconciled

## 10) Step-By-Step Maturity Path From Current Setup

## Phase A (now)

1. Keep current CD ordered apply and gates.
2. Keep diagnostics artifacts on failures.
3. Keep observability stable and private.

## Phase B (near-term)

1. Add a stage environment.
2. Add stage secrets and stage domain.
3. Add promotion gate before prod deployment.

## Phase C (infra as code)

1. Codify IAM, IRSA, EKS, DNS, certs with Terraform.
2. Add terraform plan/apply with approvals.

## Phase D (packaging)

1. Convert k8s apps into Helm charts.
2. Move env-specific values to values files.

## Phase E (GitOps)

1. Install Argo CD.
2. Define Application/ApplicationSet for environments.
3. Shift deployment authority from direct kubectl apply to Argo sync.

## Phase F (enterprise controls)

1. Policy as code (OPA/Kyverno).
2. Alert routing (Slack/PagerDuty).
3. SLO dashboards and release blockers.

## 11) Incident Response And Rollback

When failure occurs:

1. Use diagnostics artifact first.
2. Confirm not-ready pods and recent events.
3. Confirm endpoints and readiness.
4. Roll back affected deployment only.

Rollback example:

```bash
kubectl rollout undo deployment/api-gateway -n default
kubectl rollout status deployment/api-gateway -n default
```

Avoid destructive broad commands in production.

## 12) Production Checklist

Before every release:

1. GitHub vars/secrets complete.
2. DNS resolves all public hosts.
3. ALB controller healthy.
4. CSI/storage class healthy.
5. Secret synchronization ready.
6. Observability stack healthy.
7. CD completes successfully.
8. Smoke tests pass.
9. Dashboards and logs visible.

If all nine pass, release is considered healthy.
