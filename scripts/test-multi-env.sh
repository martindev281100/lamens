#!/bin/bash

# scripts/test-multi-env.sh
# Script to test multi-environment deployment for kargo-bootstrap

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
LOG_FILE="${PROJECT_ROOT}/test-multi-env.log"
GIT_REPO_URL="ssh://git@git-server.default.svc.cluster.local:30888/kargo-test-repo.git"
CHART_PATH="charts/web-app"
ENVIRONMENTS="dev staging prod"
NAMESPACE_PREFIX="kargo-test"

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
# Kargo Bootstrap Multi-Environment Test Log
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
    
    # Check if Argo CD projects exist
    for env in $ENVIRONMENTS; do
        if [ "$env" = "dev" ]; then
            project="development"
        else
            project="$env"
        fi
        
        if ! kubectl get appproject "$project" -n argocd &> /dev/null; then
            log_error "Argo CD project not found: $project"
            exit 1
        fi
    done
    
    # Check if Git server is running
    if ! kubectl get deployment git-server -n default &> /dev/null; then
        log_error "Git server deployment not found"
        exit 1
    fi
    
    log_info "All prerequisites are satisfied"
}

# Create test namespaces
create_test_namespaces() {
    log_step "Creating test namespaces"
    
    for env in $ENVIRONMENTS; do
        local namespace="${NAMESPACE_PREFIX}-${env}"
        
        # Delete namespace if it exists
        if kubectl get namespace "$namespace" &> /dev/null; then
            log_warn "Namespace $namespace already exists, deleting it"
            kubectl delete namespace "$namespace" --timeout=60s
        fi
        
        # Create namespace
        kubectl create namespace "$namespace"
        
        log_info "Test namespace created: $namespace"
    done
}

# Test kargo-bootstrap deploy command for each environment
test_deploy_commands() {
    log_step "Testing kargo-bootstrap deploy commands for all environments"
    
    cd "${PROJECT_ROOT}"
    
    for env in $ENVIRONMENTS; do
        local namespace="${NAMESPACE_PREFIX}-${env}"
        
        log_info "Deploying to environment: $env (namespace: $namespace)"
        
        # Run kargo-bootstrap deploy
        log_info "Running: ./bin/kargo-bootstrap deploy --repo ${GIT_REPO_URL} --chart ${CHART_PATH} --env ${env} --namespace ${namespace}"
        
        if ./bin/kargo-bootstrap deploy --repo "${GIT_REPO_URL}" --chart "${CHART_PATH}" --env "${env}" --namespace "$namespace" 2>&1 | tee -a "${LOG_FILE}"; then
            log_info "kargo-bootstrap deploy command for environment $env executed successfully"
        else
            log_error "kargo-bootstrap deploy command for environment $env failed"
            exit 1
        fi
    done
}

# Verify deployments in Argo CD
verify_argocd_deployments() {
    log_step "Verifying deployments in Argo CD"
    
    for env in $ENVIRONMENTS; do
        local namespace="${NAMESPACE_PREFIX}-${env}"
        
        # Get application name
        local app_name=$(kubectl get applications -n argocd -o jsonpath='{.items[?(@.spec.destination.namespace=="'$namespace'")].metadata.name}' 2>/dev/null || true)
        
        if [ -z "$app_name" ]; then
            log_error "No Argo CD application found for namespace $namespace"
            exit 1
        fi
        
        log_info "Found Argo CD application for $env: $app_name"
        
        # Wait for application to be synced
        log_info "Waiting for application $app_name to be synced..."
        
        local max_attempts=30
        local attempt=0
        
        while [ $attempt -lt $max_attempts ]; do
            local sync_status=$(kubectl get application "$app_name" -n argocd -o jsonpath='{.status.sync.status}' 2>/dev/null || true)
            local health_status=$(kubectl get application "$app_name" -n argocd -o jsonpath='{.status.health.status}' 2>/dev/null || true)
            
            log_info "Attempt $((attempt+1)) for $env: Sync status: $sync_status, Health status: $health_status"
            
            if [ "$sync_status" = "Synced" ] && [ "$health_status" = "Healthy" ]; then
                log_info "Application $app_name is synced and healthy"
                break
            fi
            
            sleep 10
            attempt=$((attempt+1))
        done
        
        if [ $attempt -eq $max_attempts ]; then
            log_error "Application $app_name did not become synced and healthy within expected time"
            exit 1
        fi
    done
}

