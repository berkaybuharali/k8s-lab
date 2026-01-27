#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# Delete All Velero Backups
# -----------------------------------------------------------------------------
# Deletes all Velero backups and their associated volume snapshots.
#
# Prerequisites:
# - Cluster running with Velero installed (make apply)
#
# Usage: ./delete-all-backups.sh <cloud>
# Run via: make delete-all-backups <cloud>
# -----------------------------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"
source "${SCRIPT_DIR}/../lib/velero.sh"

main() {
    local cloud=$1

    validate_cloud "$cloud"
    source_cloud_modules "$cloud"

    setup_error_handling
    k8s_connect
    velero_delete_all_backups
}

main "$@"
