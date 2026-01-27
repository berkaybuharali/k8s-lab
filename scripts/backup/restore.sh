#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# Restore Kubernetes Applications from Backup
# -----------------------------------------------------------------------------
# Restores the application namespace from a Velero backup.
# Handles the full restore pipeline:
# 1. Connect to cluster via tunnel
# 2. Install cluster tools (CSI driver + StorageClass + Velero)
# 3. Restore applications from most recent backup
# 4. Verify applications and data
#
# Why tools before restore?
# Velero restores PVCs which need a StorageClass and CSI driver to provision
# new volumes from disk snapshots. Without these, PVCs stay Pending forever.
#
# Usage: ./restore.sh <cloud>
# Run via: make restore <cloud>
# -----------------------------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"
source "${LIB_DIR}/workloads.sh"
source "${LIB_DIR}/velero.sh"

install_tools() {
    local cloud=$1
    log_step "Installing cluster tools (CSI + StorageClass + Velero)"

    case "$cloud" in
        gcp)
            source "${LIB_DIR}/gcp/csi.sh"
            gcp_csi_install

            # Apply StorageClass
            local apps_dir="${REPO_ROOT}/apps"
            if [[ -f "${apps_dir}/${cloud}/storageclass.yaml" ]]; then
                log_info "Applying ${cloud} StorageClass"
                kubectl apply -f "${apps_dir}/${cloud}/storageclass.yaml"
            fi

            source "${LIB_DIR}/gcp/velero.sh"
            gcp_velero_install
            ;;
    esac

    log_info "Cluster tools installed"
}

check_prerequisites() {
    log_step "Checking prerequisites"
    if [[ ! -f "${CONFIGS_DIR}/kubeconfig" ]]; then
        log_error "kubeconfig not found. Run 'make deploy-infra <cloud>' first"
        exit 1
    fi
    log_info "Prerequisites satisfied"
}

main() {
    local cloud=$1

    validate_cloud "$cloud"
    source_cloud_modules "$cloud"

    echo "=============================================="
    echo "  Kubernetes Lab - Restore from Backup (${cloud})"
    echo "=============================================="
    echo ""

    setup_error_handling
    check_prerequisites
    k8s_connect
    install_tools "$cloud"
    velero_restore
    apps_verify

    log_info "Restore complete. Applications restored from backup."
}

main "$@"
