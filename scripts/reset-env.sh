#!/bin/bash

# scripts/reset-env.sh
# Script to reset the test environment to a clean state for kargo-bootstrap testing

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
LOG_FILE="${PROJECT_ROOT}/reset-env.log"

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
    log_step "Initializing reset log file"
    
    cat > "${LOG_FILE}" << EOF
# Kargo Bootstrap Test Environment Reset Log
# Date: $(date)
# User: $(whoami)
# Host: $(hostname)

EOF
    
    log_info "Reset log file initialized: ${LOG_FILE}"
}

# Reset Argo CD applications
reset_argocd_applications() {
    log_step "Resetting Argo CD applications"
    
    # Set KUBECONFIG for Argo CD operations
    export KUBECONFIG="${KUBECONFIG_PATH}"
    
    # Check if Argo CD is installed
    if ! kubectl get namespace argocd &> /dev/null; then
        log_warn "Argo CD namespace not found. Skipping Argo CD application reset."
        return 0
    fi
    
    # Get all applications
    local apps=$(kubectl get applications -n argocd -o jsonpath='{.items[*].metadata.name}' 2>/dev/null || true)
    
    if [ -n "$apps" ]; then
        log_info "Deleting Argo CD applications: $apps"
        
        for app in $apps; do
            log_info "Deleting application: $app"
            kubectl delete application "$app" -n argocd --ignore-not-found=true 2>&1 | tee -a "${LOG_FILE}" || true
        done
    else
        log_info "No Argo CD applications found"
    fi
    
    log_info "Argo CD applications reset completed"
}

# Reset Argo CD projects
reset_argocd_projects() {
    log_step "Resetting Argo CD projects"
    
    # Set KUBECONFIG for Argo CD operations
    export KUBECONFIG="${KUBECONFIG_PATH}"
    
    # Check if Argo CD is installed
    if ! kubectl get namespace argocd &> /dev/null; then
        log_warn "Argo CD namespace not found. Skipping Argo CD project reset."
        return 0
    fi
    
    # Get all projects except the default project
    local projects=$(kubectl get appprojects -n argocd -o jsonpath='{.items[*].metadata.name}' 2>/dev/null || true)
    
    if [ -n "$projects" ]; then
        log_info "Deleting Argo CD projects: $projects"
        
        for project in $projects; do
            # Skip the default project
            if [ "$project" != "default" ]; then
                log_info "Deleting project: $project"
                kubectl delete appproject "$project" -n argocd --ignore-not-found=true 2>&1 | tee -a "${LOG_FILE}" || true
            fi
        done
    else
        log_info "No Argo CD projects found"
    fi
    
    log_info "Argo CD projects reset completed"
}

# Reset Kubernetes namespaces
reset_namespaces() {
    log_step "Resetting Kubernetes namespaces"
    
    # Set KUBECONFIG for Kubernetes operations
    export KUBECONFIG="${KUBECONFIG_PATH}"
    
    # Get all namespaces except system namespaces
    local namespaces=$(kubectl get namespaces -o jsonpath='{.items[*].metadata.name}' 2>/dev/null || true)
    
    # System namespaces to keep
    local system_namespaces="default kube-system kube-public kube-node-lease ingress-nginx argocd"
    
    if [ -n "$namespaces" ]; then
        for namespace in $namespaces; do
            # Skip system namespaces
            if [[ ! " $system_namespaces " =~ " $namespace " ]]; then
                log_info "Deleting namespace: $namespace"
                kubectl delete namespace "$namespace" --ignore-not-found=true --timeout=60s 2>&1 | tee -a "${LOG_FILE}" || true
            fi
        done
    else
        log_info "No namespaces found"
    fi
    
    log_info "Kubernetes namespaces reset completed"
}

