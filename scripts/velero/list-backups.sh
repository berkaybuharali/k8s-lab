#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# List Velero Backups
# -----------------------------------------------------------------------------
# Lists all Velero backups with their status.
#
# Prerequisites:
# - Cluster running with Velero installed (make apply)
#
# Usage: ./list-backups.sh <cloud>
# Run via: make list-backups <cloud>
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
    velero_list_backups
}

main "$@"
