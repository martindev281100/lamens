# Advanced Features

This document covers advanced features and capabilities of kargo-bootstrap for power users and complex deployment scenarios.

## Multi-Repository Deployments

Deploy applications from multiple Git repositories in a single workflow:

```bash
# Deploy frontend from one repository
kargo-bootstrap deploy \
  --repository-url https://github.com/example/frontend-charts \
  --app-name frontend \
  --environments staging,production \
  --non-interactive

# Deploy backend from another repository
kargo-bootstrap deploy \
  --repository-url https://github.com/example/backend-charts \
  --app-name backend \
  --environments staging,production \
  --non-interactive
```

### Script for Multi-Repository Deployment

Create a script to deploy multiple applications:

```bash
#!/bin/bash
# deploy-all.sh

set -e

# Configuration
REPOSITORIES=(
  "https://github.com/example/frontend-charts:frontend"
  "https://github.com/example/backend-charts:backend"
  "https://github.com/example/api-charts:api"
)
ENVIRONMENTS="staging,production"

# Deploy each application
for repo_app in "${REPOSITORIES[@]}"; do
  IFS=':' read -r repo app <<< "$repo_app"
  echo "Deploying $app from $repo..."

  kargo-bootstrap deploy \
    --repository-url "$repo" \
    --app-name "$app" \
    --environments "$ENVIRONMENTS" \
    --non-interactive \
    --auto-approve

  echo "✅ $app deployed successfully"
done

echo "🚀 All applications deployed successfully"
```

## Custom Environment Configurations

Create sophisticated environment configurations with custom values, labels, and annotations:

```yaml
# advanced-env-config.yaml
environments:
  - name: development
    type: development
    description: "Development environment for local testing"
    namespace: "dev"
    autoSync: true
    prune: true
    selfHeal: true
    replicaCount: 1
    values:
      resources:
        limits:
          cpu: 500m
          memory: 512Mi
        requests:
          cpu: 100m
          memory: 128Mi
      env:
        LOG_LEVEL: debug
        FEATURE_FLAGS: "new-feature,beta-feature"
    labels:
      env: development
      team: platform
      cost-center: engineering
    annotations:
      kargo-bootstrap.io/environment: development
      kargo-bootstrap.io/managed-by: "kargo-bootstrap"
      monitoring.io/prometheus: "true"

  - name: staging
    type: staging
    description: "Staging environment for integration testing"
    namespace: "staging"
    autoSync: true
    prune: true
    selfHeal: true
    replicaCount: 2
    values:
      resources:
        limits:
          cpu: 1000m
          memory: 1Gi
        requests:
          cpu: 500m
          memory: 512Mi
      env:
        LOG_LEVEL: info
        FEATURE_FLAGS: "new-feature"
      affinity:
        podAntiAffinity:
          preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              podAffinityTerm:
                labelSelector:
                  matchExpressions:
                    - key: app
                      operator: In
                      values:
                        - myapp
                topologyKey: kubernetes.io/hostname
    labels:
      env: staging
      team: platform
      cost-center: engineering
    annotations:
      kargo-bootstrap.io/environment: staging
      kargo-bootstrap.io/managed-by: "kargo-bootstrap"
      monitoring.io/prometheus: "true"

  - name: production
    type: production
    description: "Production environment with high availability"
    namespace: "prod"
    autoSync: false
    prune: false
    selfHeal: true
    replicaCount: 5
    values:
      resources:
        limits:
          cpu: 2000m
          memory: 2Gi
        requests:
          cpu: 1000m
          memory: 1Gi
      env:
        LOG_LEVEL: warn
        FEATURE_FLAGS: ""
      affinity:
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchExpressions:
                  - key: app
                    operator: In
                    values:
                      - myapp
              topologyKey: kubernetes.io/hostname
      tolerations:
        - key: "dedicated"
          operator: "Equal"
          value: "production"
          effect: "NoSchedule"
      nodeSelector:
        node-type: production
    labels:
      env: production
      team: platform
      cost-center: engineering
    annotations:
      kargo-bootstrap.io/environment: production
      kargo-bootstrap.io/managed-by: "kargo-bootstrap"
      monitoring.io/prometheus: "true"
      security.io/scan: "true"
```

Use this configuration:

```bash
kargo-bootstrap env list --config advanced-env-config.yaml
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --environments staging,production
```

## Integration with CI/CD Pipelines

### GitHub Actions

