#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# Backup Kubernetes Applications
# -----------------------------------------------------------------------------
# Creates a Velero backup of the application namespace to cloud storage.
# Captures all Kubernetes resources (deployments, services, PVCs) and
# creates volume snapshots of persistent disks.
#
# Prerequisites:
# - Cluster running with applications deployed (make apply)
# - Velero installed (done automatically by make apply)
#
# Usage: ./backup.sh <cloud>
# Run via: make backup <cloud>
# -----------------------------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"
source "${SCRIPT_DIR}/../lib/velero.sh"

main() {
    local cloud=$1

    validate_cloud "$cloud"
    source_cloud_modules "$cloud"

    echo "=============================================="
    echo "  Kubernetes Lab - Backup (${cloud})"
    echo "=============================================="
    echo ""

    setup_error_handling
    k8s_connect
    velero_backup

    log_info "Backup complete. Restore with: make restore ${cloud}"
}

main "$@"
