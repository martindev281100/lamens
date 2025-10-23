# Examples

This document provides practical examples of using kargo-bootstrap for various deployment scenarios.

## Basic Examples

### Interactive Deployment

Deploy an application with interactive prompts:

```bash
kargo-bootstrap deploy
```

### Non-Interactive Deployment

Deploy without prompts using command-line flags:

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --environments staging,production \
  --non-interactive
```

### Dry-Run Mode

Preview what would be deployed without making changes:

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --environments staging \
  --dry-run
```

## Git Repository Examples

### Public Repository

Deploy from a public Git repository:

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --environments staging
```

### Private Repository with Token

Deploy from a private repository using a token:

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/private-charts \
  --token YOUR_GITHUB_TOKEN \
  --environments production
```

### Private Repository with SSH Key

Deploy using SSH authentication:

```bash
kargo-bootstrap deploy \
  --repository-url git@github.com:example/private-charts.git \
  --ssh-key-path ~/.ssh/id_rsa \
  --environments production
```

### Specific Git Revision

Deploy a specific tag, branch, or commit:

```bash
# Deploy a specific tag
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --tag v1.2.0 \
  --environments production

# Deploy a specific branch
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --branch develop \
  --environments staging

# Deploy a specific commit
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --commit abc123def456 \
  --environments staging
```

## Environment Examples

### Single Environment

Deploy to a single environment:

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --environments production
```

### Multiple Environments

Deploy to multiple environments:

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --environments development,staging,production
```

### Custom Application Name

Specify a custom application name:

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --app-name my-custom-app \
  --environments staging
```

## ArgoCD Examples

### List Projects

List all available ArgoCD projects:

```bash
kargo-bootstrap argocd list
```

### List Projects with Details

Show detailed information about projects:

```bash
kargo-bootstrap argocd list --details
```

### Get Project Details

Get details about a specific project:

```bash
kargo-bootstrap argocd get --project my-project
```

### Create Application from YAML

Create an ArgoCD Application from a YAML file:

```bash
kargo-bootstrap argocd create --file my-app.yaml
```

### List Applications

List all ArgoCD Applications:

```bash
kargo-bootstrap argocd list-apps
```

### Check Application Status

Check the status of an application:

```bash
kargo-bootstrap argocd status --name my-app
```

### Sync Application

Manually sync an application:

```bash
kargo-bootstrap argocd sync --name my-app
```

### Sync with Specific Revision

Sync an application with a specific revision:

```bash
kargo-bootstrap argocd sync --name my-app --revision v1.2.0
```

## Git Operations Examples

### Clone Repository

Test cloning a repository:

```bash
kargo-bootstrap git clone --url https://github.com/example/helm-charts
```

### List Branches

List all branches in a repository:

```bash
kargo-bootstrap git list-branches --url https://github.com/example/helm-charts
```

### List Tags

List all tags in a repository:

```bash
kargo-bootstrap git list-tags --url https://github.com/example/helm-charts
```

### Discover Charts

List all Helm charts in a repository:

```bash
kargo-bootstrap git list-charts --url https://github.com/example/helm-charts
```

### Filter Charts by Name

Filter charts by name:

```bash
kargo-bootstrap git list-charts \
  --url https://github.com/example/helm-charts \
  --chart-name myapp
```

### Get Chart Information

Get detailed information about a specific chart:

```bash
kargo-bootstrap git chart-info \
  --url https://github.com/example/helm-charts \
  --chart-name myapp
```

## Environment Configuration Examples

### List Default Environments

List all default environments:

```bash
kargo-bootstrap env list
```

### List Custom Environments

List environments from a custom configuration file:

```bash
kargo-bootstrap env list --config my-env-config.yaml
```

### Add Custom Environment

Add a new environment:

```bash
kargo-bootstrap env config add production --auto-approve
```

### Remove Environment

Remove an environment:

```bash
kargo-bootstrap env config remove test --auto-approve
```

### Validate Environments

Validate specific environments:

```bash
kargo-bootstrap env validate staging production
```

### Validate All Environments

Validate all configured environments:

```bash
kargo-bootstrap env validate
```

## Rendering Examples

### Render Application Manifest

Render an ArgoCD Application manifest:

```bash
kargo-bootstrap render application \
  --name myapp \
  --project default \
  --repo https://github.com/example/helm-charts \
  --path charts/myapp
