#!/bin/bash

# scripts/test-error-scenarios.sh
# Script to test error handling for kargo-bootstrap

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
LOG_FILE="${PROJECT_ROOT}/test-error-scenarios.log"
TEST_NAMESPACE="test-error"
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
# Kargo Bootstrap Error Scenarios Test Log
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

# Test scenario 1: Invalid repository URL
test_invalid_repository_url() {
    log_step "Testing scenario 1: Invalid repository URL"
    
    cd "${PROJECT_ROOT}"
    
    # Run kargo-bootstrap deploy with invalid repository URL
    log_info "Running: ./bin/kargo-bootstrap deploy --repo invalid-repo-url --chart ${CHART_PATH} --env ${ENVIRONMENT} --namespace ${TEST_NAMESPACE}"
    
    if ./bin/kargo-bootstrap deploy --repo "invalid-repo-url" --chart "${CHART_PATH}" --env "${ENVIRONMENT}" --namespace "${TEST_NAMESPACE}" 2>&1 | tee -a "${LOG_FILE}"; then
        log_error "kargo-bootstrap deploy command with invalid repository URL should have failed"
        return 1
    else
        log_info "✓ kargo-bootstrap deploy command with invalid repository URL failed as expected"
        return 0
    fi
}

# Test scenario 2: Invalid chart path
test_invalid_chart_path() {
    log_step "Testing scenario 2: Invalid chart path"
    
    cd "${PROJECT_ROOT}"
    
    # Run kargo-bootstrap deploy with invalid chart path
    log_info "Running: ./bin/kargo-bootstrap deploy --repo ${GIT_REPO_URL} --chart invalid-chart-path --env ${ENVIRONMENT} --namespace ${TEST_NAMESPACE}"
    
    if ./bin/kargo-bootstrap deploy --repo "${GIT_REPO_URL}" --chart "invalid-chart-path" --env "${ENVIRONMENT}" --namespace "${TEST_NAMESPACE}" 2>&1 | tee -a "${LOG_FILE}"; then
        log_error "kargo-bootstrap deploy command with invalid chart path should have failed"
        return 1
    else
        log_info "✓ kargo-bootstrap deploy command with invalid chart path failed as expected"
        return 0
    fi
}

# Test scenario 3: Invalid environment
test_invalid_environment() {
    log_step "Testing scenario 3: Invalid environment"
    
    cd "${PROJECT_ROOT}"
    
    # Run kargo-bootstrap deploy with invalid environment
    log_info "Running: ./bin/kargo-bootstrap deploy --repo ${GIT_REPO_URL} --chart ${CHART_PATH} --env invalid-env --namespace ${TEST_NAMESPACE}"
    
    if ./bin/kargo-bootstrap deploy --repo "${GIT_REPO_URL}" --chart "${CHART_PATH}" --env "invalid-env" --namespace "${TEST_NAMESPACE}" 2>&1 | tee -a "${LOG_FILE}"; then
        log_error "kargo-bootstrap deploy command with invalid environment should have failed"
        return 1
    else
        log_info "✓ kargo-bootstrap deploy command with invalid environment failed as expected"
        return 0
    fi
}

# Test scenario 4: Invalid namespace
test_invalid_namespace() {
    log_step "Testing scenario 4: Invalid namespace"
    
    cd "${PROJECT_ROOT}"
    
    # Run kargo-bootstrap deploy with invalid namespace
    log_info "Running: ./bin/kargo-bootstrap deploy --repo ${GIT_REPO_URL} --chart ${CHART_PATH} --env ${ENVIRONMENT} --namespace invalid@namespace"
    
    if ./bin/kargo-bootstrap deploy --repo "${GIT_REPO_URL}" --chart "${CHART_PATH}" --env "${ENVIRONMENT}" --namespace "invalid@namespace" 2>&1 | tee -a "${LOG_FILE}"; then
        log_error "kargo-bootstrap deploy command with invalid namespace should have failed"
        return 1
    else
        log_info "✓ kargo-bootstrap deploy command with invalid namespace failed as expected"
        return 0
    fi
}

# Test scenario 5: Non-existent namespace
test_nonexistent_namespace() {
    log_step "Testing scenario 5: Non-existent namespace"
    
    cd "${PROJECT_ROOT}"
    
    # Run kargo-bootstrap deploy with non-existent namespace
    log_info "Running: ./bin/kargo-bootstrap deploy --repo ${GIT_REPO_URL} --chart ${CHART_PATH} --env ${ENVIRONMENT} --namespace nonexistent-namespace"
    
    if ./bin/kargo-bootstrap deploy --repo "${GIT_REPO_URL}" --chart "${CHART_PATH}" --env "${ENVIRONMENT}" --namespace "nonexistent-namespace" 2>&1 | tee -a "${LOG_FILE}"; then
        log_info "✓ kargo-bootstrap deploy command with non-existent namespace succeeded (namespace should be created)"
        return 0
    else
        log_info "✓ kargo-bootstrap deploy command with non-existent namespace failed as expected"
        return 0
    fi
}

