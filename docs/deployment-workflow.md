# Deployment Workflow

This document explains the deployment workflow used by kargo-bootstrap, which follows a structured process from project selection to application deployment.

## Overview

The kargo-bootstrap deployment workflow follows these steps:

1. **Project Selection** - Choose an ArgoCD project
2. **Repository Selection** - Select a Git repository
3. **Revision Selection** - Choose a branch, tag, or commit
4. **Chart Discovery** - Discover and select Helm charts
5. **Application Configuration** - Configure application name and environments
6. **Namespace Creation** - Create namespaces for each environment
7. **YAML Generation** - Generate ArgoCD Application manifests
8. **Deployment** - Apply manifests and monitor sync status

## Detailed Workflow

### Step 1: Project Selection

The first step is to select an ArgoCD project where the application will be deployed.

#### Interactive Mode

```bash
kargo-bootstrap deploy
```

You'll be prompted to select from available projects:

```
🔄 Step 1: Selecting ArgoCD Project
? Select an ArgoCD project: [Use arrows to move, type to filter]
> default     Default project
  production  Production apps
  staging     Staging environment
```

#### Non-Interactive Mode

```bash
kargo-bootstrap deploy --non-interactive
```

In non-interactive mode, kargo-bootstrap will:

1. Try to find a project named "default"
2. If not found, use the first available project
3. If no projects exist, exit with an error

#### API Call

The tool makes an API call to list ArgoCD projects:

```bash
kubectl get appprojects -n argocd
```

### Step 2: Repository Selection

Next, you'll select a Git repository containing Helm charts.

#### Interactive Mode

```
🔄 Step 2: Selecting Git Repository
? Enter Git repository URL: https://github.com/example/helm-charts
```

#### Non-Interactive Mode

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --non-interactive
```

#### Repository Validation

The tool validates the repository URL and checks if it's accessible:

```bash
git ls-remote https://github.com/example/helm-charts
```

### Step 3: Revision Selection

Choose a specific branch, tag, or commit to deploy.

#### Interactive Mode

```
🔄 Step 3: Selecting Git Revision
? Select revision: [Use arrows to move, type to filter]
> main       Main branch
  develop    Development branch
  v1.0.0     Tag: v1.0.0
  abc123def  Commit: abc123def
```

#### Non-Interactive Mode

```bash
# Deploy a specific branch
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --branch main \
  --non-interactive

# Deploy a specific tag
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --tag v1.0.0 \
  --non-interactive

# Deploy a specific commit
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --commit abc123def \
  --non-interactive
```

#### Git Operations

The tool performs these Git operations:

1. List branches: `git branch -r`
2. List tags: `git tag -l`
3. Clone repository: `git clone --branch <branch> <url>`
4. Checkout specific commit: `git checkout <commit>`

### Step 4: Chart Discovery

The tool discovers Helm charts in the selected repository.

#### Interactive Mode

```
🔄 Step 4: Discovering Helm Charts
✅ Found 3 chart(s):

NAME      VERSION    APP VERSION    DESCRIPTION           PATH
----      -------    -----------    -----------           ----
myapp     1.0.0      1.0.0          My application        charts/myapp
webapp    2.1.0      2.1.0          Web frontend          charts/webapp
api       0.5.0      0.5.0          API service           charts/api

? Select a chart: [Use arrows to move, type to filter]
> myapp     1.0.0    My application
```

#### Non-Interactive Mode

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --non-interactive
```

In non-interactive mode, the tool uses the first chart found.

#### Chart Discovery Process

1. Clone the repository to a temporary directory
2. Scan for `Chart.yaml` files
3. Parse chart metadata (name, version, description)
4. Validate chart structure
5. Display available charts

#### API Commands

```bash
# Find all Chart.yaml files
find . -name "Chart.yaml" -type f

# Parse chart metadata
helm inspect chart ./charts/myapp
```

### Step 5: Application Configuration

Configure the application name and target environments.

#### Interactive Mode

```
🔄 Step 5: Application Name
? Enter application name: myapp
✅ Application name: myapp

🔄 Step 6: Selecting Deployment Environments
? Select environments: [Use arrows to move, space to select, enter to confirm]
> [x] development
  [x] staging
  [ ] production
✅ Selected environments: development, staging
```

#### Non-Interactive Mode

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --app-name myapp \
  --environments staging,production \
  --non-interactive
```

#### Default Values

- Application name: Derived from chart name if not specified
- Environments: Default to "development" if not specified
- Namespace: Generated as `<app-name>-<environment>` if not specified

### Step 6: Namespace Creation

The tool creates namespaces for each selected environment.

#### Process

For each environment:

1. Generate namespace name: `<app-name>-<environment>`
2. Check if namespace exists
3. Create namespace if it doesn't exist
4. Add labels and annotations

#### Example

```bash
# Create namespace for development
kubectl create namespace myapp-development

# Add labels and annotations
kubectl label namespace myapp-development \
  kargo-bootstrap.io/app=myapp \
  kargo-bootstrap.io/environment=development \
  kargo-bootstrap.io/managed=true

kubectl annotate namespace myapp-development \
  kargo-bootstrap.io/created-at=2023-10-05T10:15:00Z \
  kargo-bootstrap.io/managed-by=kargo-bootstrap
