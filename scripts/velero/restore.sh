#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# Restore Kubernetes Applications from Backup
# -----------------------------------------------------------------------------
# Restores the application namespace from a Velero backup.
# Handles the full restore pipeline:
# 1. Connect to cluster via tunnel
# 2. Install CSI driver + StorageClass (required before PVC restore)
# 3. Install Velero with cloud-specific plugin
# 4. Restore from most recent backup
# 5. Verify applications and data
#
# Why CSI before restore?
# Velero restores PVCs which need a StorageClass and CSI driver to provision
# new volumes from disk snapshots. Without these, PVCs stay Pending forever.
#
# Usage: ./restore.sh <cloud>
# Run via: make restore <cloud>
# -----------------------------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"
source "${LIB_DIR}/apps.sh"
source "${LIB_DIR}/velero.sh"

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

apply_storageclass() {
    local cloud=$1
    local apps_dir="${REPO_ROOT}/apps"

    if [[ -f "${apps_dir}/${cloud}/storageclass.yaml" ]]; then
        log_info "Applying ${cloud} StorageClass"
        kubectl apply -f "${apps_dir}/${cloud}/storageclass.yaml"
    fi
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
    k8s_connect

    # Prerequisites: CSI driver and StorageClass must exist before restore
    install_csi_driver "$cloud"
    apply_storageclass "$cloud"

    # Install Velero (cloud-specific plugin + config)
    install_velero "$cloud"

    # Restore applications from backup
    velero_restore

    # Verify applications are running and data is intact
    apps_verify

    log_info "Restore complete. Applications are running."
}

main "$@"
