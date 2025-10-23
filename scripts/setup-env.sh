#!/bin/bash

# scripts/setup-env.sh
# Script to set up the complete test environment for kargo-bootstrap testing

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "${SCRIPT_DIR}")"
KUBECONFIG_PATH="${HOME}/.kube/config-kind-kargo-test"
LOG_FILE="${PROJECT_ROOT}/setup-env.log"

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1" | tee -a "${LOG_FILE}"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1" | tee -a "${LOG_FILE}"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" | tee -a "${LOG_FILE}"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1" | tee -a "${LOG_FILE}"
}

# Check prerequisites
check_prerequisites() {
    log_step "Checking prerequisites"
    
    local missing_tools=()
    
    if ! command -v docker &> /dev/null; then
        missing_tools+=("docker")
    fi
    
    if ! command -v kind &> /dev/null; then
        missing_tools+=("kind")
    fi
    
    if ! command -v kubectl &> /dev/null; then
        missing_tools+=("kubectl")
    fi
    
    if ! command -v helm &> /dev/null; then
        missing_tools+=("helm")
    fi
    
    if ! command -v git &> /dev/null; then
        missing_tools+=("git")
    fi
    
    if [ ${#missing_tools[@]} -ne 0 ]; then
        log_error "Missing required tools: ${missing_tools[*]}"
        echo ""
        echo "Please install the missing tools:"
        for tool in "${missing_tools[@]}"; do
            case $tool in
                docker)
                    echo "  - Docker: https://docs.docker.com/get-docker/"
                    ;;
                kind)
                    echo "  - Kind: https://kind.sigs.k8s.io/docs/user/quick-start/#installation"
                    ;;
                kubectl)
                    echo "  - kubectl: https://kubernetes.io/docs/tasks/tools/"
                    ;;
                helm)
                    echo "  - Helm: https://helm.sh/docs/intro/install/"
                    ;;
                git)
                    echo "  - Git: https://git-scm.com/book/en/v2/Getting-Started-Installing-Git"
                    ;;
            esac
        done
        exit 1
    fi
    
    log_info "All prerequisites are satisfied"
}

# Check Docker daemon
check_docker() {
    log_step "Checking Docker daemon"
    
    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running. Please start Docker."
        exit 1
    fi
    
    log_info "Docker daemon is running"
}

# Initialize log file
init_log() {
    log_step "Initializing log file"
    
    cat > "${LOG_FILE}" << EOF
# Kargo Bootstrap Test Environment Setup Log
# Date: $(date)
# User: $(whoami)
# Host: $(hostname)

EOF
    
    log_info "Log file initialized: ${LOG_FILE}"
}

# Set up Kind cluster
setup_kind_cluster() {
    log_step "Setting up Kind cluster"
    
    if [ -f "${SCRIPT_DIR}/setup-kind.sh" ]; then
        "${SCRIPT_DIR}/setup-kind.sh" 2>&1 | tee -a "${LOG_FILE}"
        log_info "Kind cluster setup completed"
    else
        log_error "setup-kind.sh script not found"
        exit 1
    fi
}

# Install Argo CD
install_argocd() {
    log_step "Installing Argo CD"
    
    # Set KUBECONFIG for Argo CD installation
    export KUBECONFIG="${KUBECONFIG_PATH}"
    
    if [ -f "${SCRIPT_DIR}/install-argocd.sh" ]; then
        "${SCRIPT_DIR}/install-argocd.sh" 2>&1 | tee -a "${LOG_FILE}"
        log_info "Argo CD installation completed"
    else
        log_error "install-argocd.sh script not found"
        exit 1
    fi
}

# Set up test repository
setup_test_repository() {
    log_step "Setting up test repository"
    
    # Set KUBECONFIG for test repository setup
    export KUBECONFIG="${KUBECONFIG_PATH}"
    
    if [ -f "${SCRIPT_DIR}/setup-test-repo.sh" ]; then
        "${SCRIPT_DIR}/setup-test-repo.sh" 2>&1 | tee -a "${LOG_FILE}"
        log_info "Test repository setup completed"
    else
        log_error "setup-test-repo.sh script not found"
        exit 1
    fi
}

# Build kargo-bootstrap binary
build_kargo_bootstrap() {
    log_step "Building kargo-bootstrap binary"
    
    cd "${PROJECT_ROOT}"
    
    if make build 2>&1 | tee -a "${LOG_FILE}"; then
        log_info "kargo-bootstrap binary built successfully"
    else
        log_error "Failed to build kargo-bootstrap binary"
        exit 1
    fi
}