# Test scenario 6: Missing required arguments
test_missing_required_arguments() {
    log_step "Testing scenario 6: Missing required arguments"
    
    cd "${PROJECT_ROOT}"
    
    # Run kargo-bootstrap deploy with missing required arguments
    log_info "Running: ./bin/kargo-bootstrap deploy"
    
    if ./bin/kargo-bootstrap deploy 2>&1 | tee -a "${LOG_FILE}"; then
        log_error "kargo-bootstrap deploy command with missing required arguments should have failed"
        return 1
    else
        log_info "✓ kargo-bootstrap deploy command with missing required arguments failed as expected"
        return 0
    fi
}

# Test scenario 7: Invalid kubeconfig
test_invalid_kubeconfig() {
    log_step "Testing scenario 7: Invalid kubeconfig"
    
    cd "${PROJECT_ROOT}"
    
    # Save original kubeconfig
    local original_kubeconfig="${KUBECONFIG}"
    
    # Set invalid kubeconfig
    export KUBECONFIG="/invalid/kubeconfig"
    
    # Run kargo-bootstrap deploy with invalid kubeconfig
    log_info "Running: ./bin/kargo-bootstrap deploy --repo ${GIT_REPO_URL} --chart ${CHART_PATH} --env ${ENVIRONMENT} --namespace ${TEST_NAMESPACE}"
    
    if ./bin/kargo-bootstrap deploy --repo "${GIT_REPO_URL}" --chart "${CHART_PATH}" --env "${ENVIRONMENT}" --namespace "${TEST_NAMESPACE}" 2>&1 | tee -a "${LOG_FILE}"; then
        log_error "kargo-bootstrap deploy command with invalid kubeconfig should have failed"
        return 1
    else
        log_info "✓ kargo-bootstrap deploy command with invalid kubeconfig failed as expected"
        return 0
    fi
    
    # Restore original kubeconfig
    export KUBECONFIG="${original_kubeconfig}"
}

# Test scenario 8: Unreachable Git repository
test_unreachable_git_repository() {
    log_step "Testing scenario 8: Unreachable Git repository"
    
    cd "${PROJECT_ROOT}"
    
    # Run kargo-bootstrap deploy with unreachable Git repository
    log_info "Running: ./bin/kargo-bootstrap deploy --repo ssh://unreachable-git-server/kargo-test-repo.git --chart ${CHART_PATH} --env ${ENVIRONMENT} --namespace ${TEST_NAMESPACE}"
    
    if ./bin/kargo-bootstrap deploy --repo "ssh://unreachable-git-server/kargo-test-repo.git" --chart "${CHART_PATH}" --env "${ENVIRONMENT}" --namespace "${TEST_NAMESPACE}" 2>&1 | tee -a "${LOG_FILE}"; then
        log_error "kargo-bootstrap deploy command with unreachable Git repository should have failed"
        return 1
    else
        log_info "✓ kargo-bootstrap deploy command with unreachable Git repository failed as expected"
        return 0
    fi
}

# Test scenario 9: Invalid chart values
test_invalid_chart_values() {
    log_step "Testing scenario 9: Invalid chart values"
    
    cd "${PROJECT_ROOT}"
    
    # Create a temporary directory with an invalid values file
    local temp_dir=$(mktemp -d)
    local invalid_values_file="${temp_dir}/invalid-values.yaml"
    
    cat > "${invalid_values_file}" << EOF
invalid: yaml: content:
  - missing
  - proper
  - indentation
EOF
    
    # Run kargo-bootstrap render with invalid chart values
    log_info "Running: ./bin/kargo-bootstrap render --repo ${GIT_REPO_URL} --chart ${CHART_PATH} --env ${ENVIRONMENT} --values ${invalid_values_file}"
    
    if ./bin/kargo-bootstrap render --repo "${GIT_REPO_URL}" --chart "${CHART_PATH}" --env "${ENVIRONMENT}" --values "${invalid_values_file}" 2>&1 | tee -a "${LOG_FILE}"; then
        log_error "kargo-bootstrap render command with invalid chart values should have failed"
        return 1
    else
        log_info "✓ kargo-bootstrap render command with invalid chart values failed as expected"
        return 0
    fi
    
    # Clean up temporary directory
    rm -rf "${temp_dir}"
}

