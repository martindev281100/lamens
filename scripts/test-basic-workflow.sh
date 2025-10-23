#!/bin/bash

# scripts/test-basic-workflow.sh
# Script to test basic deployment workflow for kargo-bootstrap

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
LOG_FILE="${PROJECT_ROOT}/test-basic-workflow.log"
TEST_NAMESPACE="test-basic"
GIT_REPO_URL="ssh://git@git-server.default.svc.cluster.local:30888/kargo-test-repo.git"
CHART_PATH="charts/web-app"
ENVIRONMENT="dev"

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
    log_step "Initializing test log file"
    
    cat > "${LOG_FILE}" << EOF
# Kargo Bootstrap Basic Workflow Test Log
# Date: $(date)
# User: $(whoami)
# Host: $(hostname)

EOF
    
    log_info "Test log file initialized: ${LOG_FILE}"
}

# Check prerequisites
check_prerequisites() {
    log_step "Checking prerequisites"
    
    # Check if kubectl is installed
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed"
        exit 1
    fi
    
    # Check if kubeconfig exists
    if [ ! -f "${KUBECONFIG_PATH}" ]; then
        log_error "Kubeconfig not found: ${KUBECONFIG_PATH}"
        exit 1
    fi
    
    # Set KUBECONFIG
    export KUBECONFIG="${KUBECONFIG_PATH}"
    
    # Check if cluster is accessible
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot access Kubernetes cluster"
        exit 1
    fi
    
    # Check if kargo-bootstrap binary exists
    if [ ! -f "${PROJECT_ROOT}/bin/kargo-bootstrap" ]; then
        log_error "kargo-bootstrap binary not found"
        exit 1
    fi
    
    # Check if Argo CD is installed
    if ! kubectl get namespace argocd &> /dev/null; then
        log_error "Argo CD namespace not found"
        exit 1
    fi
    
    # Check if Git server is running
    if ! kubectl get deployment git-server -n default &> /dev/null; then
        log_error "Git server deployment not found"
        exit 1
    fi
    
    log_info "All prerequisites are satisfied"
}

# Create test namespace
create_test_namespace() {
    log_step "Creating test namespace: ${TEST_NAMESPACE}"
    
    # Delete namespace if it exists
    if kubectl get namespace "${TEST_NAMESPACE}" &> /dev/null; then
        log_warn "Namespace ${TEST_NAMESPACE} already exists, deleting it"
        kubectl delete namespace "${TEST_NAMESPACE}" --timeout=60s
    fi
    
    # Create namespace
    kubectl create namespace "${TEST_NAMESPACE}"
    
    log_info "Test namespace created: ${TEST_NAMESPACE}"
}

# Test kargo-bootstrap deploy command
test_deploy_command() {
    log_step "Testing kargo-bootstrap deploy command"
    
    # Run kargo-bootstrap deploy
    cd "${PROJECT_ROOT}"
    
    log_info "Running: ./bin/kargo-bootstrap deploy --repo ${GIT_REPO_URL} --chart ${CHART_PATH} --env ${ENVIRONMENT} --namespace ${TEST_NAMESPACE}"
    
    if ./bin/kargo-bootstrap deploy --repo "${GIT_REPO_URL}" --chart "${CHART_PATH}" --env "${ENVIRONMENT}" --namespace "${TEST_NAMESPACE}" 2>&1 | tee -a "${LOG_FILE}"; then
        log_info "kargo-bootstrap deploy command executed successfully"
    else
        log_error "kargo-bootstrap deploy command failed"
        exit 1
    fi
}

