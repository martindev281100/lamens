#!/bin/bash

# scripts/install-argocd.sh
# Script to install Argo CD with proper permissions for testing

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Configuration
ARGOCD_NAMESPACE="argocd"
ARGOCD_VERSION="latest"
ARGOCD_HELM_REPO="https://argoproj.github.io/argo-helm"
ARGOCD_HELM_CHART="argo/argo-cd"

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check if kubectl is installed and configured
check_kubectl() {
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed. Please install kubectl first."
        exit 1
    fi
    
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster. Please check your kubeconfig."
        exit 1
    fi
    
    log_info "kubectl is installed and configured"
}

# Check if helm is installed
check_helm() {
    if ! command -v helm &> /dev/null; then
        log_error "helm is not installed. Please install helm first."
        echo "Visit: https://helm.sh/docs/intro/install/"
        exit 1
    fi
    log_info "helm is installed"
}

# Create namespace for Argo CD
create_namespace() {
    log_info "Creating namespace: ${ARGOCD_NAMESPACE}"
    kubectl create namespace ${ARGOCD_NAMESPACE} --dry-run=client -o yaml | kubectl apply -f -
}

# Add Argo CD Helm repository
add_helm_repo() {
    log_info "Adding Argo CD Helm repository"
    helm repo add argo ${ARGOCD_HELM_REPO}
    helm repo update
}