# Verify deployments in Kubernetes
verify_kubernetes_deployments() {
    log_step "Verifying deployments in Kubernetes"
    
    for env in $ENVIRONMENTS; do
        local namespace="${NAMESPACE_PREFIX}-${env}"
        
        # Check if deployment exists
        local deployment_name=$(kubectl get deployments -n "$namespace" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
        
        if [ -z "$deployment_name" ]; then
            log_error "No deployment found in namespace $namespace"
            exit 1
        fi
        
        log_info "Found deployment for $env: $deployment_name"
        
        # Wait for deployment to be ready
        log_info "Waiting for deployment $deployment_name to be ready..."
        
        if kubectl rollout status deployment/"$deployment_name" -n "$namespace" --timeout=300s; then
            log_info "Deployment $deployment_name is ready"
        else
            log_error "Deployment $deployment_name did not become ready within expected time"
            exit 1
        fi
        
        # Check pods
        local pod_count=$(kubectl get pods -n "$namespace" --field-selector=status.phase=Running --no-headers | wc -l)
        
        if [ "$pod_count" -gt 0 ]; then
            log_info "Found $pod_count running pods in namespace $namespace"
        else
            log_error "No running pods found in namespace $namespace"
            exit 1
        fi
        
        # Check services
        local service_count=$(kubectl get services -n "$namespace" --no-headers | wc -l)
        
        if [ "$service_count" -gt 0 ]; then
            log_info "Found $service_count services in namespace $namespace"
        else
            log_error "No services found in namespace $namespace"
            exit 1
        fi
        
        # Check environment-specific configurations
        verify_environment_config "$env" "$namespace"
    done
}

# Verify environment-specific configurations
verify_environment_config() {
    local env="$1"
    local namespace="$2"
    
    log_info "Verifying environment-specific configurations for $env"
    
    # Get deployment name
    local deployment_name=$(kubectl get deployments -n "$namespace" -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || true)
    
    case "$env" in
        "dev")
            # Verify dev-specific configuration
            local replicas=$(kubectl get deployment "$deployment_name" -n "$namespace" -o jsonpath='{.spec.replicas}')
            if [ "$replicas" = "1" ]; then
                log_info "✓ Dev environment has 1 replica"
            else
                log_error "✗ Dev environment should have 1 replica, found $replicas"
                exit 1
            fi
            ;;
        "staging")
            # Verify staging-specific configuration
            local replicas=$(kubectl get deployment "$deployment_name" -n "$namespace" -o jsonpath='{.spec.replicas}')
            if [ "$replicas" = "2" ]; then
                log_info "✓ Staging environment has 2 replicas"
            else
                log_error "✗ Staging environment should have 2 replicas, found $replicas"
                exit 1
            fi
            ;;
        "prod")
            # Verify prod-specific configuration
            local replicas=$(kubectl get deployment "$deployment_name" -n "$namespace" -o jsonpath='{.spec.replicas}')
            if [ "$replicas" = "3" ]; then
                log_info "✓ Prod environment has 3 replicas"
            else
                log_error "✗ Prod environment should have 3 replicas, found $replicas"
                exit 1
            fi
            ;;
    esac
}