```yaml
name: Deploy with kargo-bootstrap

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v3

      - name: Install kargo-bootstrap
        run: |
          curl -L https://github.com/your-org/kargo-bootstrap/releases/latest/download/kargo-bootstrap-linux-amd64.tar.gz | tar xz
          sudo mv kargo-bootstrap /usr/local/bin/

      - name: Test ArgoCD connectivity
        run: kargo-bootstrap argocd test

      - name: Deploy to staging
        if: github.ref == 'refs/heads/main'
        run: |
          kargo-bootstrap deploy \
            --repository-url ${{ github.repository }} \
            --environments staging \
            --non-interactive \
            --auto-approve
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

      - name: Deploy to production
        if: github.ref == 'refs/heads/main' && github.event_name == 'push'
        run: |
          kargo-bootstrap deploy \
            --repository-url ${{ github.repository }} \
            --environments production \
            --non-interactive \
            --auto-approve
        env:
          GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}

      - name: Verify deployment
        run: |
          kargo-bootstrap argocd status --name myapp-staging
          if [ "${{ github.ref }}" == "refs/heads/main" ]; then
            kargo-bootstrap argocd status --name myapp-production
          fi
```

### GitLab CI

```yaml
# .gitlab-ci.yml
stages:
  - test
  - deploy

variables:
  KUBECTL_VERSION: "1.24.0"
  KARGO_BOOTSTRAP_VERSION: "v0.1.0"

before_script:
  - |
    # Install kubectl
    curl -LO "https://dl.k8s.io/release/v${KUBECTL_VERSION}/bin/linux/amd64/kubectl"
    chmod +x kubectl
    sudo mv kubectl /usr/local/bin/

    # Install kargo-bootstrap
    curl -L "https://github.com/your-org/kargo-bootstrap/releases/download/${KARGO_BOOTSTRAP_VERSION}/kargo-bootstrap-linux-amd64.tar.gz" | tar xz
    sudo mv kargo-bootstrap /usr/local/bin/

    # Configure kubectl
    echo "$KUBECONFIG_CONTENT" | base64 -d > ~/.kube/config

test:
  stage: test
  script:
    - kargo-bootstrap argocd test
    - kargo-bootstrap git clone --url $CI_REPOSITORY_URL --dry-run

deploy_staging:
  stage: deploy
  script:
    - |
      kargo-bootstrap deploy \
        --repository-url $CI_REPOSITORY_URL \
        --environments staging \
        --non-interactive \
        --auto-approve
  environment:
    name: staging
    url: https://staging.example.com
  only:
    - main

deploy_production:
  stage: deploy
  script:
    - |
      kargo-bootstrap deploy \
        --repository-url $CI_REPOSITORY_URL \
        --environments production \
        --non-interactive \
        --auto-approve
  environment:
    name: production
    url: https://example.com
  when: manual
  only:
    - main
```

### Jenkins Pipeline

```groovy
pipeline {
    agent any

    environment {
        KUBECONFIG = credentials('kubeconfig')
        GITHUB_TOKEN = credentials('github-token')
    }

    stages {
        stage('Checkout') {
            steps {
                checkout scm
            }
        }

        stage('Install Tools') {
            steps {
                sh '''
                    # Install kargo-bootstrap
                    curl -L https://github.com/your-org/kargo-bootstrap/releases/latest/download/kargo-bootstrap-linux-amd64.tar.gz | tar xz
                    sudo mv kargo-bootstrap /usr/local/bin/
                '''
            }
        }

        stage('Test Connectivity') {
            steps {
                sh 'kargo-bootstrap argocd test'
            }
        }

        stage('Deploy to Staging') {
            steps {
                sh '''
                    kargo-bootstrap deploy \\
                        --repository-url ${GIT_URL} \\
                        --environments staging \\
                        --non-interactive \\
                        --auto-approve
                '''
            }
        }

        stage('Deploy to Production') {
            steps {
                input message: 'Deploy to production?', ok: 'Deploy'
                sh '''
                    kargo-bootstrap deploy \\
                        --repository-url ${GIT_URL} \\
                        --environments production \\
                        --non-interactive \\
                        --auto-approve
                '''
            }
        }
    }

    post {
        always {
            sh 'kargo-bootstrap argocd status --name myapp-staging'
        }
        success {
            echo 'Deployment successful!'
        }
        failure {
            echo 'Deployment failed!'
        }
    }
}
```

## Monitoring and Observability Integration

### Uptrace Integration

Configure observability with Uptrace:

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --environments production \
  --uptrace-endpoint http://uptrace-collector.uptrace.svc:4318 \
  --uptrace-dsn http://token@uptrace-collector.uptrace.svc:4318/2
