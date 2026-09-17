# ArgoCD Applications

Production-grade GitOps deployment manifests for all microservices.

## Structure

- `apps-dev.yaml` - Development environment applications (8 services)
- `apps-staging.yaml` - Staging environment applications (8 services)
- `apps-prod.yaml` - Production environment applications (8 services)

## Deployment

Each application manifest contains ArgoCD Application resources that:

1. **Point to Helm charts** in the corresponding `helm/` directory
2. **Use environment-specific values** (values-dev.yaml, values-staging.yaml, values-prod.yaml)
3. **Enable automatic syncing** - ArgoCD watches the repo and applies changes automatically
4. **Prune resources** - When a service is removed from Git, it's deleted from the cluster
5. **Self-heal** - ArgoCD continuously reconciles cluster state to match Git

## Installation

1. **Install ArgoCD** (if not already installed):
   ```bash
   kubectl create namespace argocd
   kubectl apply -n argocd -f https://raw.githubusercontent.com/argoproj/argo-cd/stable/manifests/install.yaml
   ```

2. **Apply applications for your environment**:
   ```bash
   # Development
   kubectl apply -f argocd/apps-dev.yaml
   
   # Staging
   kubectl apply -f argocd/apps-staging.yaml
   
   # Production
   kubectl apply -f argocd/apps-prod.yaml
   ```

3. **Connect your Git repository** to ArgoCD:
   ```bash
   argocd repo add https://github.com/ride-go/platform-infra --username <username> --password <token>
   ```

## Environment Promotion Workflow

1. **Code merges** to `main` branch
2. **Container image** is built and pushed with tag (commit SHA or version)
3. **Update Helm values** - edit `helm/<service>/values-prod.yaml` to change image tag
4. **Commit to Git** - the change is automatically detected
5. **ArgoCD syncs** - applies the change to prod cluster
6. **No downtime** - rolling update handles pod replacement gracefully

## Manual Sync

Force ArgoCD to sync immediately:

```bash
argocd app sync <app-name>
```

## Monitoring Deployments

View application status in ArgoCD UI:

```bash
# Get ArgoCD admin password
kubectl -n argocd get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d

# Port-forward to ArgoCD server
kubectl -n argocd port-forward svc/argocd-server 8080:443

# Access UI at https://localhost:8080
```

## Secrets Management

For sensitive data (DB passwords, API keys, etc.):

1. **Option 1: Sealed Secrets**
   - Encrypt secrets in Git using `sealed-secrets` controller
   - Only the cluster can decrypt them
   - Commit encrypted secrets safely to Git

2. **Option 2: External Secrets**
   - Reference AWS Secrets Manager, Azure Key Vault, etc.
   - Pull secrets at runtime from external service
   - No sensitive data in Git

3. **Option 3: ArgoCD Vault Plugin**
   - Use HashiCorp Vault for centralized secret management
   - Plugin integrates with ArgoCD templating

See `helm/<service>/templates/secret.yaml` for where to populate sensitive values.