# Test kargo-bootstrap status command for each environment
test_status_commands() {
    log_step "Testing kargo-bootstrap status commands for all environments"
    
    cd "${PROJECT_ROOT}"
    
    for env in $ENVIRONMENTS; do
        local namespace="${NAMESPACE_PREFIX}-${env}"
        
        log_info "Checking status for environment: $env (namespace: $namespace)"
        
        # Run kargo-bootstrap status
        log_info "Running: ./bin/kargo-bootstrap status --namespace ${namespace}"
        
        if ./bin/kargo-bootstrap status --namespace "$namespace" 2>&1 | tee -a "${LOG_FILE}"; then
            log_info "kargo-bootstrap status command for environment $env executed successfully"
        else
            log_error "kargo-bootstrap status command for environment $env failed"
            exit 1
        fi
    done
}

# Test kargo-bootstrap render command for each environment
test_render_commands() {
    log_step "Testing kargo-bootstrap render commands for all environments"
    
    cd "${PROJECT_ROOT}"
    
    for env in $ENVIRONMENTS; do
        log_info "Rendering for environment: $env"
        
        # Run kargo-bootstrap render
        log_info "Running: ./bin/kargo-bootstrap render --repo ${GIT_REPO_URL} --chart ${CHART_PATH} --env ${env}"
        
        if ./bin/kargo-bootstrap render --repo "${GIT_REPO_URL}" --chart "${CHART_PATH}" --env "$env" 2>&1 | tee -a "${LOG_FILE}"; then
            log_info "kargo-bootstrap render command for environment $env executed successfully"
        else
            log_error "kargo-bootstrap render command for environment $env failed"
            exit 1
        fi
    done
}

# Test environment promotion workflow
test_promotion_workflow() {
    log_step "Testing environment promotion workflow"
    
    # This test simulates a promotion workflow from dev to staging to prod
    # In a real scenario, this would involve more complex validation and approval steps
    
    log_info "Simulating promotion from dev to staging"
    
    # Get the latest commit from the dev deployment
    local dev_namespace="${NAMESPACE_PREFIX}-dev"
    local dev_app_name=$(kubectl get applications -n argocd -o jsonpath='{.items[?(@.spec.destination.namespace=="'$dev_namespace'")].metadata.name}' 2>/dev/null || true)
    local dev_revision=$(kubectl get application "$dev_app_name" -n argocd -o jsonpath='{.status.sync.revision}' 2>/dev/null || true)
    
    log_info "Dev deployment revision: $dev_revision"
    
    # Promote to staging by deploying the same revision
    local staging_namespace="${NAMESPACE_PREFIX}-staging"
    
    log_info "Promoting to staging with revision: $dev_revision"
    
    # In a real scenario, you might use a specific command to promote
    # For this test, we'll just verify that the staging deployment is using the same revision
    local staging_app_name=$(kubectl get applications -n argocd -o jsonpath='{.items[?(@.spec.destination.namespace=="'$staging_namespace'")].metadata.name}' 2>/dev/null || true)
    local staging_revision=$(kubectl get application "$staging_app_name" -n argocd -o jsonpath='{.status.sync.revision}' 2>/dev/null || true)
    
    if [ "$dev_revision" = "$staging_revision" ]; then
        log_info "✓ Staging deployment is using the same revision as dev"
    else
        log_error "✗ Staging deployment revision ($staging_revision) does not match dev revision ($dev_revision)"
        exit 1
    fi
    
    log_info "Simulating promotion from staging to prod"
    
    # Promote to prod by deploying the same revision
    local prod_namespace="${NAMESPACE_PREFIX}-prod"
    
    log_info "Promoting to prod with revision: $staging_revision"
    
    # Verify that the prod deployment is using the same revision
    local prod_app_name=$(kubectl get applications -n argocd -o jsonpath='{.items[?(@.spec.destination.namespace=="'$prod_namespace'")].metadata.name}' 2>/dev/null || true)
    local prod_revision=$(kubectl get application "$prod_app_name" -n argocd -o jsonpath='{.status.sync.revision}' 2>/dev/null || true)
    
    if [ "$staging_revision" = "$prod_revision" ]; then
        log_info "✓ Prod deployment is using the same revision as staging"
    else
        log_error "✗ Prod deployment revision ($prod_revision) does not match staging revision ($staging_revision)"
        exit 1
    fi
    
    log_info "Environment promotion workflow test completed successfully"
}

