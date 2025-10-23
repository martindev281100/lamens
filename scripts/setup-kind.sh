#!/bin/bash

# scripts/setup-kind.sh
# Script to create a Kind cluster with proper configuration for Argo CD testing

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Configuration
CLUSTER_NAME="kargo-test"
KUBECONFIG_PATH="${HOME}/.kube/config-kind-${CLUSTER_NAME}"
KIND_CONFIG_FILE="kind-config.yaml"

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

# Check if kind is installed
check_kind() {
    if ! command -v kind &> /dev/null; then
        log_error "kind is not installed. Please install kind first."
        echo "Visit: https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
        exit 1
    fi
    log_info "kind is installed"
}

# Check if kubectl is installed
check_kubectl() {
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed. Please install kubectl first."
        echo "Visit: https://kubernetes.io/docs/tasks/tools/"
        exit 1
    fi
    log_info "kubectl is installed"
}

# Create Kind configuration
create_kind_config() {
    log_info "Creating Kind configuration"
    cat > ${KIND_CONFIG_FILE} << EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: ${CLUSTER_NAME}
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
  - containerPort: 30080
    hostPort: 30080
    protocol: TCP
  - containerPort: 30443
    hostPort: 30443
    protocol: TCP
  - containerPort: 31080
    hostPort: 31080
    protocol: TCP
  - containerPort: 31443
    hostPort: 31443
    protocol: TCP
networking:
  apiServerAddress: "127.0.0.1"
  apiServerPort: 6443
  podSubnet: "10.244.0.0/16"
  serviceSubnet: "10.96.0.0/12"
EOF
    log_info "Kind configuration created at ${KIND_CONFIG_FILE}"
}

# Check if cluster already exists
check_cluster() {
    if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
        log_warn "Cluster '${CLUSTER_NAME}' already exists"
        read -p "Do you want to delete and recreate it? (y/N): " -n 1 -r
        echo
        if [[ $REPLY =~ ^[Yy]$ ]]; then
            log_info "Deleting existing cluster"
            kind delete cluster --name ${CLUSTER_NAME}
        else
            log_info "Using existing cluster"
            return 0
        fi
    fi
    return 1
}

# Create Kind cluster
create_cluster() {
    log_info "Creating Kind cluster '${CLUSTER_NAME}'"
    kind create cluster --config ${KIND_CONFIG_FILE} --name ${CLUSTER_NAME}
    
    # Update kubeconfig
    export KUBECONFIG=${KUBECONFIG_PATH}
    kind get kubeconfig --name ${CLUSTER_NAME} > ${KUBECONFIG_PATH}
    
    log_info "Cluster created successfully"
    log_info "Kubeconfig saved to: ${KUBECONFIG_PATH}"
    log_info "Use: export KUBECONFIG=${KUBECONFIG_PATH}"
}

# Install NGINX Ingress Controller
install_ingress() {
    log_info "Installing NGINX Ingress Controller"
    
    # Add necessary labels for ingress
    kubectl label nodes ${CLUSTER_NAME}-control-plane ingress-ready=true --overwrite
    
    # Install ingress controller
    kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
    
    # Wait for ingress controller to be ready
    log_info "Waiting for ingress controller to be ready..."
    kubectl wait --namespace ingress-nginx \
        --for=condition=ready pod \
        --selector=app.kubernetes.io/component=controller \
        --timeout=300s
    
    log_info "NGINX Ingress Controller installed successfully"
}

# Add additional configurations for Argo CD
configure_cluster() {
    log_info "Configuring cluster for Argo CD"
    
    # Create namespace for Argo CD
    kubectl create namespace argocd --dry-run=client -o yaml | kubectl apply -f -
    
    # Add necessary node labels and taints for testing
    kubectl label nodes ${CLUSTER_NAME}-control-plane environment=test --overwrite
    kubectl label nodes ${CLUSTER_NAME}-control-plane workload-type=generic --overwrite
    
    log_info "Cluster configured for Argo CD"
}

# Verify cluster setup
verify_setup() {
    log_info "Verifying cluster setup"
    
    # Check cluster nodes
    kubectl get nodes
    
    # Check cluster info
    kubectl cluster-info
    
    # Check ingress controller
    kubectl get pods -n ingress-nginx
    
    log_info "Cluster setup verified"
}

# Cleanup function
cleanup() {
    if [ -f "${KIND_CONFIG_FILE}" ]; then
        rm -f ${KIND_CONFIG_FILE}
        log_info "Cleaned up temporary files"
    fi
}

# Main execution
main() {
    log_info "Setting up Kind cluster for kargo-bootstrap testing"
    
    # Set up cleanup trap
    trap cleanup EXIT
    
    # Run checks
    check_kind
    check_kubectl
    
    # Create configuration
    create_kind_config
    
    # Check if cluster exists
    if ! check_cluster; then
        # Create new cluster
        create_cluster
    fi
    
    # Set KUBECONFIG for subsequent commands
    export KUBECONFIG=${KUBECONFIG_PATH}
    
    # Install ingress
    install_ingress
    
    # Configure cluster
    configure_cluster
    
    # Verify setup
    verify_setup
    
    log_info "Kind cluster setup completed successfully!"
    echo ""
    echo "To use this cluster, run:"
    echo "export KUBECONFIG=${KUBECONFIG_PATH}"
    echo ""
    echo "Cluster access:"
    echo "- API Server: https://127.0.0.1:6443"
    echo "- HTTP (Ingress): http://localhost"
    echo "- HTTPS (Ingress): https://localhost"
    echo "- Argo CD UI will be available at: http://localhost/argocd"
}

# Run main function
main "$@"