#!/bin/bash

# scripts/cleanup-env.sh
# Script to clean up the test environment for kargo-bootstrap testing

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
LOG_FILE="${PROJECT_ROOT}/cleanup-env.log"
CLUSTER_NAME="kargo-test"
REPO_NAME="kargo-test-repo"

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

# Initialize log file
init_log() {
    log_step "Initializing cleanup log file"
    
    cat > "${LOG_FILE}" << EOF
# Kargo Bootstrap Test Environment Cleanup Log
# Date: $(date)
# User: $(whoami)
# Host: $(hostname)

EOF
    
    log_info "Cleanup log file initialized: ${LOG_FILE}"
}

# Check if kind is installed
check_kind() {
    if ! command -v kind &> /dev/null; then
        log_error "kind is not installed. Cannot clean up Kind cluster."
        return 1
    fi
    return 0
}

# Check if kubectl is installed
check_kubectl() {
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed. Cannot clean up Kubernetes resources."
        return 1
    fi
    return 0
}

# Clean up Kind cluster
cleanup_kind_cluster() {
    log_step "Cleaning up Kind cluster"
    
    if ! check_kind; then
        log_warn "Skipping Kind cluster cleanup"
        return 0
    fi
    
    # Check if cluster exists
    if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
        log_info "Deleting Kind cluster: ${CLUSTER_NAME}"
        
        if kind delete cluster --name "${CLUSTER_NAME}" 2>&1 | tee -a "${LOG_FILE}"; then
            log_info "Kind cluster deleted successfully"
        else
            log_error "Failed to delete Kind cluster"
            return 1
        fi
    else
        log_info "Kind cluster '${CLUSTER_NAME}' does not exist"
    fi
    
    # Clean up kubeconfig
    if [ -f "${KUBECONFIG_PATH}" ]; then
        log_info "Removing kubeconfig file: ${KUBECONFIG_PATH}"
        rm -f "${KUBECONFIG_PATH}"
    fi
}

# Clean up test repository directory
cleanup_test_repository() {
    log_step "Cleaning up test repository"
    
    local repo_dir="${PROJECT_ROOT}/${REPO_NAME}"
    
    if [ -d "${repo_dir}" ]; then
        log_info "Removing test repository directory: ${repo_dir}"
        rm -rf "${repo_dir}"
    else
        log_info "Test repository directory does not exist: ${repo_dir}"
    fi
}

# Clean up build artifacts
cleanup_build_artifacts() {
    log_step "Cleaning up build artifacts"
    
    # Clean up binary
    if [ -f "${PROJECT_ROOT}/bin/kargo-bootstrap" ]; then
        log_info "Removing kargo-bootstrap binary"
        rm -f "${PROJECT_ROOT}/bin/kargo-bootstrap"
    fi
    
    # Clean up other build artifacts
    if [ -d "${PROJECT_ROOT}/bin" ]; then
        log_info "Cleaning up bin directory"
        find "${PROJECT_ROOT}/bin" -type f -name "*" -delete 2>/dev/null || true
    fi
}

# Clean up log files
cleanup_log_files() {
    log_step "Cleaning up log files"
    
    # Clean up setup log
    if [ -f "${PROJECT_ROOT}/setup-env.log" ]; then
        log_info "Removing setup log file"
        rm -f "${PROJECT_ROOT}/setup-env.log"
    fi
    
    # Clean up test logs
    if [ -d "${PROJECT_ROOT}/test-logs" ]; then
        log_info "Removing test logs directory"
        rm -rf "${PROJECT_ROOT}/test-logs"
    fi
    
    # Keep the current cleanup log for reference
    log_info "Keeping current cleanup log: ${LOG_FILE}"
}

# Clean up temporary files
cleanup_temp_files() {
    log_step "Cleaning up temporary files"
    
    # Clean up Kind config files
    find "${PROJECT_ROOT}" -name "kind-config*.yaml" -type f -delete 2>/dev/null || true
    
    # Clean up Argo CD values files
    find "${PROJECT_ROOT}" -name "argocd-values*.yaml" -type f -delete 2>/dev/null || true
    
    # Clean up Git server files
    find "${PROJECT_ROOT}" -name "git-server*.yaml" -type f -delete 2>/dev/null || true
    
    # Clean up test project files
    find "${PROJECT_ROOT}" -name "*-project.yaml" -type f -delete 2>/dev/null || true
    
    # Clean up test application files
    find "${PROJECT_ROOT}" -name "*-application.yaml" -type f -delete 2>/dev/null || true
    
    # Clean up test ingress files
    find "${PROJECT_ROOT}" -name "*-ingress.yaml" -type f -delete 2>/dev/null || true
    
    log_info "Temporary files cleaned up"
}

# Clean up Docker resources
cleanup_docker_resources() {
    log_step "Cleaning up Docker resources"
    
    # Check if Docker is running
    if ! docker info &> /dev/null; then
        log_warn "Docker is not running. Skipping Docker cleanup."
        return 0
    fi
    
    # Clean up Docker containers related to the test environment
    local containers=$(docker ps -a --filter "name=${CLUSTER_NAME}" --format "{{.ID}}" 2>/dev/null || true)
    
    if [ -n "$containers" ]; then
        log_info "Removing Docker containers related to ${CLUSTER_NAME}"
        echo "$containers" | xargs -r docker rm -f 2>&1 | tee -a "${LOG_FILE}" || true
    fi
    
    # Clean up Docker images related to the test environment
    local images=$(docker images --filter "reference=*${CLUSTER_NAME}*" --format "{{.ID}}" 2>/dev/null || true)
    
    if [ -n "$images" ]; then
        log_info "Removing Docker images related to ${CLUSTER_NAME}"
        echo "$images" | xargs -r docker rmi -f 2>&1 | tee -a "${LOG_FILE}" || true
    fi
    
    # Clean up Docker volumes related to the test environment
    local volumes=$(docker volume ls --filter "name=${CLUSTER_NAME}" --format "{{.Name}}" 2>/dev/null || true)
    
    if [ -n "$volumes" ]; then
        log_info "Removing Docker volumes related to ${CLUSTER_NAME}"
        echo "$volumes" | xargs -r docker volume rm -f 2>&1 | tee -a "${LOG_FILE}" || true
    fi
    
    log_info "Docker resources cleaned up"
}