# Clean up test resources
cleanup_test_resources() {
    log_step "Cleaning up test resources"
    
    for env in $ENVIRONMENTS; do
        local namespace="${NAMESPACE_PREFIX}-${env}"
        
        # Delete Argo CD application
        local app_name=$(kubectl get applications -n argocd -o jsonpath='{.items[?(@.spec.destination.namespace=="'$namespace'")].metadata.name}' 2>/dev/null || true)
        
        if [ -n "$app_name" ]; then
            log_info "Deleting Argo CD application: $app_name"
            kubectl delete application "$app_name" -n argocd --ignore-not-found=true
        fi
        
        # Delete test namespace
        if kubectl get namespace "$namespace" &> /dev/null; then
            log_info "Deleting test namespace: $namespace"
            kubectl delete namespace "$namespace" --ignore-not-found=true --timeout=60s
        fi
    done
    
    log_info "Test resources cleaned up"
}

# Print test summary
print_test_summary() {
    log_step "Printing test summary"
    
    echo ""
    echo "=========================================="
    echo "Kargo Bootstrap Multi-Environment Test"
    echo "=========================================="
    echo ""
    echo "Test Summary:"
    echo "============="
    echo "Namespace Prefix: ${NAMESPACE_PREFIX}"
    echo "Environments: ${ENVIRONMENTS}"
    echo "Git Repository: ${GIT_REPO_URL}"
    echo "Chart Path: ${CHART_PATH}"
    echo ""
    echo "Test Results:"
    echo "============="
    echo "✓ Prerequisites check: Passed"
    echo "✓ Test namespaces creation: Passed"
    echo "✓ kargo-bootstrap deploy commands: Passed"
    echo "✓ Argo CD deployments verification: Passed"
    echo "✓ Kubernetes deployments verification: Passed"
    echo "✓ Environment-specific configurations: Passed"
    echo "✓ kargo-bootstrap status commands: Passed"
    echo "✓ kargo-bootstrap render commands: Passed"
    echo "✓ Environment promotion workflow: Passed"
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
    log_info "Starting kargo-bootstrap multi-environment test"
    
    # Set up trap for cleanup on interrupt
    trap cleanup_on_interrupt INT
    
    # Initialize log
    init_log
    
    # Check prerequisites
    check_prerequisites
    
    # Create test namespaces
    create_test_namespaces
    
    # Test kargo-bootstrap deploy commands
    test_deploy_commands
    
    # Verify deployments in Argo CD
    verify_argocd_deployments
    
    # Verify deployments in Kubernetes
    verify_kubernetes_deployments
    
    # Test kargo-bootstrap status commands
    test_status_commands
    
    # Test kargo-bootstrap render commands
    test_render_commands
    
    # Test environment promotion workflow
    test_promotion_workflow
    
    # Clean up test resources
    cleanup_test_resources
    
    # Print test summary
    print_test_summary
    
    log_info "Multi-environment test completed successfully!"
}

# Handle command line arguments
case "${1:-}" in
    --help|-h)
        echo "Usage: $0 [OPTIONS]"
        echo ""
        echo "Options:"
        echo "  --help, -h    Show this help message"
        echo ""
        echo "This script tests multi-environment deployment for kargo-bootstrap,"
        echo "including deployment to multiple environments and promotion workflows."
        exit 0
        ;;
    --namespace-prefix)
        NAMESPACE_PREFIX="${2:-kargo-test}"
        log_info "Using custom namespace prefix: ${NAMESPACE_PREFIX}"
        main
        ;;
    --chart)
        CHART_PATH="${2:-charts/web-app}"
        log_info "Using custom chart path: ${CHART_PATH}"
        main
        ;;
    --environments)
        ENVIRONMENTS="${2:-dev staging prod}"
        log_info "Using custom environments: ${ENVIRONMENTS}"
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