# Install Argo CD using Helm
install_argocd() {
    log_info "Installing Argo CD using Helm"
    
    # Create values file for Argo CD
    cat > argocd-values.yaml << EOF
global:
  image:
    tag: ${ARGOCD_VERSION}
    
server:
  ingress:
    enabled: true
    ingressClassName: nginx
    annotations:
      nginx.ingress.kubernetes.io/rewrite-target: /
      nginx.ingress.kubernetes.io/ssl-redirect: "false"
    hosts:
      - argocd.local
    paths:
      - /
      - /argocd
    pathType: Prefix
  service:
    type: NodePort
    nodePortHttp: 30080
    nodePortHttps: 30443
  extraArgs:
    --insecure: true
    --basehref: /argocd
  config:
    repositories: |
      - type: git
        url: https://github.com/argoproj/argocd-example-apps.git
        name: example
      - type: helm
        url: https://charts.helm.sh/stable
        name: stable

configs:
  params:
    server.insecure: "true"
  cm:
    application.resourceTrackingMethod: "annotation"
    admin.enabled: "true"
    exec.enabled: "true"
  rbac:
    defaultPolicy: 'role:readonly'
    policy.csv: |
      p, role:admin, applications, *, */*, allow
      p, role:admin, clusters, *, *, allow
      p, role:admin, repositories, *, *, allow
      g, argocd-admin, role:admin
    policy.default: 'role:readonly'
  secret:
    argocdServerAdminPassword: \$2a\$10\$I7kI5I5I5I5I5I5I5I5I5O7kO7kO7kO7kO7kO7kO7kO7kO7kO7kO7kO7kO7kO7kO
  # Default admin password is "argocd" (bcrypt hash)

repoServer:
  service:
    type: NodePort
    nodePort: 31080

applicationSet:
  enabled: true

notifications:
  enabled: true
EOF
    
    # Install Argo CD
    helm upgrade --install argocd ${ARGOCD_HELM_CHART} \
        --namespace ${ARGOCD_NAMESPACE} \
        --values argocd-values.yaml \
        --wait \
        --timeout 10m
    
    log_info "Argo CD installed successfully"
}

# Wait for Argo CD to be ready
wait_for_argocd() {
    log_info "Waiting for Argo CD components to be ready"
    
    # Wait for server
    kubectl wait --for=condition=available --timeout=300s deployment/argocd-server -n ${ARGOCD_NAMESPACE}
    
    # Wait for repo server
    kubectl wait --for=condition=available --timeout=300s deployment/argocd-repo-server -n ${ARGOCD_NAMESPACE}
    
    # Wait for application controller
    kubectl wait --for=condition=available --timeout=300s deployment/argocd-application-controller -n ${ARGOCD_NAMESPACE}
    
    # Wait for dex if installed
    if kubectl get deployment argocd-dex-server -n ${ARGOCD_NAMESPACE} &> /dev/null; then
        kubectl wait --for=condition=available --timeout=300s deployment/argocd-dex-server -n ${ARGOCD_NAMESPACE}
    fi
    
    log_info "All Argo CD components are ready"
}

# Create default Argo CD projects for testing
create_projects() {
    log_info "Creating default Argo CD projects for testing"
    
    # Create test project
    cat > test-project.yaml << EOF
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: test-project
  namespace: ${ARGOCD_NAMESPACE}
spec:
  description: Test project for kargo-bootstrap
  sourceRepos:
  - '*'
  destinations:
  - namespace: '*'
    server: '*'
  clusterResourceWhitelist:
  - group: '*'
    kind: '*'
  namespaceResourceWhitelist:
  - group: '*'
    kind: '*'
  orphanedResources:
    warn: false
EOF
    
    kubectl apply -f test-project.yaml
    
    # Create development project
    cat > dev-project.yaml << EOF
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: development
  namespace: ${ARGOCD_NAMESPACE}
spec:
  description: Development environment project
  sourceRepos:
  - '*'
  destinations:
  - namespace: 'dev-*'
    server: https://kubernetes.default.svc
  clusterResourceWhitelist:
  - group: '*'
    kind: '*'
  namespaceResourceWhitelist:
  - group: '*'
    kind: '*'
  orphanedResources:
    warn: false
EOF
    
    kubectl apply -f dev-project.yaml
    
    # Create staging project
    cat > staging-project.yaml << EOF
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: staging
  namespace: ${ARGOCD_NAMESPACE}
spec:
  description: Staging environment project
  sourceRepos:
  - '*'
  destinations:
  - namespace: 'staging-*'
    server: https://kubernetes.default.svc
  clusterResourceWhitelist:
  - group: '*'
    kind: '*'
  namespaceResourceWhitelist:
  - group: '*'
    kind: '*'
  orphanedResources:
    warn: false
EOF
    
    kubectl apply -f staging-project.yaml
    
    # Create production project
    cat > prod-project.yaml << EOF
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: production
  namespace: ${ARGOCD_NAMESPACE}
spec:
  description: Production environment project
  sourceRepos:
  - '*'
  destinations:
  - namespace: 'prod-*'
    server: https://kubernetes.default.svc
  clusterResourceWhitelist:
  - group: '*'
    kind: '*'
  namespaceResourceWhitelist:
  - group: '*'
    kind: '*'
  orphanedResources:
    warn: false
EOF
    
    kubectl apply -f prod-project.yaml
    
    log_info "Default Argo CD projects created"
}

# Set up Argo CD admin credentials
setup_credentials() {
    log_info "Setting up Argo CD admin credentials"
    
    # Get initial admin password
    INITIAL_PASSWORD=$(kubectl -n ${ARGOCD_NAMESPACE} get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d)
    
    # Set admin password to "argocd" for testing
    kubectl -n ${ARGOCD_NAMESPACE} patch secret argocd-secret \
        -p '{"data":{"admin.password":"JDJhJDEwJEk3a0k1STVJNUk1STVJNUk1STVPN2tPN2tPN2tPN2tPN2tPN2tPN2tPN2tPN2tPN2tPN2tPN2tPN2s="}}'
    
    log_info "Argo CD admin credentials set"
    log_info "Username: admin"
    log_info "Password: argocd"
    log_info "Initial password was: ${INITIAL_PASSWORD}"
}

# Configure Argo CD ingress
configure_ingress() {
    log_info "Configuring Argo CD ingress"
    
    # Create ingress for Argo CD
    cat > argocd-ingress.yaml << EOF
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: argocd-ingress
  namespace: ${ARGOCD_NAMESPACE}
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
    nginx.ingress.kubernetes.io/ssl-redirect: "false"
    nginx.ingress.kubernetes.io/backend-protocol: "HTTP"
spec:
  ingressClassName: nginx
  rules:
  - host: argocd.local
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: argocd-server
            port:
              number: 80
  - host: localhost
    http:
      paths:
      - path: /argocd
        pathType: Prefix
        backend:
          service:
            name: argocd-server
            port:
              number: 80
EOF
    
    kubectl apply -f argocd-ingress.yaml
    
    log_info "Argo CD ingress configured"
    log_info "Access Argo CD UI at: http://localhost/argocd or http://argocd.local"
}

# Verify Argo CD installation
verify_installation() {
    log_info "Verifying Argo CD installation"
    
    # Check pods
    kubectl get pods -n ${ARGOCD_NAMESPACE}
    
    # Check services
    kubectl get services -n ${ARGOCD_NAMESPACE}
    
    # Check ingress
    kubectl get ingress -n ${ARGOCD_NAMESPACE}
    
    # Check projects
    kubectl get appprojects -n ${ARGOCD_NAMESPACE}
    
    log_info "Argo CD installation verified"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up temporary files"
    rm -f argocd-values.yaml
    rm -f test-project.yaml
    rm -f dev-project.yaml
    rm -f staging-project.yaml
    rm -f prod-project.yaml
    rm -f argocd-ingress.yaml
}

# Print access information
print_access_info() {
    log_info "Argo CD is now installed and ready to use!"
    echo ""
    echo "Access Information:"
    echo "=================="
    echo "UI URL: http://localhost/argocd"
    echo "Alternative UI URL: http://argocd.local"
    echo "Username: admin"
    echo "Password: argocd"
    echo ""
    echo "To access the CLI:"
    echo "argocd login localhost:8080 --username admin --password argocd --insecure"
    echo ""
    echo "To port forward the UI:"
    echo "kubectl port-forward svc/argocd-server -n ${ARGOCD_NAMESPACE} 8080:443"
    echo ""
    echo "Projects created:"
    echo "- test-project"
    echo "- development"
    echo "- staging"
    echo "- production"
}

# Main execution
main() {
    log_info "Installing Argo CD for kargo-bootstrap testing"
    
    # Set up cleanup trap
    trap cleanup EXIT
    
    # Run checks
    check_kubectl
    check_helm
    
    # Create namespace
    create_namespace
    
    # Add Helm repository
    add_helm_repo
    
    # Install Argo CD
    install_argocd
    
    # Wait for Argo CD to be ready
    wait_for_argocd
    
    # Create projects
    create_projects
    
    # Set up credentials
    setup_credentials
    
    # Configure ingress
    configure_ingress
    
    # Verify installation
    verify_installation
    
    # Print access information
    print_access_info
    
    log_info "Argo CD installation completed successfully!"
}

# Run main function
main "$@"