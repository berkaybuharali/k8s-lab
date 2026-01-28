#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# Restore Kubernetes Applications from Backup
# -----------------------------------------------------------------------------
# Restores the application namespace from a Velero backup.
#
# Prerequisites:
# - Infrastructure deployed (make deploy-infra)
# - Cluster tools installed (make deploy-tools)
#   - CSI driver and StorageClass required for PVC restore
#   - Velero required for restore operations
#
# Usage: ./restore.sh <cloud>
# Run via: make restore <cloud>
# Full restore flow: make deploy-infra <cloud> && make deploy-tools <cloud> && make restore <cloud>
# -----------------------------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"
source "${LIB_DIR}/workloads.sh"
source "${LIB_DIR}/velero.sh"

check_prerequisites() {
    log_step "Checking prerequisites"

    if [[ ! -f "${TALOS_CONFIGS_DIR}/kubeconfig" ]]; then
        log_error "kubeconfig not found. Run 'make deploy-infra <cloud>' first"
        exit 1
    fi

    # Check if Velero is installed
    if ! kubectl get deployment velero -n velero &>/dev/null; then
        log_error "Velero not found. Run 'make deploy-tools <cloud>' first"
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
    k8s_connect
    check_prerequisites
    velero_restore
    apps_verify

    log_info "Restore complete. Applications restored from backup."
}

main "$@"