```

### Custom Monitoring

Add custom monitoring configuration to your environment:

```yaml
environments:
  - name: production
    type: production
    values:
      monitoring:
        enabled: true
        serviceMonitor:
          enabled: true
          interval: 30s
        prometheusRules:
          enabled: true
        grafanaDashboard:
          enabled: true
          folder: "Applications"
```

## Advanced Git Operations

### Git Submodules

Work with repositories that use Git submodules:

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --git-submodules true \
  --environments production
```

### Sparse Checkout

Use sparse checkout for large repositories:

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --sparse-checkout "charts/*,docs/*" \
  --environments production
```

### Shallow Clone

Use shallow clone to speed up operations:

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --shallow-clone true \
  --depth 1 \
  --environments production
```

## Custom Chart Repositories

Use charts from custom repositories:

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --chart-repo-url https://charts.example.com \
  --chart-repo-name example-charts \
  --environments production
```

## Post-Deployment Hooks

Execute scripts after deployment:

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --environments production \
  --post-deploy-hook "./scripts/notify.sh" \
  --post-deploy-hook-args "--env production"
```

Example post-deployment script:

```bash
#!/bin/bash
# scripts/notify.sh

ENVIRONMENT=$1
APP_NAME="myapp"
STATUS="success"

# Send notification to Slack
curl -X POST -H 'Content-type: application/json' \
  --data "{\"text\":\"$APP_NAME deployed to $ENVIRONMENT: $STATUS\"}" \
  $SLACK_WEBHOOK_URL

# Run integration tests
./scripts/integration-tests.sh --env $ENVIRONMENT
```

## Custom Templates

Use custom ArgoCD Application templates:

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --environments production \
  --template-path ./templates/custom-app.yaml
```

Example custom template:

```yaml
# templates/custom-app.yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: {{.AppName}}-{{.Environment}}
  namespace: argocd
  labels:
    app: {{.AppName}}
    environment: {{.Environment}}
    team: {{.Team}}
spec:
  project: {{.Project}}
  source:
    repoURL: {{.RepoURL}}
    path: {{.Path}}
    targetRevision: {{.TargetRevision}}
    {{if .HelmValues}}
    helm:
      valueFiles: ["values.yaml"]
      values: |
        {{.HelmValues}}
    {{end}}
  destination:
    server: https://kubernetes.default.svc
    namespace: {{.Namespace}}
  syncPolicy:
    automated:
      prune: {{.Prune}}
      selfHeal: {{.SelfHeal}}
    syncOptions:
      - CreateNamespace=true
```

## Advanced Error Handling

### Retry Logic

Implement retry logic for failed deployments:

```bash
#!/bin/bash
# deploy-with-retry.sh

APP_NAME=$1
ENVIRONMENT=$2
MAX_RETRIES=3
RETRY_DELAY=30

for ((i=1; i<=MAX_RETRIES; i++)); do
  echo "Attempt $i of $MAX_RETRIES to deploy $APP_NAME to $ENVIRONMENT..."

  if kargo-bootstrap deploy \
    --repository-url https://github.com/example/helm-charts \
    --app-name "$APP_NAME" \
    --environments "$ENVIRONMENT" \
    --non-interactive \
    --auto-approve; then
    echo "✅ Deployment successful on attempt $i"
    exit 0
  else
    echo "❌ Deployment failed on attempt $i"
    if ((i < MAX_RETRIES)); then
      echo "Retrying in $RETRY_DELAY seconds..."
      sleep $RETRY_DELAY
    fi
  fi
done

echo "❌ All $MAX_RETRIES attempts failed"
exit 1
```

### Rollback Strategy

Implement a rollback strategy:

```bash
#!/bin/bash
# deploy-with-rollback.sh

APP_NAME=$1
ENVIRONMENT=$2
BACKUP_TAG="backup-$(date +%Y%m%d%H%M%S)"

# Create backup before deployment
echo "Creating backup tag $BACKUP_TAG..."
kubectl annotate application "$APP_NAME-$ENVIRONMENT" \
  "kargo-bootstrap.io/backup-tag=$BACKUP_TAG" \
  -n argocd

# Deploy new version
if kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --app-name "$APP_NAME" \
  --environments "$ENVIRONMENT" \
  --non-interactive \
  --auto-approve; then
  echo "✅ Deployment successful"

  # Wait for health check
  sleep 60

  # Check application health
  if kargo-bootstrap argocd status --name "$APP_NAME-$ENVIRONMENT" | grep -q "Healthy"; then
    echo "✅ Application is healthy"
  else
    echo "❌ Application is unhealthy, rolling back..."

    # Get the backup tag
    BACKUP_TAG=$(kubectl get application "$APP_NAME-$ENVIRONMENT" \
      -n argocd -o jsonpath='{.metadata.annotations.kargo-bootstrap\.io/backup-tag}')

    # Rollback to backup tag
    kargo-bootstrap argocd sync \
      --name "$APP_NAME-$ENVIRONMENT" \
      --revision "$BACKUP_TAG"

    echo "🔄 Rolled back to $BACKUP_TAG"
    exit 1
  fi
