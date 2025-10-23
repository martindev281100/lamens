# Configuration

This document explains the various configuration options available in kargo-bootstrap.

## Command-Line Flags

### Global Flags

These flags are available for all commands:

| Flag                 | Short | Description                                   | Default              |
| -------------------- | ----- | --------------------------------------------- | -------------------- |
| `--kubeconfig`       |       | Path to the kubeconfig file                   | `$HOME/.kube/config` |
| `--argocd-namespace` |       | Namespace where ArgoCD is installed           | `argocd`             |
| `--verbose`          | `-v`  | Enable verbose output with additional details | `false`              |

### Deploy Command Flags

| Flag                | Description                                                 | Default        |
| ------------------- | ----------------------------------------------------------- | -------------- |
| `--dry-run`         | Perform a dry-run without making any changes                | `false`        |
| `--non-interactive` | Run without interactive prompts                             | `false`        |
| `--auto-approve`    | Automatically approve deployments without confirmation      | `false`        |
| `--repository-url`  | Git repository URL containing Helm charts                   |                |
| `--app-name`        | Application name (derived from chart name if not specified) |                |
| `--environments`    | Comma-separated list of environments                        | `development`  |
| `--wait-for-sync`   | Wait for Applications to sync after creation                | `false`        |
| `--token`           | Authentication token for private repositories               |                |
| `--username`        | Username for authentication                                 |                |
| `--password`        | Password for authentication                                 |                |
| `--ssh-key-path`    | Path to SSH private key                                     |                |
| `--ssh-key`         | SSH private key content                                     |                |
| `--known-hosts`     | Path to known hosts file                                    |                |
| `--temp-dir`        | Temporary directory for cloning                             | Auto-generated |
| `--branch`          | Branch to clone                                             |                |
| `--tag`             | Tag to clone                                                |                |
| `--commit`          | Commit to checkout after cloning                            |                |

### Git Command Flags

| Flag              | Description                                   | Default        |
| ----------------- | --------------------------------------------- | -------------- |
| `--url`           | Repository URL (required)                     |                |
| `--token`         | Authentication token for private repositories |                |
| `--username`      | Username for authentication                   |                |
| `--password`      | Password for authentication                   |                |
| `--ssh-key-path`  | Path to SSH private key                       |                |
| `--ssh-key`       | SSH private key content                       |                |
| `--known-hosts`   | Path to known hosts file                      |                |
| `--temp-dir`      | Temporary directory for cloning               | Auto-generated |
| `--branch`        | Branch to clone                               |                |
| `--tag`           | Tag to clone                                  |                |
| `--commit`        | Commit to checkout after cloning              |                |
| `--shallow`       | Perform a shallow clone                       | `true`         |
| `--chart-name`    | Filter charts by name                         |                |
| `--chart-version` | Filter charts by version                      |                |
| `--chart-path`    | Path to the chart within the repository       |                |

### Environment Command Flags

| Flag             | Description                                                       | Default |
| ---------------- | ----------------------------------------------------------------- | ------- |
| `--config`       | Path to environment configuration file                            |         |
| `--auto-approve` | Automatically approve environment operations without confirmation | `false` |

### ArgoCD Command Flags

| Flag         | Description                                                 | Default |
| ------------ | ----------------------------------------------------------- | ------- |
| `--details`  | Show detailed information for each project                  | `false` |
| `--project`  | Name of the project to get details for                      |         |
| `--file`     | Path to the YAML file containing the Application definition |         |
| `--dry-run`  | Perform a dry-run without creating the Application          | `false` |
| `--name`     | Name of the Application to delete/get status/sync           |         |
| `--revision` | Specific revision to sync (default: latest)                 |         |
| `--prune`    | Prune resources during sync                                 | `false` |
| `--force`    | Force sync even if no changes detected                      | `false` |

### Render Command Flags

| Flag            | Description                                     | Default |
| --------------- | ----------------------------------------------- | ------- |
| `--name`        | Application name (required)                     |         |
| `--project`     | ArgoCD project name (required)                  |         |
| `--repo`        | Git repository URL (required)                   |         |
| `--path`        | Path to Helm chart (required)                   |         |
| `--revision`    | Target revision (default: main)                 |         |
| `--namespace`   | Target namespace (default: <app>-<env>)         |         |
| `--environment` | Environment (default: development)              |         |
| `--image-repo`  | Image repository (default: ethannguyen98/<app>) |         |
| `--image-tag`   | Image tag (default: v0.0.1)                     |         |
| `--output`      | Output file (default: stdout)                   |         |
| `--validate`    | Validate the generated YAML                     | `true`  |

## Environment Configuration

You can customize deployment environments using a YAML configuration file.

### Default Environment Configuration

kargo-bootstrap includes built-in default environments:

```yaml
environments:
  - name: development
    type: development
    description: "Development environment for testing"
    namespace: "development"
    autoSync: true
    prune: true
    selfHeal: true
    replicaCount: 1

  - name: staging
    type: staging
    description: "Staging environment for pre-production testing"
    namespace: "staging"
    autoSync: true
    prune: true
    selfHeal: true
    replicaCount: 2

  - name: production
    type: production
    description: "Production environment"
    namespace: "production"
    autoSync: false
    prune: false
    selfHeal: true
    replicaCount: 3
```

### Custom Environment Configuration

Create a custom environment configuration file:

```yaml
# my-env-config.yaml
environments:
  - name: testing
    type: testing
    description: "Testing environment for CI/CD"
    namespace: "testing"
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
    labels:
      env: testing
      team: platform
    annotations:
      kargo-bootstrap.io/managed-by: "kargo-bootstrap"

  - name: production
    type: production
    description: "Production environment with high availability"
    namespace: "production"
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
      env: production
      team: platform
    annotations:
      kargo-bootstrap.io/managed-by: "kargo-bootstrap"
```

