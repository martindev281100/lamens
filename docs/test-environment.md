# Kargo Bootstrap Test Environment

This document describes the test environment that has been set up for testing the kargo-bootstrap tool. The test environment includes a Kind cluster, Argo CD, and a test Git repository with sample Helm charts.

## Overview

The test environment is designed to provide a complete local testing setup for the kargo-bootstrap tool. It includes:

- A Kind cluster with proper port mappings for Argo CD
- Argo CD with pre-configured projects and permissions
- A test Git repository with sample Helm charts
- Scripts for setting up, cleaning up, and resetting the environment

## Prerequisites

Before setting up the test environment, ensure you have the following tools installed:

- Docker (for Kind)
- Kind (Kubernetes in Docker)
- kubectl (Kubernetes CLI)
- Helm (for Argo CD installation)
- Git (for repository operations)

## Environment Components

### Kind Cluster

The Kind cluster is configured with:

- Name: `kargo-test`
- Port mappings for Argo CD UI access
- NGINX Ingress Controller
- Proper node labels and taints

### Argo CD

Argo CD is installed with:

- Pre-configured projects (test-project, development, staging, production)
- Admin credentials (username: admin, password: argocd)
- Ingress configuration for UI access
- Proper permissions for testing

### Test Git Repository

The test Git repository includes:

- Sample Helm charts (web-app, api-service, database, monitoring)
- Environment-specific values files (dev, staging, prod)
- Configured to be accessible from within the cluster

## Setup Instructions

### Quick Setup

The easiest way to set up the test environment is to use the Makefile target:

```bash
make setup-test-env
```

This will run the complete setup process, including:

1. Creating the Kind cluster
2. Installing Argo CD
3. Setting up the test repository
4. Building the kargo-bootstrap binary

### Manual Setup

If you prefer to set up the environment manually, you can run the individual scripts:

1. Set up the Kind cluster:

   ```bash
   ./scripts/setup-kind.sh
   ```

2. Install Argo CD:

   ```bash
   export KUBECONFIG=${HOME}/.kube/config-kind-kargo-test
   ./scripts/install-argocd.sh
   ```

3. Set up the test repository:

   ```bash
   export KUBECONFIG=${HOME}/.kube/config-kind-kargo-test
   ./scripts/setup-test-repo.sh
   ```

4. Build the kargo-bootstrap binary:
   ```bash
   make build
   ```

## Access Information

After setting up the environment, you can access the components as follows:

### Kubernetes Cluster

To use the Kubernetes cluster, set the KUBECONFIG environment variable:

```bash
export KUBECONFIG=${HOME}/.kube/config-kind-kargo-test
```

### Argo CD UI

- URL: http://localhost/argocd
- Username: admin
- Password: argocd

### Git Server

- SSH URL: ssh://git@git-server.default.svc.cluster.local:30888/kargo-test-repo.git
- HTTP URL: http://git-server.default.svc.cluster.local:30889/kargo-test-repo.git

### Test Repository

- Location: ./kargo-test-repo
- Charts: web-app, api-service, database, monitoring
- Environments: dev, staging, prod

## Test Repository Structure

The test repository contains the following structure:

```
kargo-test-repo/
├── charts/
│   ├── web-app/
│   │   ├── Chart.yaml
│   │   ├── values.yaml
│   │   ├── values/
│   │   │   ├── dev.yaml
│   │   │   ├── staging.yaml
│   │   │   └── prod.yaml
│   │   └── templates/
│   │       ├── deployment.yaml
│   │       ├── service.yaml
│   │       ├── ingress.yaml
│   │       └── _helpers.tpl
│   ├── api-service/
│   │   ├── Chart.yaml
│   │   ├── values.yaml
│   │   ├── values/
│   │   │   ├── dev.yaml
│   │   │   ├── staging.yaml
│   │   │   └── prod.yaml
│   │   └── templates/
│   │       ├── deployment.yaml
│   │       ├── service.yaml
│   │       └── _helpers.tpl
│   ├── database/
│   │   ├── Chart.yaml
│   │   ├── values.yaml
│   │   ├── values/
│   │   │   ├── dev.yaml
│   │   │   ├── staging.yaml
│   │   │   └── prod.yaml
│   │   └── templates/
│   │       ├── deployment.yaml
│   │       ├── service.yaml
│   │       ├── pvc.yaml
│   │       ├── secret.yaml
│   │       └── _helpers.tpl
│   └── monitoring/
│       ├── Chart.yaml
│       ├── values.yaml
│       ├── values/
│       │   ├── dev.yaml
│       │   ├── staging.yaml
│       │   └── prod.yaml
│       └── templates/
│           ├── prometheus-deployment.yaml
│           ├── prometheus-service.yaml
│           ├── grafana-deployment.yaml
│           ├── grafana-service.yaml
│           └── _helpers.tpl
└── README.md
```