# Verify environment setup
verify_setup() {
    log_step "Verifying environment setup"
    
    # Set KUBECONFIG for verification
    export KUBECONFIG="${KUBECONFIG_PATH}"
    
    # Check Kind cluster
    if kind get clusters | grep -q "kargo-test"; then
        log_info "✓ Kind cluster 'kargo-test' is running"
    else
        log_error "✗ Kind cluster 'kargo-test' is not running"
        return 1
    fi
    
    # Check Argo CD
    if kubectl get pods -n argocd | grep -q "argocd-server"; then
        log_info "✓ Argo CD is installed"
    else
        log_error "✗ Argo CD is not installed"
        return 1
    fi
    
    # Check Git server
    if kubectl get pods -n default | grep -q "git-server"; then
        log_info "✓ Git server is running"
    else
        log_error "✗ Git server is not running"
        return 1
    fi
    
    # Check kargo-bootstrap binary
    if [ -f "${PROJECT_ROOT}/bin/kargo-bootstrap" ]; then
        log_info "✓ kargo-bootstrap binary exists"
    else
        log_error "✗ kargo-bootstrap binary not found"
        return 1
    fi
    
    log_info "Environment setup verification completed successfully"
}

# Print environment information
print_env_info() {
    log_step "Printing environment information"
    
    echo ""
    echo "=================================================="
    echo "Kargo Bootstrap Test Environment Setup Completed!"
    echo "=================================================="
    echo ""
    echo "Environment Information:"
    echo "======================="
    echo "Kind Cluster: kargo-test"
    echo "Kubeconfig: ${KUBECONFIG_PATH}"
    echo "Log File: ${LOG_FILE}"
    echo ""
    echo "Access Information:"
    echo "=================="
    echo "To use the cluster, run:"
    echo "export KUBECONFIG=${KUBECONFIG_PATH}"
    echo ""
    echo "Argo CD UI:"
    echo "URL: http://localhost/argocd"
    echo "Username: admin"
    echo "Password: argocd"
    echo ""
    echo "Git Server:"
    echo "SSH URL: ssh://git@git-server.default.svc.cluster.local:30888/kargo-test-repo.git"
    echo "HTTP URL: http://git-server.default.svc.cluster.local:30889/kargo-test-repo.git"
    echo ""
    echo "Test Repository:"
    echo "Location: ${PROJECT_ROOT}/kargo-test-repo"
    echo "Charts: web-app, api-service, database, monitoring"
    echo "Environments: dev, staging, prod"
    echo ""
    echo "Kargo Bootstrap Binary:"
    echo "Location: ${PROJECT_ROOT}/bin/kargo-bootstrap"
    echo ""
    echo "Next Steps:"
    echo "==========="
    echo "1. Set up your kubeconfig:"
    echo "   export KUBECONFIG=${KUBECONFIG_PATH}"
    echo ""
    echo "2. Test the environment by running:"
    echo "   make test-e2e"
    echo ""
    echo "3. Or run individual test scenarios:"
    echo "   ${SCRIPT_DIR}/test-basic-workflow.sh"
    echo "   ${SCRIPT_DIR}/test-multi-env.sh"
    echo "   ${SCRIPT_DIR}/test-error-scenarios.sh"
    echo ""
    echo "4. Access Argo CD UI to monitor deployments:"
    echo "   http://localhost/argocd"
    echo ""
    echo "5. Clean up the environment when done:"
    echo "   make cleanup-test-env"
    echo ""
    echo "For more information, see:"
    echo "- ${PROJECT_ROOT}/docs/test-environment.md"
    echo "- ${LOG_FILE}"
    echo ""
}

# Cleanup function for interrupted setup
cleanup_on_interrupt() {
    log_warn "Setup interrupted. Cleaning up..."
    
    # Unset KUBECONFIG
    unset KUBECONFIG
    
    log_info "Cleanup completed. Check the log file for details: ${LOG_FILE}"
    exit 1
}

# Main execution
main() {
    log_info "Starting kargo-bootstrap test environment setup"
    
    # Set up trap for cleanup on interrupt
    trap cleanup_on_interrupt INT
    
    # Initialize log
    init_log
    
    # Check prerequisites
    check_prerequisites
    
    # Check Docker daemon
    check_docker
    
    # Set up Kind cluster
    setup_kind_cluster
    
    # Install Argo CD
    install_argocd
    
    # Set up test repository
    setup_test_repository
    
    # Build kargo-bootstrap binary
    build_kargo_bootstrap
    
    # Verify setup
    verify_setup
    
    # Print environment information
    print_env_info
    
    log_info "Test environment setup completed successfully!"
}

# Handle command line arguments
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [OPTIONS]"
        echo ""
        echo "Options:"
        echo "  --help, -h    Show this help message"
        echo ""
        echo "This script sets up a complete test environment for kargo-bootstrap testing,"
        echo "including a Kind cluster, Argo CD, and a test Git repository."
        exit 0
        ;;
    "")
        # No arguments, run main function
        main
        ;;
    *)
        log_error "Unknown option: $1"
        echo "Use --help for usage information."
        exit 1
        ;;
esac