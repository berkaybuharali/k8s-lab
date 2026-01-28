#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# Backup Kubernetes Applications
# -----------------------------------------------------------------------------
# Creates a Velero backup of specified namespaces to cloud storage.
# Captures all Kubernetes resources (deployments, services, PVCs) and
# creates volume snapshots of persistent disks.
#
# Backup name format: <base-name>-<ddmmyyyyhhmm>
# Example: k8s-lab-backup-27012026-1430
#
# Prerequisites:
# - Cluster running with applications deployed (make apply)
# - Velero installed (done automatically by make apply)
#
# Environment Variables:
#   NAME: Base backup name (default: k8s-lab-backup)
#   NAMESPACES: Comma-separated namespaces (default: application)
#
# Usage: ./backup.sh <cloud>
# Run via: make backup <cloud>
# Examples:
#   make backup gcp
#   NAME=prod-backup make backup gcp
#   NAMESPACES=app1,app2,app3 make backup gcp
#   NAME=multi-ns NAMESPACES=app1,app2 make backup gcp
# -----------------------------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"
source "${LIB_DIR}/velero.sh"

main() {
    local cloud=$1

    validate_cloud "$cloud"
    source_cloud_modules "$cloud"

    # Export environment variables for velero.sh
    export VELERO_BACKUP_NAME="${NAME:-k8s-lab-backup}"
    export VELERO_NAMESPACES="${NAMESPACES:-application}"

    echo "=============================================="
    echo "  Kubernetes Lab - Backup (${cloud})"
    echo "=============================================="
    echo ""
    log_info "Backup base name: ${VELERO_BACKUP_NAME}"
    log_info "Namespaces: ${VELERO_NAMESPACES}"
    echo ""

    setup_error_handling
    k8s_connect
    velero_backup

    log_info "Backup complete: ${VELERO_LAST_BACKUP_NAME}"
    log_info "Restore with: make restore ${cloud}"
}

main "$@"