# Reset Argo CD configuration
reset_argocd_config() {
    log_step "Resetting Argo CD configuration"
    
    # Set KUBECONFIG for Argo CD operations
    export KUBECONFIG="${KUBECONFIG_PATH}"
    
    # Check if Argo CD is installed
    if ! kubectl get namespace argocd &> /dev/null; then
        log_warn "Argo CD namespace not found. Skipping Argo CD configuration reset."
        return 0
    fi
    
    # Reset Argo CD server configuration
    log_info "Resetting Argo CD server configuration"
    
    # Delete custom Argo CD configurations
    kubectl delete configmap argocd-cm -n argocd --ignore-not-found=true 2>&1 | tee -a "${LOG_FILE}" || true
    kubectl delete configmap argocd-rbac-cm -n argocd --ignore-not-found=true 2>&1 | tee -a "${LOG_FILE}" || true
    kubectl delete secret argocd-secret -n argocd --ignore-not-found=true 2>&1 | tee -a "${LOG_FILE}" || true
    
    # Restart Argo CD server to apply default configuration
    log_info "Restarting Argo CD server"
    kubectl rollout restart deployment/argocd-server -n argocd 2>&1 | tee -a "${LOG_FILE}" || true
    
    # Wait for Argo CD server to be ready
    kubectl rollout status deployment/argocd-server -n argocd --timeout=300s 2>&1 | tee -a "${LOG_FILE}" || true
    
    log_info "Argo CD configuration reset completed"
}

# Reset Git server
reset_git_server() {
    log_step "Resetting Git server"
    
    # Set KUBECONFIG for Git server operations
    export KUBECONFIG="${KUBECONFIG_PATH}"
    
    # Check if Git server is running
    if ! kubectl get deployment git-server -n default &> /dev/null; then
        log_warn "Git server deployment not found. Skipping Git server reset."
        return 0
    fi
    
    # Restart Git server
    log_info "Restarting Git server"
    kubectl rollout restart deployment/git-server -n default 2>&1 | tee -a "${LOG_FILE}" || true
    
    # Wait for Git server to be ready
    kubectl rollout status deployment/git-server -n default --timeout=300s 2>&1 | tee -a "${LOG_FILE}" || true
    
    log_info "Git server reset completed"
}

# Reset test repository
reset_test_repository() {
    log_step "Resetting test repository"
    
    local repo_name="kargo-test-repo"
    local repo_dir="${PROJECT_ROOT}/${repo_name}"
    
    if [ -d "${repo_dir}" ]; then
        log_info "Resetting test repository: ${repo_dir}"
        
        cd "${repo_dir}"
        
        # Reset to initial commit
        if git log --oneline | head -1 | grep -q "Initial commit"; then
            git reset --hard HEAD~1 2>&1 | tee -a "${LOG_FILE}" || true
        fi
        
        # Clean up untracked files
        git clean -fd 2>&1 | tee -a "${LOG_FILE}" || true
        
        # Re-initialize if needed
        if [ ! -d ".git" ]; then
            git init 2>&1 | tee -a "${LOG_FILE}" || true
            git config user.email "test@example.com"
            git config user.name "Test User"
        fi
        
        # Add all files and commit
        git add . 2>&1 | tee -a "${LOG_FILE}" || true
        git commit -m "Reset test repository" 2>&1 | tee -a "${LOG_FILE}" || true
        
        # Push to Git server if remote exists
        if git remote get-url origin &> /dev/null; then
            git push -f origin master 2>&1 | tee -a "${LOG_FILE}" || true
        fi
        
        log_info "Test repository reset completed"
    else
        log_warn "Test repository not found: ${repo_dir}"
    fi
}

# Recreate test projects
recreate_test_projects() {
    log_step "Recreating test projects"
    
    # Set KUBECONFIG for Argo CD operations
    export KUBECONFIG="${KUBECONFIG_PATH}"
    
    # Check if Argo CD is installed
    if ! kubectl get namespace argocd &> /dev/null; then
        log_warn "Argo CD namespace not found. Skipping test project recreation."
        return 0
    fi
    
    # Create test project
    cat > test-project.yaml << EOF
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: test-project
  namespace: argocd
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
    
    kubectl apply -f test-project.yaml 2>&1 | tee -a "${LOG_FILE}" || true
    rm -f test-project.yaml
    
    # Create development project
    cat > dev-project.yaml << EOF
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: development
  namespace: argocd
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
    
    kubectl apply -f dev-project.yaml 2>&1 | tee -a "${LOG_FILE}" || true
    rm -f dev-project.yaml
    
    # Create staging project
    cat > staging-project.yaml << EOF
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: staging
  namespace: argocd
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
    
    kubectl apply -f staging-project.yaml 2>&1 | tee -a "${LOG_FILE}" || true
    rm -f staging-project.yaml
    
    # Create production project
    cat > prod-project.yaml << EOF
apiVersion: argoproj.io/v1alpha1
kind: AppProject
metadata:
  name: production
  namespace: argocd
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
    
    kubectl apply -f prod-project.yaml 2>&1 | tee -a "${LOG_FILE}" || true
    rm -f prod-project.yaml
    
    log_info "Test projects recreated"
}

