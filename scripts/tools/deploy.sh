#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# Tools Deployment
# -----------------------------------------------------------------------------
# Deploys cluster tools required for applications to run.
#
# What this does:
# 1. Starts tunnel to Kubernetes API (port 6443)
# 2. Installs cloud-specific CSI driver (for persistent storage)
# 3. Creates StorageClass for persistent disks
# 4. Installs Velero with cloud-specific plugin (for backup/restore)
# 5. Stops tunnel
#
# These tools must be deployed before applications that use persistent storage
# or before performing backup/restore operations.
#
# Usage: ./deploy.sh <cloud>
# Run via: make deploy-tools <cloud>
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

install_storageclass() {
    local cloud=$1
    local apps_dir="${REPO_ROOT}/apps"

    if [[ -f "${apps_dir}/${cloud}/storageclass.yaml" ]]; then
        log_info "Applying ${cloud} StorageClass"
        kubectl apply -f "${apps_dir}/${cloud}/storageclass.yaml"
    else
        log_warn "No StorageClass found for ${cloud}"
    fi
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

check_prerequisites() {
    log_step "Checking prerequisites"
    if [[ ! -f "${CONFIGS_DIR}/kubeconfig" ]]; then
        log_error "kubeconfig not found. Run 'make deploy-infra <cloud>' first"
        exit 1
    fi
    log_info "Prerequisites satisfied"
}

print_usage() {
    local cloud=$1
    cat <<EOF

==============================================
  Tools deployed successfully!
==============================================

Next steps:
  make deploy-applications ${cloud}

Or restore from backup:
  make restore ${cloud}

EOF
}

main() {
    local cloud=$1

    validate_cloud "$cloud"
    source_cloud_modules "$cloud"
    source "${LIB_DIR}/velero.sh"

    echo "=============================================="
    echo "  Kubernetes Lab - Tools Deployment (${cloud})"
    echo "=============================================="
    echo ""

    setup_error_handling
    check_prerequisites
    k8s_connect
    install_csi_driver "$cloud"
    install_storageclass "$cloud"
    install_velero "$cloud"
    print_usage "$cloud"
}

main "$@"