# Verify deployment in Argo CD
verify_argocd_deployment() {
    log_step "Verifying deployment in Argo CD"
    
    # Get application name
    local app_name=$(kubectl get applications -n argocd -o jsonpath='{.items[?(@.spec.destination.namespace=="'${TEST_NAMESPACE}'")].metadata.name}' 2>/dev/null || true)
    
    if [ -z "$app_name" ]; then
        log_error "No Argo CD application found for namespace ${TEST_NAMESPACE}"
        exit 1
    fi
    
    log_info "Found Argo CD application: $app_name"
    
    # Wait for application to be synced
    log_info "Waiting for application to be synced..."
    
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        local sync_status=$(kubectl get application "$app_name" -n argocd -o jsonpath='{.status.sync.status}' 2>/dev/null || true)
        local health_status=$(kubectl get application "$app_name" -n argocd -o jsonpath='{.status.health.status}' 2>/dev/null || true)
        
        log_info "Attempt $((attempt+1)): Sync status: $sync_status, Health status: $health_status"
        
        if [ "$sync_status" = "Synced" ] && [ "$health_status" = "Healthy" ]; then
            log_info "Application is synced and healthy"
            break
        fi
        
        sleep 10
        attempt=$((attempt+1))
    done
    
    if [ $attempt -eq $max_attempts ]; then
        log_error "Application did not become synced and healthy within expected time"
        exit 1
    fi
}