### Using Custom Environment Configuration

```bash
# List environments from custom configuration
kargo-bootstrap env list --config my-env-config.yaml

# Validate environments from custom configuration
kargo-bootstrap env validate --config my-env-config.yaml

# Deploy using custom environments
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --environments testing,production
```

### Environment Configuration Schema

| Field          | Type    | Description                                               | Required |
| -------------- | ------- | --------------------------------------------------------- | -------- |
| `name`         | string  | Environment name                                          | Yes      |
| `type`         | string  | Environment type (development, staging, production, etc.) | Yes      |
| `description`  | string  | Environment description                                   | No       |
| `namespace`    | string  | Target namespace (default: same as name)                  | No       |
| `autoSync`     | boolean | Enable automatic sync in ArgoCD                           | No       |
| `prune`        | boolean | Enable pruning of resources                               | No       |
| `selfHeal`     | boolean | Enable self-healing in ArgoCD                             | No       |
| `replicaCount` | integer | Default replica count                                     | No       |
| `values`       | object  | Custom Helm values                                        | No       |
| `labels`       | object  | Custom labels                                             | No       |
| `annotations`  | object  | Custom annotations                                        | No       |

## Git Authentication Configuration

### Token Authentication

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/private-charts \
  --token YOUR_GITHUB_TOKEN
```

### Username/Password Authentication

```bash
kargo-bootstrap deploy \
  --repository-url https://gitlab.example.com/example/charts \
  --username YOUR_USERNAME \
  --password YOUR_PASSWORD
```

### SSH Key Authentication

```bash
# Using SSH key file
kargo-bootstrap deploy \
  --repository-url git@github.com:example/private-charts.git \
  --ssh-key-path ~/.ssh/id_rsa

# Using SSH key content
kargo-bootstrap deploy \
  --repository-url git@github.com:example/private-charts.git \
  --ssh-key "$(cat ~/.ssh/id_rsa)"
```

### Custom Known Hosts

```bash
kargo-bootstrap deploy \
  --repository-url git@gitlab.example.com:example/charts.git \
  --ssh-key-path ~/.ssh/id_rsa \
  --known-hosts ~/.ssh/custom_known_hosts
```

## ArgoCD Configuration

### Custom ArgoCD Namespace

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --environments production \
  --argocd-namespace my-argocd
```

### Custom Kubeconfig

```bash
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --environments production \
  --kubeconfig /path/to/custom/kubeconfig
```

## Configuration Files

### Environment Configuration File

Create a file named `.kargo-bootstrap.yaml` in your home directory or project root:

```yaml
# .kargo-bootstrap.yaml
environments:
  - name: development
    type: development
    description: "Development environment"
    autoSync: true
    prune: true
    selfHeal: true
    replicaCount: 1

  - name: staging
    type: staging
    description: "Staging environment"
    autoSync: true
    prune: true
    selfHeal: true
    replicaCount: 2

  - name: production
    type: production
    description: "Production environment"
    autoSync: false
    prune: false
    selfHeal: true
    replicaCount: 3
```

### Project Configuration File

Create a file named `.kargo-bootstrap-project.yaml` in your project root:

```yaml
# .kargo-bootstrap-project.yaml
project: my-project
repository: https://github.com/example/my-charts
defaultEnvironments: [staging, production]
defaultAppName: myapp
```

## Environment Variables

You can use environment variables to configure kargo-bootstrap:

| Variable            | Description                         | Default              |
| ------------------- | ----------------------------------- | -------------------- |
| `KUBECONFIG`        | Path to the kubeconfig file         | `$HOME/.kube/config` |
| `ARGOCD_NAMESPACE`  | Namespace where ArgoCD is installed | `argocd`             |
| `ARGOCD_SERVER`     | ArgoCD server URL                   |                      |
| `ARGOCD_AUTH_TOKEN` | ArgoCD authentication token         |                      |
| `GIT_TOKEN`         | Git authentication token            |                      |
| `GIT_USERNAME`      | Git username                        |                      |
| `GIT_PASSWORD`      | Git password                        |                      |
| `SSH_KEY_PATH`      | Path to SSH private key             |                      |
| `TEMP_DIR`          | Temporary directory for cloning     | System temp          |

## Configuration Precedence

Configuration is applied in the following order (highest to lowest priority):

1. Command-line flags
2. Environment variables
3. Project configuration file (`.kargo-bootstrap-project.yaml`)
4. User configuration file (`$HOME/.kargo-bootstrap.yaml`)
5. Default values

## Best Practices

1. **Use Configuration Files**: Store common settings in configuration files
2. **Environment-Specific Configs**: Use different configurations for different environments
3. **Secure Authentication**: Use SSH keys or tokens for authentication, not passwords
4. **Version Control**: Store configuration files in version control
5. **Validate Configuration**: Use the `--dry-run` flag to validate configuration before applying

## Example Configuration

Here's a complete example of a project configuration:

```yaml
# .kargo-bootstrap.yaml
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
    labels:
      env: development
      managed-by: kargo-bootstrap
    annotations:
      kargo-bootstrap.io/environment: development

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
    labels:
      env: staging
      managed-by: kargo-bootstrap
    annotations:
      kargo-bootstrap.io/environment: staging

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
      env: production
      managed-by: kargo-bootstrap
    annotations:
      kargo-bootstrap.io/environment: production
```

Use this configuration with:

```bash
kargo-bootstrap env list --config .kargo-bootstrap.yaml
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --environments staging,production
```