```

### Render with Custom Values

Render with custom values:

```bash
kargo-bootstrap render application \
  --name myapp \
  --project default \
  --repo https://github.com/example/helm-charts \
  --path charts/myapp \
  --environment production \
  --image-repo myregistry/myapp \
  --image-tag v1.2.0
```

### Render to File

Save the rendered manifest to a file:

```bash
kargo-bootstrap render application \
  --name myapp \
  --project default \
  --repo https://github.com/example/helm-charts \
  --path charts/myapp \
  --output myapp.yaml
```

### Render Helm Values

Render just the Helm values:

```bash
kargo-bootstrap render helm-values \
  --name myapp \
  --environment production
```

## Advanced Examples

### CI/CD Pipeline Integration

Example GitHub Actions workflow:

```yaml
name: Deploy with kargo-bootstrap

on:
  push:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2

      - name: Install kargo-bootstrap
        run: |
          curl -L https://github.com/your-org/kargo-bootstrap/releases/latest/download/kargo-bootstrap-linux-amd64.tar.gz | tar xz
          sudo mv kargo-bootstrap /usr/local/bin/

      - name: Deploy to staging
        run: |
          kargo-bootstrap deploy \
            --repository-url ${{ github.repository }} \
            --environments staging \
            --non-interactive \
            --auto-approve
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### Multi-Repository Deployment

Deploy applications from multiple repositories:

```bash
# Deploy frontend
kargo-bootstrap deploy \
  --repository-url https://github.com/example/frontend-charts \
  --app-name frontend \
  --environments staging,production \
  --non-interactive

# Deploy backend
kargo-bootstrap deploy \
  --repository-url https://github.com/example/backend-charts \
  --app-name backend \
  --environments staging,production \
  --non-interactive
```

### Custom ArgoCD Namespace

Use a custom ArgoCD namespace:

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --environments production \
  --argocd-namespace my-argocd
```

### Custom Kubeconfig

Use a custom kubeconfig file:

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --environments production \
  --kubeconfig /path/to/custom/kubeconfig
```

### Verbose Output

Enable verbose logging for troubleshooting:

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --environments staging \
  --verbose
```

## Error Handling Examples

### Retry Failed Deployments

If a deployment fails, you can retry with the same parameters:

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --environments staging \
  --auto-approve
```

### Validate Before Deployment

Validate your configuration before deploying:

```bash
kargo-bootstrap render application \
  --name myapp \
  --project default \
  --repo https://github.com/example/helm-charts \
  --path charts/myapp \
  --validate
```

### Test Git Operations

Test Git operations before deploying:

```bash
kargo-bootstrap git clone --url https://github.com/example/helm-charts
kargo-bootstrap git list-charts --url https://github.com/example/helm-charts
```

## Best Practices

1. **Use Dry-Run First**: Always use `--dry-run` to preview changes
2. **Start with Development**: Deploy to development first, then staging, then production
3. **Use Specific Revisions**: Pin to specific tags or commits for production deployments
4. **Validate Configuration**: Use the render command to validate before deploying
5. **Monitor Deployments**: Check application status after deployment

## Troubleshooting Examples

### Debug Connection Issues

Debug Kubernetes and ArgoCD connection issues:

```bash
kargo-bootstrap argocd test --verbose
```

### Debug Git Issues

Test Git operations with authentication:

```bash
kargo-bootstrap git clone \
  --url https://github.com/example/private-charts \
  --token YOUR_TOKEN
```

### Check Application Status

Check detailed application status:

```bash
kargo-bootstrap argocd status --name my-app
```

### Force Sync Application

Force sync an application even if no changes detected:

```bash
kargo-bootstrap argocd sync --name my-app --force
```
