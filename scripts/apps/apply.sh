#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# Application Deployment
# -----------------------------------------------------------------------------
# Deploys lab applications to the Kubernetes cluster.
#
# What this does:
# 1. Starts tunnel to Kubernetes API (port 6443)
# 2. Installs cloud-specific CSI driver (for persistent storage)
# 3. Installs Velero with cloud-specific plugin (for backup/restore)
# 4. Creates StorageClass for persistent disks
# 5. Deploys applications (NGINX, Redis)
# 6. Stops tunnel
#
# Usage: ./apply.sh <cloud>
# Run via: make apply <cloud>
# -----------------------------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"

install_csi_driver() {
    local cloud=$1
    case "$cloud" in
        gcp)
            source "${LIB_DIR}/gcp/csi.sh"
            gcp_csi_install
            ;;
    esac
}

install_velero() {
    local cloud=$1
    case "$cloud" in
        gcp)
            source "${LIB_DIR}/gcp/velero.sh"
            gcp_velero_install
            ;;
    esac
}

check_apply_prerequisites() {
    log_step "Checking prerequisites"
    if [[ ! -f "${CONFIGS_DIR}/kubeconfig" ]]; then
        log_error "kubeconfig not found. Run 'make deploy <cloud>' first"
        exit 1
    fi
    log_info "Prerequisites satisfied"
}

print_apply_usage() {
    local cloud=$1
    source "${LIB_DIR}/apps.sh"
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
  make seed ${cloud}

EOF
}

main() {
    local cloud=$1

    validate_cloud "$cloud"
    source_cloud_modules "$cloud"
    source "${LIB_DIR}/apps.sh"
    source "${LIB_DIR}/velero.sh"

    echo "=============================================="
    echo "  Kubernetes Lab - Application Deployment (${cloud})"
    echo "=============================================="
    echo ""

    setup_error_handling
    check_apply_prerequisites
    k8s_connect
    install_csi_driver "$cloud"
    install_velero "$cloud"
    apps_deploy "$cloud"
    print_apply_usage "$cloud"
}

main "$@"
