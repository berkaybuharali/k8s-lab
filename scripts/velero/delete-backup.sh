#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# Delete Velero Backup
# -----------------------------------------------------------------------------
# Deletes a Velero backup by name and its associated volume snapshots.
#
# Prerequisites:
# - Cluster running with Velero installed (make apply)
#
# Usage: ./delete-backup.sh <cloud> <backup-name>
# Run via: make delete-backup <cloud> <name>
# -----------------------------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"
source "${SCRIPT_DIR}/../lib/velero.sh"

main() {
    local cloud=$1
    local backup_name="${2:-}"

    validate_cloud "$cloud"
    source_cloud_modules "$cloud"

    setup_error_handling
    k8s_connect
    velero_delete_backup "$backup_name"
}

main "$@"