# Verify deployment in Kubernetes
verify_kubernetes_deployment() {
    log_step "Verifying deployment in Kubernetes"
    
    # Check if deployment exists
    local deployment_name=$(kubectl get deployments -n "${TEST_NAMESPACE}" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
    
    if [ -z "$deployment_name" ]; then
        log_error "No deployment found in namespace ${TEST_NAMESPACE}"
        exit 1
    fi
    
    log_info "Found deployment: $deployment_name"
    
    # Wait for deployment to be ready
    log_info "Waiting for deployment to be ready..."
    
    if kubectl rollout status deployment/"$deployment_name" -n "${TEST_NAMESPACE}" --timeout=300s; then
        log_info "Deployment is ready"
    else
        log_error "Deployment did not become ready within expected time"
        exit 1
    fi
    
    # Check pods
    local pod_count=$(kubectl get pods -n "${TEST_NAMESPACE}" --field-selector=status.phase=Running --no-headers | wc -l)
    
    if [ "$pod_count" -gt 0 ]; then
        log_info "Found $pod_count running pods in namespace ${TEST_NAMESPACE}"
    else
        log_error "No running pods found in namespace ${TEST_NAMESPACE}"
        exit 1
    fi
    
    # Check services
    local service_count=$(kubectl get services -n "${TEST_NAMESPACE}" --no-headers | wc -l)
    
    if [ "$service_count" -gt 0 ]; then
        log_info "Found $service_count services in namespace ${TEST_NAMESPACE}"
    else
        log_error "No services found in namespace ${TEST_NAMESPACE}"
        exit 1
    fi
}

# Test kargo-bootstrap status command
test_status_command() {
    log_step "Testing kargo-bootstrap status command"
    
    # Run kargo-bootstrap status
    cd "${PROJECT_ROOT}"
    
    log_info "Running: ./bin/kargo-bootstrap status --namespace ${TEST_NAMESPACE}"
    
    if ./bin/kargo-bootstrap status --namespace "${TEST_NAMESPACE}" 2>&1 | tee -a "${LOG_FILE}"; then
        log_info "kargo-bootstrap status command executed successfully"
    else
        log_error "kargo-bootstrap status command failed"
        exit 1
    fi
}

# Test kargo-bootstrap render command
test_render_command() {
    log_step "Testing kargo-bootstrap render command"
    
    # Run kargo-bootstrap render
    cd "${PROJECT_ROOT}"
    
    log_info "Running: ./bin/kargo-bootstrap render --repo ${GIT_REPO_URL} --chart ${CHART_PATH} --env ${ENVIRONMENT}"
    
    if ./bin/kargo-bootstrap render --repo "${GIT_REPO_URL}" --chart "${CHART_PATH}" --env "${ENVIRONMENT}" 2>&1 | tee -a "${LOG_FILE}"; then
        log_info "kargo-bootstrap render command executed successfully"
    else
        log_error "kargo-bootstrap render command failed"
        exit 1
    fi
}

# Clean up test resources
cleanup_test_resources() {
    log_step "Cleaning up test resources"
    
    # Delete Argo CD application
    local app_name=$(kubectl get applications -n argocd -o jsonpath='{.items[?(@.spec.destination.namespace=="'${TEST_NAMESPACE}'")].metadata.name}' 2>/dev/null || true)
    
    if [ -n "$app_name" ]; then
        log_info "Deleting Argo CD application: $app_name"
        kubectl delete application "$app_name" -n argocd --ignore-not-found=true
    fi
    
    # Delete test namespace
    if kubectl get namespace "${TEST_NAMESPACE}" &> /dev/null; then
        log_info "Deleting test namespace: ${TEST_NAMESPACE}"
        kubectl delete namespace "${TEST_NAMESPACE}" --ignore-not-found=true --timeout=60s
    fi
    
    log_info "Test resources cleaned up"
}

# Print test summary
print_test_summary() {
    log_step "Printing test summary"
    
    echo ""
    echo "=========================================="
    echo "Kargo Bootstrap Basic Workflow Test"
    echo "=========================================="
    echo ""
    echo "Test Summary:"
    echo "============="
    echo "Test Namespace: ${TEST_NAMESPACE}"
    echo "Git Repository: ${GIT_REPO_URL}"
    echo "Chart Path: ${CHART_PATH}"
    echo "Environment: ${ENVIRONMENT}"
    echo ""
    echo "Test Results:"
    echo "============="
    echo "✓ Prerequisites check: Passed"
    echo "✓ Test namespace creation: Passed"
    echo "✓ kargo-bootstrap deploy command: Passed"
    echo "✓ Argo CD deployment verification: Passed"
    echo "✓ Kubernetes deployment verification: Passed"
    echo "✓ kargo-bootstrap status command: Passed"
    echo "✓ kargo-bootstrap render command: Passed"
    echo "✓ Test resources cleanup: Passed"
    echo ""
    echo "Log File: ${LOG_FILE}"
    echo ""
    echo "Test completed successfully!"
    echo ""
}

# Cleanup function for interrupted test
cleanup_on_interrupt() {
    log_warn "Test interrupted. Cleaning up..."
    cleanup_test_resources
    log_info "Cleanup completed. Check the log file for details: ${LOG_FILE}"
    exit 1
}

# Main execution
main() {
    log_info "Starting kargo-bootstrap basic workflow test"
    
    # Set up trap for cleanup on interrupt
    trap cleanup_on_interrupt INT
    
    # Initialize log
    init_log
    
    # Check prerequisites
    check_prerequisites
    
    # Create test namespace
    create_test_namespace
    
    # Test kargo-bootstrap deploy command
    test_deploy_command
    
    # Verify deployment in Argo CD
    verify_argocd_deployment
    
    # Verify deployment in Kubernetes
    verify_kubernetes_deployment
    
    # Test kargo-bootstrap status command
    test_status_command
    
    # Test kargo-bootstrap render command
    test_render_command
    
    # Clean up test resources
    cleanup_test_resources
    
    # Print test summary
    print_test_summary
    
    log_info "Basic workflow test completed successfully!"
}

# Handle command line arguments
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [OPTIONS]"
        echo ""
        echo "Options:"
        echo "  --help, -h    Show this help message"
        echo ""
        echo "This script tests the basic deployment workflow for kargo-bootstrap,"
        echo "including deployment, status checking, and rendering."
        exit 0
        ;;
    --namespace)
        TEST_NAMESPACE="${2:-test-basic}"
        log_info "Using custom namespace: ${TEST_NAMESPACE}"
        main
        ;;
    --chart)
        CHART_PATH="${2:-charts/web-app}"
        log_info "Using custom chart path: ${CHART_PATH}"
        main
        ;;
    --env)
        ENVIRONMENT="${2:-dev}"
        log_info "Using custom environment: ${ENVIRONMENT}"
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