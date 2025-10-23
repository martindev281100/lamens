# Quick Start Guide

This guide provides a step-by-step workflow for first-time users to deploy an application with kargo-bootstrap in just a few minutes.

## Prerequisites

- Kubernetes cluster with ArgoCD installed
- kubectl configured with cluster access
- Git repository with Helm charts

## Step 1: Installation

```bash
# Install kargo-bootstrap
curl -L https://github.com/your-org/kargo-bootstrap/releases/latest/download/kargo-bootstrap-linux-amd64.tar.gz | tar xz
sudo mv kargo-bootstrap /usr/local/bin/

# Verify installation
kargo-bootstrap version
```

## Step 2: Verify Setup

```bash
# Test ArgoCD connectivity
kargo-bootstrap argocd test
```

Expected output:

```
Testing Kubernetes connection...
✅ Successfully connected to Kubernetes
Testing ArgoCD connectivity in namespace 'argocd'...
✅ Successfully connected to ArgoCD
✅ All ArgoCD connectivity tests passed!
```

## Step 3: Prepare Your Repository

Ensure your Git repository has this structure:

```
your-repo/
├── charts/
│   └── myapp/
│       ├── Chart.yaml
│       ├── values.yaml
│       └── templates/
```

## Step 4: Quick Interactive Deployment

```bash
# Start the deployment workflow
kargo-bootstrap deploy
```

Follow the prompts:

1. Select an ArgoCD project (usually "default")
2. Enter your Git repository URL
3. Select a branch/tag/commit
4. Choose a Helm chart
5. Enter an application name
6. Select environments (start with "development")
7. Confirm the deployment

## Step 5: Verify Deployment

```bash
# Check your applications
kargo-bootstrap argocd list-apps

# Check status of your app
kargo-bootstrap argocd status --name myapp-development

# Check Kubernetes resources
kubectl get pods -n myapp-development
```

## Expected Output

After successful deployment, you should see:

```
✅ Found 1 ArgoCD Application(s)
NAME               NAMESPACE    PROJECT    HEALTH    SYNC    REVISION
----               --------    -------    ------    ----    --------
myapp-development  argocd       default    Healthy   Synced  main
```

## Common Commands

```bash
# List available projects
kargo-bootstrap argocd list

# Check application status
kargo-bootstrap argocd status --name myapp-development

# Sync application manually
kargo-bootstrap argocd sync --name myapp-development

# Delete application
kargo-bootstrap argocd delete --name myapp-development
```

## Next Steps

- Read the [Getting Started Guide](getting-started.md) for detailed instructions
- Explore [Examples](examples.md) for advanced use cases
- Check [Configuration](configuration.md) for customization options

## Troubleshooting

If you encounter issues:

1. **ArgoCD connection failed**:

   ```bash
   kargo-bootstrap argocd test --verbose
   ```

2. **Git repository access denied**:

   ```bash
   # Test with authentication
   kargo-bootstrap git clone --url YOUR_REPO_URL --token YOUR_TOKEN
   ```

3. **Application not syncing**:
   ```bash
   kargo-bootstrap argocd status --name myapp-development
   kargo-bootstrap argocd sync --name myapp-development
   ```

For more help, see the [Troubleshooting Guide](troubleshooting.md).