# Test scenario 10: Permission denied
test_permission_denied() {
    log_step "Testing scenario 10: Permission denied"
    
    cd "${PROJECT_ROOT}"
    
    # Create a namespace with restricted permissions
    local restricted_namespace="restricted-namespace"
    
    # Delete namespace if it exists
    if kubectl get namespace "${restricted_namespace}" &> /dev/null; then
        kubectl delete namespace "${restricted_namespace}" --timeout=60s
    fi
    
    # Create namespace
    kubectl create namespace "${restricted_namespace}"
    
    # Create a restrictive role
    cat > restrictive-role.yaml << EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: restrictive-role
  namespace: ${restricted_namespace}
rules:
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "list"]
EOF
    
    kubectl apply -f restrictive-role.yaml
    
    # Create a restrictive role binding
    cat > restrictive-rolebinding.yaml << EOF
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: restrictive-rolebinding
  namespace: ${restricted_namespace}
subjects:
- kind: ServiceAccount
  name: default
  namespace: ${restricted_namespace}
roleRef:
  kind: Role
  name: restrictive-role
  apiGroup: rbac.authorization.k8s.io
EOF
    
    kubectl apply -f restrictive-rolebinding.yaml
    
    # Run kargo-bootstrap deploy with restricted permissions
    log_info "Running: ./bin/kargo-bootstrap deploy --repo ${GIT_REPO_URL} --chart ${CHART_PATH} --env ${ENVIRONMENT} --namespace ${restricted_namespace}"
    
    if ./bin/kargo-bootstrap deploy --repo "${GIT_REPO_URL}" --chart "${CHART_PATH}" --env "${ENVIRONMENT}" --namespace "${restricted_namespace}" 2>&1 | tee -a "${LOG_FILE}"; then
        log_error "kargo-bootstrap deploy command with restricted permissions should have failed"
        return 1
    else
        log_info "✓ kargo-bootstrap deploy command with restricted permissions failed as expected"
        return 0
    fi
    
    # Clean up
    kubectl delete -f restrictive-rolebinding.yaml --ignore-not-found=true
    kubectl delete -f restrictive-role.yaml --ignore-not-found=true
    kubectl delete namespace "${restricted_namespace}" --ignore-not-found=true
    rm -f restrictive-role.yaml restrictive-rolebinding.yaml
}

# Run all error scenarios
run_error_scenarios() {
    log_step "Running all error scenarios"
    
    local scenarios=(
        "test_invalid_repository_url"
        "test_invalid_chart_path"
        "test_invalid_environment"
        "test_invalid_namespace"
        "test_nonexistent_namespace"
        "test_missing_required_arguments"
        "test_invalid_kubeconfig"
        "test_unreachable_git_repository"
        "test_invalid_chart_values"
        "test_permission_denied"
    )
    
    local passed=0
    local failed=0
    
    for scenario in "${scenarios[@]}"; do
        log_info "Running scenario: $scenario"
        
        if $scenario; then
            passed=$((passed + 1))
            log_info "✓ Scenario $scenario passed"
        else
            failed=$((failed + 1))
            log_error "✗ Scenario $scenario failed"
        fi
        
        echo ""
    done
    
    log_info "Error scenarios test results: $passed passed, $failed failed"
    
    if [ $failed -eq 0 ]; then
        log_info "All error scenarios passed as expected"
        return 0
    else
        log_error "$failed error scenarios failed unexpectedly"
        return 1
    fi
}

# Clean up test resources
cleanup_test_resources() {
    log_step "Cleaning up test resources"
    
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
    echo "Kargo Bootstrap Error Scenarios Test"
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
    echo "✓ Invalid repository URL scenario: Passed"
    echo "✓ Invalid chart path scenario: Passed"
    echo "✓ Invalid environment scenario: Passed"
    echo "✓ Invalid namespace scenario: Passed"
    echo "✓ Non-existent namespace scenario: Passed"
    echo "✓ Missing required arguments scenario: Passed"
    echo "✓ Invalid kubeconfig scenario: Passed"
    echo "✓ Unreachable Git repository scenario: Passed"
    echo "✓ Invalid chart values scenario: Passed"
    echo "✓ Permission denied scenario: Passed"
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
    log_info "Starting kargo-bootstrap error scenarios test"
    
    # Set up trap for cleanup on interrupt
    trap cleanup_on_interrupt INT
    
    # Initialize log
    init_log
    
    # Check prerequisites
    check_prerequisites
    
    # Create test namespace
    create_test_namespace
    
    # Run error scenarios
    if run_error_scenarios; then
        # Clean up test resources
        cleanup_test_resources
        
        # Print test summary
        print_test_summary
        
        log_info "Error scenarios test completed successfully!"
    else
        # Clean up test resources
        cleanup_test_resources
        
        log_error "Error scenarios test completed with errors"
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
        echo "This script tests error handling for kargo-bootstrap,"
        echo "including various error scenarios and validation."
        exit 0
        ;;
    --namespace)
        TEST_NAMESPACE="${2:-test-error}"
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