else
  echo "❌ Deployment failed"
  exit 1
fi
```

## Performance Optimization

### Parallel Deployments

Deploy to multiple environments in parallel:

```bash
#!/bin/bash
# parallel-deploy.sh

APP_NAME=$1
ENVIRONMENTS="staging,production"

# Function to deploy to a single environment
deploy_to_env() {
  local env=$1
  echo "Deploying $APP_NAME to $env..."

  if kargo-bootstrap deploy \
    --repository-url https://github.com/example/helm-charts \
    --app-name "$APP_NAME" \
    --environments "$env" \
    --non-interactive \
    --auto-approve; then
    echo "✅ $APP_NAME deployed to $env"
  else
    echo "❌ Failed to deploy $APP_NAME to $env"
    return 1
  fi
}

# Export the function for parallel execution
export -f deploy_to_env
export APP_NAME

# Deploy to all environments in parallel
echo "$ENVIRONMENTS" | tr ',' '\n' | xargs -I {} -P 0 bash -c 'deploy_to_env "$@"' _ {}

echo "🚀 All deployments completed"
```

### Optimized Git Operations

Optimize Git operations for large repositories:

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --environments production \
  --shallow-clone true \
  --depth 1 \
  --sparse-checkout "charts/*" \
  --git-protocol ssh
```

## Security Best Practices

### Secure Git Authentication

Use secure authentication methods:

```bash
# Use SSH key with passphrase
kargo-bootstrap deploy \
  --repository-url git@github.com:example/helm-charts.git \
  --ssh-key-path ~/.ssh/id_rsa \
  --ssh-passphrase-env SSH_PASSPHRASE

# Use short-lived token
TOKEN=$(curl -s -X POST "https://github.com/login/oauth/access_token" \
  -H "Accept: application/json" \
  -d "client_id=$CLIENT_ID&client_secret=$CLIENT_SECRET&code=$CODE" | jq -r .access_token)

kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --token "$TOKEN"
```

### Secure Image Pull

Use secure image pull secrets:

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --environments production \
  --image-pull-secret regcred \
  --image-pull-secret-namespace production
```

### RBAC Configuration

Configure proper RBAC for kargo-bootstrap:

```yaml
# rbac.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kargo-bootstrap
  namespace: argocd
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: kargo-bootstrap
rules:
  - apiGroups: [""]
    resources: ["namespaces"]
    verbs: ["create", "get", "list", "update", "patch"]
  - apiGroups: [""]
    resources: ["secrets"]
    verbs: ["create", "get", "list", "update", "patch"]
  - apiGroups: ["argoproj.io"]
    resources: ["applications", "appprojects"]
    verbs: ["create", "get", "list", "update", "patch", "delete"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: kargo-bootstrap
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: kargo-bootstrap
subjects:
  - kind: ServiceAccount
    name: kargo-bootstrap
    namespace: argocd
```

## Extending kargo-bootstrap

### Custom Plugins

Create custom plugins for kargo-bootstrap:

```go
// plugins/hello.go
package plugins

import (
    "fmt"
    "github.com/your-org/kargo-bootstrap/pkg/plugin"
)

type HelloPlugin struct{}

func (p *HelloPlugin) Name() string {
    return "hello"
}

func (p *HelloPlugin) Description() string {
    return "A simple hello world plugin"
}

func (p *HelloPlugin) Execute(ctx plugin.Context) error {
    name := ctx.GetString("name", "World")
    fmt.Printf("Hello, %s!\n", name)
    return nil
}

func init() {
    plugin.Register(&HelloPlugin{})
}
```

### Custom Commands

Create custom commands:

```go
// cmd/custom.go
package cmd

import (
    "fmt"
    "github.com/spf13/cobra"
)

var customCmd = &cobra.Command{
    Use:   "custom",
    Short: "A custom command",
    Long:  "This is a custom command for kargo-bootstrap",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Println("Executing custom command")
    },
}

func init() {
    rootCmd.AddCommand(customCmd)
}
```

These advanced features provide powerful capabilities for complex deployment scenarios. For more information, see the [Configuration Guide](configuration.md) and [Examples](examples.md).
