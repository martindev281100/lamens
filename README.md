# kargo-bootstrap

[![Go Version](https://img.shields.io/badge/Go-1.25.1+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

`kargo-bootstrap` is a powerful CLI tool that streamlines the process of bootstrapping Kargo projects with ArgoCD integration. It provides a guided workflow for deploying applications to Kubernetes clusters using GitOps principles with Helm charts.

## Features

- **Interactive Deployment Workflow**: Step-by-step guided process for deploying applications
- **Git Repository Integration**: Clone and discover Helm charts from Git repositories
- **Multi-Environment Support**: Deploy to staging, production, and custom environments
- **ArgoCD Integration**: Create and manage ArgoCD Applications automatically
- **Helm Chart Discovery**: Automatically find and validate Helm charts in repositories
- **Environment Configuration**: Pre-configured environments with customizable settings
- **Dry-Run Mode**: Preview changes before applying them
- **Non-Interactive Mode**: Automate deployments in CI/CD pipelines
- **Comprehensive Error Handling**: Detailed error messages with suggestions

## Prerequisites

- Kubernetes cluster (v1.24+)
- ArgoCD installed in your cluster
- kubectl configured with cluster access
- Git repository containing Helm charts

## Installation

### From Source

```bash
git clone https://github.com/your-org/kargo-bootstrap.git
cd kargo-bootstrap
go build -o kargo-bootstrap cmd/kargo-bootstrap/main.go
sudo mv kargo-bootstrap /usr/local/bin/
```

### Using Go Install

```bash
go install github.com/your-org/kargo-bootstrap/cmd/kargo-bootstrap@latest
```

### Download Binary

Download the appropriate binary for your platform from the [Releases](https://github.com/your-org/kargo-bootstrap/releases) page.

## Quick Start

1. **Test your connection**:

   ```bash
   kargo-bootstrap argocd test
   ```

2. **List available ArgoCD projects**:

   ```bash
   kargo-bootstrap argocd list
   ```

3. **Deploy an application interactively**:

   ```bash
   kargo-bootstrap deploy
   ```

4. **Deploy with specific parameters**:
   ```bash
   kargo-bootstrap deploy \
     --repository-url https://github.com/example/helm-charts \
     --environments staging,production \
     --non-interactive
   ```

## Usage

### Main Commands

#### `deploy` - Deploy Applications

The main command for deploying applications with Kargo and ArgoCD integration.

```bash
# Interactive deployment
kargo-bootstrap deploy

# Non-interactive deployment
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --environments staging,production \
  --non-interactive

# Dry-run to preview changes
kargo-bootstrap deploy --dry-run

# Deploy with automatic approval
kargo-bootstrap deploy --auto-approve
```

#### `argocd` - Interact with ArgoCD

Manage and inspect ArgoCD projects and applications.

```bash
# List ArgoCD projects
kargo-bootstrap argocd list

# Get project details
kargo-bootstrap argocd get --project my-project

# Test ArgoCD connectivity
kargo-bootstrap argocd test

# Create application from YAML
kargo-bootstrap argocd create --file app.yaml

# List applications
kargo-bootstrap argocd list-apps

# Check application status
kargo-bootstrap argocd status --name my-app

# Sync an application
kargo-bootstrap argocd sync --name my-app
```

#### `git` - Test Git Operations

Test Git repository operations and discover Helm charts.

```bash
# Clone a repository
kargo-bootstrap git clone --url https://github.com/example/helm-charts

# List branches
kargo-bootstrap git list-branches --url https://github.com/example/helm-charts

# List tags
kargo-bootstrap git list-tags --url https://github.com/example/helm-charts

# Discover Helm charts
kargo-bootstrap git list-charts --url https://github.com/example/helm-charts

# Get chart information
kargo-bootstrap git chart-info --url https://github.com/example/helm-charts --chart-name myapp
```

#### `env` - Manage Environments

Configure and validate deployment environments.

```bash
# List available environments
kargo-bootstrap env list

# Add a custom environment
kargo-bootstrap env config add my-env

# Remove an environment
kargo-bootstrap env config remove my-env

# Validate environments
kargo-bootstrap env validate staging production

# Show default environments
kargo-bootstrap env defaults
```

#### `render` - Generate YAML

Render ArgoCD Application YAML for testing and preview.

```bash
# Render an application manifest
kargo-bootstrap render application \
  --name myapp \
  --project default \
  --repo https://github.com/example/helm-charts \
  --path charts/myapp

# Render Helm values
kargo-bootstrap render helm-values --name myapp --environment production
```

### Global Flags

```bash
--kubeconfig string        Path to the kubeconfig file (default is $HOME/.kube/config)
--argocd-namespace string  Namespace where ArgoCD is installed (default "argocd")
--verbose, -v              Enable verbose output with additional details
```

### Deployment Workflow

The kargo-bootstrap tool follows a structured deployment workflow:

1. **Project Selection**: Choose an ArgoCD project for deployment
2. **Repository Selection**: Select a Git repository containing Helm charts
3. **Revision Selection**: Choose a branch, tag, or commit to deploy
4. **Chart Discovery**: Discover and select available Helm charts
5. **Application Configuration**: Configure application name and environments
6. **Namespace Creation**: Create namespaces for each environment
7. **YAML Generation**: Generate ArgoCD Application manifests
8. **Deployment**: Apply manifests and monitor sync status

## Configuration

### Environment Configuration

Configure custom environments using a YAML file:

```yaml
# env-config.yaml
environments:
  - name: staging
    type: staging
    description: "Staging environment for testing"
    namespace: "staging"
    autoSync: true
    prune: true
    selfHeal: true
    replicaCount: 1

  - name: production
    type: production
    description: "Production environment"
    namespace: "production"
    autoSync: false
    prune: false
    selfHeal: true
    replicaCount: 3
```

Use the configuration:

```bash
kargo-bootstrap env list --config env-config.yaml
```

### Git Authentication

Configure Git authentication for private repositories:

```bash
# Using token
kargo-bootstrap deploy --token YOUR_TOKEN

# Using username/password
kargo-bootstrap deploy --username YOUR_USERNAME --password YOUR_PASSWORD

# Using SSH key
kargo-bootstrap deploy --ssh-key-path ~/.ssh/id_rsa
```

## Examples

### Basic Deployment

```bash
# Deploy a simple application
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --environments staging \
  --non-interactive
```

### Multi-Environment Deployment

```bash
# Deploy to multiple environments
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --environments staging,production \
  --auto-approve
```

### Private Repository

```bash
# Deploy from a private repository
kargo-bootstrap deploy \
  --repository-url https://github.com/example/private-charts \
  --token YOUR_GITHUB_TOKEN \
  --environments production
```

### Dry-Run and Validation

```bash
# Preview deployment without making changes
kargo-bootstrap deploy \
  --repository-url https://github.com/example/helm-charts \
  --environments staging \
  --dry-run
```

## Troubleshooting

### Common Issues

1. **ArgoCD Connection Failed**

   ```bash
   # Test ArgoCD connectivity
   kargo-bootstrap argocd test --kubeconfig ~/.kube/config
   ```

2. **Git Repository Access Denied**

   ```bash
   # Test Git operations
   kargo-bootstrap git clone --url https://github.com/example/repo --token YOUR_TOKEN
   ```

3. **Namespace Creation Failed**
   ```bash
   # Check permissions
   kubectl auth can-i create namespace
   ```

### Debug Mode

Enable verbose logging for detailed troubleshooting:

```bash
kargo-bootstrap deploy --verbose
```

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Support

- Create an issue in [GitHub Issues](https://github.com/your-org/kargo-bootstrap/issues)
- Check the [Documentation](docs/) for detailed guides
- Join our [Discord Community](https://discord.gg/kargo-bootstrap)
