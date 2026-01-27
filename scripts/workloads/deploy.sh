#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# Application Deployment
# -----------------------------------------------------------------------------
# Deploys lab applications to the Kubernetes cluster.
#
# What this does:
# 1. Starts tunnel to Kubernetes API (port 6443)
# 2. Creates application namespace
# 3. Deploys NGINX (stateless, 2 replicas)
# 4. Deploys Redis (stateful, with persistent storage)
# 5. Waits for deployments to be ready
# 6. Stops tunnel
#
# Prerequisites:
# - Cluster tools must be installed first (make deploy-tools <cloud>)
# - CSI driver and StorageClass required for Redis PVC
#
# Usage: ./deploy.sh <cloud>
# Run via: make deploy-applications <cloud>
# -----------------------------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"

check_prerequisites() {
    log_step "Checking prerequisites"
    if [[ ! -f "${CONFIGS_DIR}/kubeconfig" ]]; then
        log_error "kubeconfig not found. Run 'make deploy-infra <cloud>' first"
        exit 1
    fi

    # Check if StorageClass exists
    if ! kubectl get storageclass standard &>/dev/null; then
        log_error "StorageClass 'standard' not found. Run 'make deploy-tools <cloud>' first"
        exit 1
    fi

    log_info "Prerequisites satisfied"
}

deploy_applications() {
    local cloud=$1
    log_step "Deploying applications"

    local apps_dir="${REPO_ROOT}/apps"

    # Apply namespace
    log_info "Creating namespace 'application'"
    kubectl apply -f "${apps_dir}/namespace.yaml"

    # Deploy NGINX (stateless, 2 replicas across zones)
    log_info "Deploying NGINX"
    kubectl apply -f "${apps_dir}/nginx.yaml"

    # Deploy Redis (stateful, 1 replica with persistent storage)
    log_info "Deploying Redis with persistent storage"
    kubectl apply -f "${apps_dir}/redis.yaml"

    # Wait for deployments to be ready
    log_info "Waiting for NGINX deployment"
    kubectl rollout status deployment/nginx -n application --timeout=120s

    log_info "Waiting for Redis deployment"
    kubectl rollout status deployment/redis -n application --timeout=300s

    log_info "Applications deployed"
}

print_usage() {
    local cloud=$1
    source "${LIB_DIR}/workloads.sh"
    apps_status

    cat <<EOF
To access applications:
  make connect ${cloud}

Then in another terminal:
  export KUBECONFIG=${CONFIGS_DIR}/kubeconfig

  # Test NGINX
  kubectl port-forward svc/nginx -n application 8080:80
  curl http://localhost:8080

  # Test Redis
  kubectl exec -it deploy/redis -n application -- redis-cli ping

To seed Redis with test data:
  make seed-redis ${cloud}

To backup applications:
  make backup ${cloud}

EOF
}

main() {
    local cloud=$1

    validate_cloud "$cloud"
    source_cloud_modules "$cloud"
    source "${LIB_DIR}/workloads.sh"

    echo "=============================================="
    echo "  Kubernetes Lab - Application Deployment (${cloud})"
    echo "=============================================="
    echo ""

    setup_error_handling
    check_prerequisites
    k8s_connect
    deploy_applications "$cloud"
    print_usage "$cloud"
}

main "$@"