# Clean up network resources
cleanup_network_resources() {
    log_step "Cleaning up network resources"
    
    # Remove /etc/hosts entries for test environment
    local hosts_file="/etc/hosts"
    local temp_hosts_file=$(mktemp)
    
    if [ -f "${hosts_file}" ]; then
        log_info "Removing test environment entries from ${hosts_file}"
        
        # Create a temporary file without test environment entries
        grep -v "argocd.local\|web-app.*.local\|api-service.*.local\|monitoring.local" "${hosts_file}" > "${temp_hosts_file}" 2>/dev/null || true
        
        # Replace the original hosts file if changes were made
        if [ -s "${temp_hosts_file}" ]; then
            if sudo cp "${temp_hosts_file}" "${hosts_file}" 2>/dev/null; then
                log_info "Updated ${hosts_file}"
            else
                log_warn "Could not update ${hosts_file}. You may need to remove test entries manually."
            fi
        fi
        
        rm -f "${temp_hosts_file}"
    fi
}

# Verify cleanup
verify_cleanup() {
    log_step "Verifying cleanup"
    
    local cleanup_success=true
    
    # Check if Kind cluster still exists
    if check_kind && kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
        log_error "Kind cluster '${CLUSTER_NAME}' still exists"
        cleanup_success=false
    else
        log_info "✓ Kind cluster '${CLUSTER_NAME}' is cleaned up"
    fi
    
    # Check if test repository still exists
    if [ -d "${PROJECT_ROOT}/${REPO_NAME}" ]; then
        log_error "Test repository directory still exists: ${PROJECT_ROOT}/${REPO_NAME}"
        cleanup_success=false
    else
        log_info "✓ Test repository is cleaned up"
    fi
    
    # Check if kubeconfig still exists
    if [ -f "${KUBECONFIG_PATH}" ]; then
        log_error "Kubeconfig file still exists: ${KUBECONFIG_PATH}"
        cleanup_success=false
    else
        log_info "✓ Kubeconfig is cleaned up"
    fi
    
    # Check if binary still exists
    if [ -f "${PROJECT_ROOT}/bin/kargo-bootstrap" ]; then
        log_error "kargo-bootstrap binary still exists"
        cleanup_success=false
    else
        log_info "✓ Build artifacts are cleaned up"
    fi
    
    if [ "$cleanup_success" = true ]; then
        log_info "Cleanup verification completed successfully"
        return 0
    else
        log_error "Cleanup verification failed"
        return 1
    fi
}

# Print cleanup summary
print_cleanup_summary() {
    log_step "Printing cleanup summary"
    
    echo ""
    echo "=========================================="
    echo "Kargo Bootstrap Test Environment Cleanup"
    echo "=========================================="
    echo ""
    echo "Cleanup Summary:"
    echo "================"
    echo "Kind Cluster: ${CLUSTER_NAME} - Deleted"
    echo "Kubeconfig: ${KUBECONFIG_PATH} - Removed"
    echo "Test Repository: ${PROJECT_ROOT}/${REPO_NAME} - Removed"
    echo "Build Artifacts: Cleaned up"
    echo "Log Files: Cleaned up (except current cleanup log)"
    echo "Temporary Files: Cleaned up"
    echo "Docker Resources: Cleaned up"
    echo "Network Resources: Cleaned up"
    echo ""
    echo "Log File: ${LOG_FILE}"
    echo ""
    echo "To set up the test environment again, run:"
    echo "make setup-test-env"
    echo "or"
    echo "${SCRIPT_DIR}/setup-env.sh"
    echo ""
}

# Cleanup function for interrupted cleanup
cleanup_on_interrupt() {
    log_warn "Cleanup interrupted. Partial cleanup may have occurred."
    log_info "Check the log file for details: ${LOG_FILE}"
    exit 1
}

# Main execution
main() {
    log_info "Starting kargo-bootstrap test environment cleanup"
    
    # Set up trap for cleanup on interrupt
    trap cleanup_on_interrupt INT
    
    # Initialize log
    init_log
    
    # Clean up Kind cluster
    cleanup_kind_cluster
    
    # Clean up test repository
    cleanup_test_repository
    
    # Clean up build artifacts
    cleanup_build_artifacts
    
    # Clean up log files
    cleanup_log_files
    
    # Clean up temporary files
    cleanup_temp_files
    
    # Clean up Docker resources
    cleanup_docker_resources
    
    # Clean up network resources
    cleanup_network_resources
    
    # Verify cleanup
    if verify_cleanup; then
        # Print cleanup summary
        print_cleanup_summary
        
        log_info "Test environment cleanup completed successfully!"
    else
        log_error "Test environment cleanup completed with errors"
        exit 1
    fi
}

# Handle command line arguments
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [OPTIONS]"
        echo ""
        echo "Options:"
        echo "  --help, -h    Show this help message"
        echo ""
        echo "This script cleans up the test environment for kargo-bootstrap testing,"
        echo "including the Kind cluster, test repository, build artifacts, and other resources."
        exit 0
        ;;
    --force)
        log_info "Running cleanup in force mode"
        main
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