## Testing Scenarios

### Basic Workflow

To test the basic deployment workflow:

1. Set up your kubeconfig:

   ```bash
   export KUBECONFIG=${HOME}/.kube/config-kind-kargo-test
   ```

2. Run the basic workflow test:
   ```bash
   ./scripts/test-basic-workflow.sh
   ```

### Multi-Environment Deployment

To test multi-environment deployment:

1. Set up your kubeconfig:

   ```bash
   export KUBECONFIG=${HOME}/.kube/config-kind-kargo-test
   ```

2. Run the multi-environment test:
   ```bash
   ./scripts/test-multi-env.sh
   ```

### Error Scenarios

To test error handling:

1. Set up your kubeconfig:

   ```bash
   export KUBECONFIG=${HOME}/.kube/config-kind-kargo-test
   ```

2. Run the error scenarios test:
   ```bash
   ./scripts/test-error-scenarios.sh
   ```

### End-to-End Testing

To run all end-to-end tests:

```bash
make test-e2e
```

## Environment Management

### Reset Environment

To reset the environment to a clean state:

```bash
make reset-test-env
```

Or manually:

```bash
./scripts/reset-env.sh
```

This will:

- Delete all Argo CD applications
- Reset Argo CD projects
- Reset Kubernetes namespaces (except system namespaces)
- Reset Argo CD configuration
- Restart the Git server
- Reset the test repository

### Clean Up Environment

To completely clean up the test environment:

```bash
make cleanup-test-env
```

Or manually:

```bash
./scripts/cleanup-env.sh
```

This will:

- Delete the Kind cluster
- Remove the test repository
- Clean up build artifacts
- Clean up log files
- Clean up temporary files
- Clean up Docker resources

## Troubleshooting

### Common Issues

1. **Kind cluster creation fails**

   - Ensure Docker is running
   - Check if you have sufficient disk space
   - Verify Kind is installed correctly

2. **Argo CD installation fails**

   - Ensure Helm is installed
   - Check if the cluster has sufficient resources
   - Verify the kubeconfig is set correctly

3. **Git server setup fails**

   - Check if the cluster is running
   - Verify the Git server deployment
   - Check the Git server logs

4. **Test repository setup fails**
   - Ensure Git is installed
   - Check if you have write permissions in the project directory
   - Verify the Git server is accessible

### Debugging

1. Check the log files:

   - Setup log: `./setup-env.log`
   - Cleanup log: `./cleanup-env.log`
   - Reset log: `./reset-env.log`

2. Check the cluster status:

   ```bash
   kubectl cluster-info
   kubectl get nodes
   kubectl get pods -n argocd
   ```

3. Check Argo CD status:

   ```bash
   kubectl get pods -n argocd
   kubectl get applications -n argocd
   kubectl get appprojects -n argocd
   ```

4. Check Git server status:
   ```bash
   kubectl get pods -l app=git-server
   kubectl logs -l app=git-server
   ```

## Advanced Configuration

### Customizing the Kind Cluster

You can customize the Kind cluster by modifying the `scripts/setup-kind.sh` script. The cluster configuration is defined in the `create_kind_config` function.

### Customizing Argo CD

You can customize the Argo CD installation by modifying the `scripts/install-argocd.sh` script. The Argo CD configuration is defined in the `install_argocd` function.

### Customizing the Test Repository

You can customize the test repository by modifying the `scripts/setup-test-repo.sh` script. The repository structure and charts are defined in the `create_helm_charts` function.

## Contributing

When contributing to the test environment:

1. Update the documentation for any changes
2. Test the changes in the test environment
3. Ensure all test scenarios pass
4. Update the scripts as needed

## Resources

- [Kind Documentation](https://kind.sigs.k8s.io/)
- [Argo CD Documentation](https://argoproj.github.io/argo-cd/)
- [Helm Documentation](https://helm.sh/docs/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
