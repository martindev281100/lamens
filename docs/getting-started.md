# Getting Started with kargo-bootstrap

This guide will help you get up and running with kargo-bootstrap, a CLI tool for bootstrapping Kargo projects with ArgoCD integration.

## Prerequisites

Before you begin, ensure you have the following:

### Required

- **Kubernetes cluster** (v1.24 or later)
- **kubectl** configured with cluster access
- **ArgoCD** installed in your cluster
- **Git repository** containing Helm charts

### Optional

- **Docker registry** access for container images
- **Monitoring tools** (Uptrace/OTEL) for observability

## Installation

### Option 1: Download Binary

1. Go to the [Releases](https://github.com/your-org/kargo-bootstrap/releases) page
2. Download the appropriate binary for your platform
3. Extract and move to your PATH:

```bash
# For Linux/macOS
tar -xzf kargo-bootstrap-v0.1.0-linux-amd64.tar.gz
sudo mv kargo-bootstrap /usr/local/bin/

# For Windows
# Extract and add to your PATH
```

### Option 2: Build from Source

```bash
git clone https://github.com/your-org/kargo-bootstrap.git
cd kargo-bootstrap
go build -o kargo-bootstrap cmd/kargo-bootstrap/main.go
sudo mv kargo-bootstrap /usr/local/bin/
```

### Option 3: Using Go Install

```bash
go install github.com/your-org/kargo-bootstrap/cmd/kargo-bootstrap@latest
```

## Initial Setup

### 1. Verify Installation

```bash
kargo-bootstrap version
```

You should see version information similar to:

```
kargo-bootstrap version v0.1.0
Commit: abc123def
Built: 2023-10-05T10:15:00Z
```

### 2. Test Kubernetes Connection

Ensure kubectl can connect to your cluster:

```bash
kubectl cluster-info
```

### 3. Test ArgoCD Installation

Verify ArgoCD is installed and accessible:

```bash
kargo-bootstrap argocd test
```

Expected output:

```
Testing Kubernetes connection...
✅ Successfully connected to Kubernetes
Current context: my-cluster

Testing ArgoCD connectivity in namespace 'argocd'...
✅ Successfully connected to ArgoCD

Testing project listing...
✅ Successfully listed 2 project(s)
Available projects:
  - default
  - production
```

If this fails, check that ArgoCD is installed in the correct namespace:

```bash
# Check ArgoCD installation
kubectl get pods -n argocd

# If ArgoCD is in a different namespace
kargo-bootstrap argocd test --argocd-namespace my-argocd
```

## Your First Deployment

Let's walk through deploying your first application using kargo-bootstrap.

### Step 1: List Available Projects

```bash
kargo-bootstrap argocd list
```

This will show you available ArgoCD projects:

```
✅ Found 2 ArgoCD project(s)
NAME        DESCRIPTION           SOURCE REPOS    DESTINATIONS
----        -----------           ------------    ------------
default     Default project       2 repos         1 dest
production  Production apps       1 repo          2 dest
```

### Step 2: Prepare Your Git Repository

Ensure your Git repository contains Helm charts. The repository should have:

```
your-repo/
├── charts/
│   ├── myapp/
│   │   ├── Chart.yaml
│   │   ├── values.yaml
│   │   └── templates/
│   │       ├── deployment.yaml
│   │       ├── service.yaml
│   │       └── ...
│   └── another-app/
│       ├── Chart.yaml
│       └── ...
└── README.md
```

### Step 3: Interactive Deployment

Run the deploy command to start the interactive workflow:

```bash
kargo-bootstrap deploy
```

You'll be guided through the following steps:

#### Step 3.1: Select ArgoCD Project

```
🔄 Step 1: Selecting ArgoCD Project
? Select an ArgoCD project: [Use arrows to move, type to filter]
> default     Default project
  production  Production apps
```

#### Step 3.2: Select Git Repository

```
🔄 Step 2: Selecting Git Repository
? Enter Git repository URL: https://github.com/your-org/helm-charts
```

#### Step 3.3: Select Git Revision

```
🔄 Step 3: Selecting Git Revision
? Select revision: [Use arrows to move, type to filter]
> main       Main branch
  develop    Development branch
  v1.0.0     Tag: v1.0.0
  abc123def  Commit: abc123def
```

#### Step 3.4: Select Helm Chart

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

#### Step 3.5: Configure Application

```
🔄 Step 5: Application Name
? Enter application name: myapp
✅ Application name: myapp
```

#### Step 3.6: Select Environments

```
🔄 Step 6: Selecting Deployment Environments
? Select environments: [Use arrows to move, space to select, enter to confirm]
> [x] development
  [x] staging
  [ ] production
✅ Selected environments: development, staging
```

### Step 4: Review and Confirm

After completing the configuration steps, you'll see a summary:

```
Deployment Summary:
==================
Application: myapp
Project: default
Repository: https://github.com/your-org/helm-charts
Chart: myapp (version 1.0.0)
Revision: main (branch)

Environments:
  - development (namespace: myapp-development)
  - staging (namespace: myapp-staging)

Do you want to proceed with this deployment? (y/N)
```

Type `y` to proceed or `n` to cancel.

### Step 5: Monitor Deployment

Once confirmed, kargo-bootstrap will:

1. Create namespaces for each environment
2. Generate ArgoCD Application manifests
3. Apply the manifests to your cluster
4. Monitor the sync status

Expected output:

```
🔄 Step 7: Creating namespaces for environments
Creating namespace 'myapp-development' for environment 'development'...
✅ Successfully created namespace 'myapp-development'
Creating namespace 'myapp-staging' for environment 'staging'...
✅ Successfully created namespace 'myapp-staging'

🔄 Step 8: Generating ArgoCD Application YAML
🔄 Generating ArgoCD Application for environment 'development'...
✅ Generated ArgoCD Application YAML for myapp-development (2048 bytes)
📝 Applying Application 'myapp-development' to namespace 'argocd'...
✅ Successfully created Application 'myapp-development'

🔄 Generating ArgoCD Application for environment 'staging'...
✅ Generated ArgoCD Application YAML for myapp-staging (2048 bytes)
📝 Applying Application 'myapp-staging' to namespace 'argocd'...
✅ Successfully created Application 'myapp-staging'

✅ Deployment configuration completed:
...
✅ Created namespaces:
  - myapp-development (environment: development)
  - myapp-staging (environment: staging)

✅ Generated ArgoCD Applications:
  - myapp-development
  - myapp-staging

🚀 Deployment completed successfully!
```

## Verifying Your Deployment

### Check ArgoCD Applications

```bash
kargo-bootstrap argocd list-apps
```

You should see your new applications:

```
✅ Found 2 ArgoCD Application(s)
NAME               NAMESPACE          PROJECT    HEALTH    SYNC    REVISION
----               --------          -------    ------    ----    --------
myapp-development  argocd             default    Healthy   Synced  main
myapp-staging      argocd             default    Healthy   Synced  main
```

### Check Kubernetes Resources

```bash
# Check namespaces
kubectl get namespaces | grep myapp

# Check pods in development
kubectl get pods -n myapp-development

# Check services
kubectl get svc -n myapp-development
```

### Check Application Status

```bash
kargo-bootstrap argocd status --name myapp-development
```

## Next Steps

Congratulations! You've successfully deployed your first application with kargo-bootstrap. Here are some next steps to explore:

1. **Explore Non-Interactive Mode**: Learn how to automate deployments
2. **Configure Custom Environments**: Set up your own environment configurations
3. **Advanced Git Operations**: Work with private repositories and SSH keys
4. **Monitoring Integration**: Set up observability with Uptrace/OTEL
5. **CI/CD Integration**: Integrate kargo-bootstrap into your pipelines

Continue to the [Examples](examples.md) guide to see more advanced usage patterns.