```

#### Namespace Template

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: myapp-development
  labels:
    kargo-bootstrap.io/app: myapp
    kargo-bootstrap.io/environment: development
    kargo-bootstrap.io/managed: "true"
    app.kubernetes.io/name: myapp
    app.kubernetes.io/component: application
  annotations:
    kargo-bootstrap.io/created-at: "2023-10-05T10:15:00Z"
    kargo-bootstrap.io/managed-by: "kargo-bootstrap"
    kargo-bootstrap.io/project: "default"
    kargo-bootstrap.io/repository: "https://github.com/example/helm-charts"
```

### Step 7: YAML Generation

The tool generates ArgoCD Application manifests for each environment.

#### Process

For each environment:

1. Generate Helm values based on environment configuration
2. Create Application configuration
3. Render Application YAML
4. Validate the generated YAML

#### Helm Values Generation

```yaml
# Example Helm values for development
ght-app:
  replicaCount: 1
  port: 5005
  image:
    repository: ethannguyen98/myapp
    tag: v0.0.1
  imagePullSecrets:
    - name: docker
  envSecret: myapp-development-secret
  podLabels:
    app: myapp
    version: v0.0.1
  env:
    - name: OTEL_SERVICE_NAME
      value: myapp-development
    - name: OTEL_EXPORTER_OTLP_ENDPOINT
      value: http://uptrace-collector.uptrace.svc:4318
    - name: OTEL_EXPORTER_OTLP_HEADERS
      value: uptrace-dsn=http://token@uptrace-collector.uptrace.svc:4318/2
```

#### Application Configuration

```yaml
# Example Application configuration
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: myapp-development
  namespace: argocd
  labels:
    kargo-bootstrap.io/app: myapp
    kargo-bootstrap.io/environment: development
  annotations:
    kargo-bootstrap.io/created-at: "2023-10-05T10:15:00Z"
spec:
  project: default
  source:
    repoURL: https://github.com/example/helm-charts
    path: charts/myapp
    targetRevision: main
    helm:
      valueFiles: ["values.yaml"]
      values: |
        ght-app:
          replicaCount: 1
          imagePullSecrets:
            - name: docker
          podLabels:
            app: myapp
            version: v0.0.1
  destination:
    server: https://kubernetes.default.svc
    namespace: myapp-development
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
```

### Step 8: Deployment

The tool applies the generated manifests and monitors the sync status.

#### Process

For each environment:

1. Apply the Application manifest
2. Wait for the Application to be created
3. Check the initial sync status
4. Optionally wait for sync completion

#### API Commands

```bash
# Apply the Application manifest
kubectl apply -f myapp-development.yaml

# Get Application status
kubectl get application myapp-development -n argocd -o yaml

# Wait for sync completion (if requested)
argocd app wait myapp-development --timeout 300s
```

#### Sync Status Monitoring

The tool monitors these status fields:

- `status.health.status`: Healthy, Degraded, Missing, Progressing
- `status.sync.status`: Synced, OutOfSync, Unknown
- `status.sync.revision`: The currently synced revision

#### Example Output

```
✅ Successfully created Application 'myapp-development'
Initial status:
  Health: Healthy
  Sync: Synced
✅ Application 'myapp-development' is synced and healthy
  Health: Healthy
  Sync: Synced
  Revision: main
```

## Workflow Customization

### Custom Environment Configuration

You can customize the workflow by providing custom environment configurations:

```yaml
# my-env-config.yaml
environments:
  - name: production
    type: production
    description: "Production environment"
    namespace: "prod"
    autoSync: false
    prune: false
    selfHeal: true
    replicaCount: 3
    values:
      resources:
        limits:
          cpu: 2000m
          memory: 2Gi
        requests:
          cpu: 1000m
          memory: 1Gi
```

### Custom Helm Values

You can override default Helm values:

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --environments production \
  --image-repo myregistry/myapp \
  --image-tag v1.2.0
```

### Custom Application Name

You can specify a custom application name:

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --app-name my-custom-app \
  --environments staging,production
```

## Workflow Automation

### Non-Interactive Mode

For CI/CD automation, use non-interactive mode:

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --branch main \
  --app-name myapp \
  --environments staging,production \
  --non-interactive \
  --auto-approve
```

### Dry-Run Mode

For testing and validation, use dry-run mode:

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --environments staging \
  --dry-run
```

### Wait for Sync

For deployment pipelines, wait for sync completion:

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --environments production \
  --wait-for-sync
```

## Best Practices

1. **Use Specific Revisions**: Pin to specific tags or commits for production deployments
2. **Test in Staging**: Always deploy to staging before production
3. **Use Dry-Run**: Preview changes before applying them
4. **Monitor Sync Status**: Wait for sync completion in CI/CD pipelines
5. **Customize Environments**: Configure environments according to your needs
6. **Use Non-Interactive Mode**: Automate deployments in CI/CD pipelines

## Troubleshooting

If the workflow fails at any step:

1. **Check Verbose Output**: Use the `--verbose` flag for detailed logging
2. **Test Individual Components**: Use the `git` and `argocd` commands to test components
3. **Validate Configuration**: Use the `render` command to validate configuration
4. **Check Logs**: Review application and controller logs
5. **Manual Intervention**: Manually complete the failed step if needed

For more troubleshooting tips, see the [Troubleshooting Guide](troubleshooting.md).