# Verify reset
verify_reset() {
    log_step "Verifying reset"
    
    # Set KUBECONFIG for verification
    export KUBECONFIG="${KUBECONFIG_PATH}"
    
    local reset_success=true
    
    # Check if Kind cluster exists
    if kind get clusters | grep -q "kargo-test"; then
        log_info "✓ Kind cluster 'kargo-test' is running"
    else
        log_error "✗ Kind cluster 'kargo-test' is not running"
        reset_success=false
    fi
    
    # Check if Argo CD is installed
    if kubectl get pods -n argocd | grep -q "argocd-server"; then
        log_info "✓ Argo CD is installed"
    else
        log_error "✗ Argo CD is not installed"
        reset_success=false
    fi
    
    # Check if Git server is running
    if kubectl get pods -n default | grep -q "git-server"; then
        log_info "✓ Git server is running"
    else
        log_error "✗ Git server is not running"
        reset_success=false
    fi
    
    # Check if test projects exist
    local projects="test-project development staging production"
    for project in $projects; do
        if kubectl get appproject "$project" -n argocd &> /dev/null; then
            log_info "✓ Argo CD project '$project' exists"
        else
            log_error "✗ Argo CD project '$project' does not exist"
            reset_success=false
        fi
    done
    
    # Check if no applications exist
    local apps=$(kubectl get applications -n argocd -o jsonpath='{.items[*].metadata.name}' 2>/dev/null || true)
    if [ -z "$apps" ]; then
        log_info "✓ No Argo CD applications exist"
    else
        log_error "✗ Argo CD applications still exist: $apps"
        reset_success=false
    fi
    
    if [ "$reset_success" = true ]; then
        log_info "Reset verification completed successfully"
        return 0
    else
        log_error "Reset verification failed"
        return 1
    fi
}

# Print reset summary
print_reset_summary() {
    log_step "Printing reset summary"
    
    echo ""
    echo "=========================================="
    echo "Kargo Bootstrap Test Environment Reset"
    echo "=========================================="
    echo ""
    echo "Reset Summary:"
    echo "=============="
    echo "Argo CD Applications: Deleted"
    echo "Argo CD Projects: Recreated"
    echo "Kubernetes Namespaces: Reset (except system namespaces)"
    echo "Argo CD Configuration: Reset to defaults"
    echo "Git Server: Restarted"
    echo "Test Repository: Reset to initial state"
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
    echo "For more information, see:"
    echo "- ${PROJECT_ROOT}/docs/test-environment.md"
    echo "- ${LOG_FILE}"
    echo ""
}

# Cleanup function for interrupted reset
cleanup_on_interrupt() {
    log_warn "Reset interrupted. Partial reset may have occurred."
    log_info "Check the log file for details: ${LOG_FILE}"
    exit 1
}

# Main execution
main() {
    log_info "Starting kargo-bootstrap test environment reset"
    
    # Set up trap for cleanup on interrupt
    trap cleanup_on_interrupt INT
    
    # Initialize log
    init_log
    
    # Reset Argo CD applications
    reset_argocd_applications
    
    # Reset Argo CD projects
    reset_argocd_projects
    
    # Reset Kubernetes namespaces
    reset_namespaces
    
    # Reset Argo CD configuration
    reset_argocd_config
    
    # Reset Git server
    reset_git_server
    
    # Reset test repository
    reset_test_repository
    
    # Recreate test projects
    recreate_test_projects
    
    # Verify reset
    if verify_reset; then
        # Print reset summary
        print_reset_summary
        
        log_info "Test environment reset completed successfully!"
    else
        log_error "Test environment reset completed with errors"
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
        echo "This script resets the test environment for kargo-bootstrap testing"
        echo "to a clean state, removing applications and resetting configurations."
        exit 0
        ;;
    --force)
        log_info "Running reset in force mode"
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