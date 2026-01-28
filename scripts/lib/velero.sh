#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# Velero Functions (Cloud-Agnostic)
# -----------------------------------------------------------------------------
# Backup and restore operations that work with any cloud provider.
# Cloud-specific installation is handled by <cloud>/velero.sh modules.
#
# Velero concepts:
# - Backup: Captures Kubernetes resource manifests + persistent volume snapshots
# - Restore: Recreates resources from a backup, rebinds PVCs to new volumes
# - BackupStorageLocation (BSL): Where backup data is stored (e.g., GCS bucket)
# - VolumeSnapshotLocation (VSL): Where volume snapshots are taken (e.g., GCE disks)
#
# Note: Many functions use kubectl instead of velero CLI to query backup/restore
# status to avoid client-side rate limiter errors. Velero CLI has hardcoded
# client-go rate limits (5 QPS / 10 burst) that cause "Wait would exceed context
# deadline" errors with high-latency connections like IAP tunnels.
# -----------------------------------------------------------------------------

# Default values (can be overridden by environment variables)
VELERO_BACKUP_BASE_NAME="${VELERO_BACKUP_NAME:-k8s-lab-backup}"
VELERO_NAMESPACES="${VELERO_NAMESPACES:-application}"

# Auto-detect hooks files if they exist (no manual export needed)
# Override by setting VELERO_BACKUP_HOOKS_FILE or VELERO_RESTORE_HOOKS_FILE
DEFAULT_BACKUP_HOOKS="${CONFIGS_DIR}/velero/backup-hooks.yaml"
DEFAULT_RESTORE_HOOKS="${CONFIGS_DIR}/velero/restore-hooks.yaml"

if [[ -z "${VELERO_BACKUP_HOOKS_FILE:-}" ]] && [[ -f "${DEFAULT_BACKUP_HOOKS}" ]]; then
    VELERO_BACKUP_HOOKS_FILE="${DEFAULT_BACKUP_HOOKS}"
else
    VELERO_BACKUP_HOOKS_FILE="${VELERO_BACKUP_HOOKS_FILE:-}"
fi

if [[ -z "${VELERO_RESTORE_HOOKS_FILE:-}" ]] && [[ -f "${DEFAULT_RESTORE_HOOKS}" ]]; then
    VELERO_RESTORE_HOOKS_FILE="${DEFAULT_RESTORE_HOOKS}"
else
    VELERO_RESTORE_HOOKS_FILE="${VELERO_RESTORE_HOOKS_FILE:-}"
fi

# -----------------------------------------------------------------------------
# Wait for Velero to be Ready
# -----------------------------------------------------------------------------
# Waits for the Velero deployment to have ready replicas.
# Called after install to ensure Velero can accept backup/restore commands.
# -----------------------------------------------------------------------------
velero_wait_ready() {
    log_step "Waiting for Velero deployment to be ready"
    kubectl rollout status deployment/velero -n velero --timeout=120s
    log_info "Velero is ready"
}

# -----------------------------------------------------------------------------
# Create Backup
# -----------------------------------------------------------------------------
# Creates a Velero backup with automatic timestamp suffix.
# Supports custom backup name, namespaces, and optional hooks file.
#
# Environment variables:
#   VELERO_BACKUP_NAME: Base name (default: k8s-lab-backup)
#   VELERO_NAMESPACES: Comma-separated namespaces (default: application)
#   VELERO_BACKUP_HOOKS_FILE: Path to backup hooks YAML (optional)
#
# The backup name will always have -ddmmyyyyhhmm timestamp appended.
# Example: k8s-lab-backup-27012026-1430
#
# --include-namespaces: Backs up specified namespaces
# --wait: Blocks until backup completes or fails
# --hooks-file: Centralized backup hooks (optional, pod annotations still work)
#
# Note: Pod annotations take precedence over hooks file.
# Redis already has backup hooks via annotations (apps/redis.yaml).
# -----------------------------------------------------------------------------
velero_backup() {
    # Generate timestamp: ddmmyyyyhhmm (UTC)
    local timestamp
    timestamp=$(date -u +"%d%m%Y-%H%M")

    # Construct backup name with timestamp
    local backup_name="${VELERO_BACKUP_BASE_NAME}-${timestamp}"

    log_step "Creating Velero backup: ${backup_name}"
    log_info "Namespaces: ${VELERO_NAMESPACES}"

    # Build backup command
    local backup_cmd=(velero backup create "${backup_name}"
        --include-namespaces "${VELERO_NAMESPACES}"
        --wait)

    # Add hooks file if provided
    if [[ -n "${VELERO_BACKUP_HOOKS_FILE}" ]]; then
        if [[ ! -f "${VELERO_BACKUP_HOOKS_FILE}" ]]; then
            log_error "Backup hooks file not found: ${VELERO_BACKUP_HOOKS_FILE}"
            return 1
        fi
        log_info "Using backup hooks: ${VELERO_BACKUP_HOOKS_FILE}"
        backup_cmd+=(--hooks-file "${VELERO_BACKUP_HOOKS_FILE}")
    fi

    # Execute backup
    "${backup_cmd[@]}"

    velero_verify_backup "${backup_name}"

    # Export for other functions to use
    export VELERO_LAST_BACKUP_NAME="${backup_name}"
}

# -----------------------------------------------------------------------------
# Verify Backup
# -----------------------------------------------------------------------------
# Checks that the backup completed successfully.
# Parameters:
#   $1: backup_name (optional, uses VELERO_LAST_BACKUP_NAME if not provided)
# -----------------------------------------------------------------------------
velero_verify_backup() {
    local backup_name="${1:-${VELERO_LAST_BACKUP_NAME}}"

    log_step "Verifying backup: ${backup_name}"

    local status
    status=$(kubectl get backup "${backup_name}" -n velero -o jsonpath='{.status.phase}')

    if [[ "$status" != "Completed" ]]; then
        log_error "Backup failed with status: ${status}"
        velero backup describe "${backup_name}" --details
        return 1
    fi

    log_info "Backup verified: ${status}"
    velero backup describe "${backup_name}"
}

# -----------------------------------------------------------------------------
# Restore from Backup
# -----------------------------------------------------------------------------
# Restores the application namespace from the most recent backup.
# Velero recreates all resources and rebinds PVCs to new volumes
# from disk snapshots.
# -----------------------------------------------------------------------------
# -----------------------------------------------------------------------------
# List Backups
# -----------------------------------------------------------------------------
# Lists all Velero backups with their status.
# -----------------------------------------------------------------------------
velero_list_backups() {
    log_step "Listing Velero backups"
    velero backup get
}

# -----------------------------------------------------------------------------
# Delete Backup
# -----------------------------------------------------------------------------
# Deletes a Velero backup by name. If no name is provided, deletes the
# default backup (k8s-lab-backup). Also removes associated volume snapshots.
# -----------------------------------------------------------------------------
velero_delete_backup() {
    local backup_name="$1"

    if [[ -z "$backup_name" ]]; then
        log_error "Backup name required. Usage: make delete-backup <cloud> <name>"
        log_info "List backups with: make list-backups <cloud>"
        return 1
    fi

    log_step "Deleting Velero backup: ${backup_name}"

    if ! kubectl get backup "${backup_name}" -n velero &>/dev/null; then
        log_error "Backup not found: ${backup_name}"
        return 1
    fi

    velero backup delete "${backup_name}" --confirm
    log_info "Backup deleted: ${backup_name}"
}

# -----------------------------------------------------------------------------
# Delete All Backups
# -----------------------------------------------------------------------------
# Deletes all Velero backups and their associated volume snapshots.
# -----------------------------------------------------------------------------
velero_delete_all_backups() {
    log_step "Deleting all Velero backups"

    local backups
    backups=$(kubectl get backup -n velero -o json | jq -r '.items[].metadata.name' 2>/dev/null || true)

    if [[ -z "$backups" ]]; then
        log_info "No backups found"
        return 0
    fi

    log_info "Found backups:"
    echo "$backups" >&2

    velero backup delete --all --confirm
    log_info "All backups deleted"
}

# -----------------------------------------------------------------------------
# Restore from Backup
# -----------------------------------------------------------------------------
# Restores from the most recent backup (or specified backup name).
# Supports optional restore hooks for post-restore validation.
#
# Parameters:
#   $1: backup_name (optional, uses latest if not provided)
#
# Environment variables:
#   VELERO_RESTORE_HOOKS_FILE: Path to restore hooks YAML (optional)
#
# Velero recreates all resources and rebinds PVCs to new volumes from snapshots.
# Post-restore hooks run after pods are running for validation.
# -----------------------------------------------------------------------------
velero_restore() {
    local backup_name="${1:-}"

    # If no backup name provided, find the latest successful backup
    if [[ -z "$backup_name" ]]; then
        backup_name=$(kubectl get backup -n velero -o json | jq -r '.items | sort_by(.status.completionTimestamp) | reverse | .[0].metadata.name')
        if [[ -z "$backup_name" || "$backup_name" == "null" ]]; then
            log_error "No backups found. Please specify a backup name."
            return 1
        fi
        log_info "Using latest backup: ${backup_name}"
    fi

    log_step "Restoring from backup: ${backup_name}"

    # Build restore command
    local restore_cmd=(velero restore create
        --from-backup "${backup_name}"
        --wait)

    # Add hooks file if provided
    if [[ -n "${VELERO_RESTORE_HOOKS_FILE}" ]]; then
        if [[ ! -f "${VELERO_RESTORE_HOOKS_FILE}" ]]; then
            log_error "Restore hooks file not found: ${VELERO_RESTORE_HOOKS_FILE}"
            return 1
        fi
        log_info "Using restore hooks: ${VELERO_RESTORE_HOOKS_FILE}"
        restore_cmd+=(--hooks-file "${VELERO_RESTORE_HOOKS_FILE}")
    fi

    # Execute restore
    "${restore_cmd[@]}"

    velero_verify_restore
}

# -----------------------------------------------------------------------------
# Verify Restore
# -----------------------------------------------------------------------------
# Checks Velero restore CR status only. Does not verify application state.
# For application verification, use apps_verify() from apps.sh.
# -----------------------------------------------------------------------------
velero_verify_restore() {
    log_step "Verifying Velero restore status"

    # Get most recent restore
    local restore_name
    restore_name=$(kubectl get restore -n velero -o json | jq -r '.items | sort_by(.metadata.creationTimestamp) | reverse | .[0].metadata.name')

    if [[ -z "$restore_name" || "$restore_name" == "null" ]]; then
        log_error "No restore found"
        return 1
    fi

    local status
    status=$(kubectl get restore "${restore_name}" -n velero -o jsonpath='{.status.phase}')

    case "$status" in
        Completed)
            log_info "Restore status: ${status}"
            ;;
        PartiallyFailed)
            log_warn "Restore status: ${status} (may be OK if PVs were recreated)"
            velero restore describe "${restore_name}" --details
            ;;
        *)
            log_error "Restore status: ${status}"
            velero restore describe "${restore_name}" --details
            return 1
            ;;
    esac
}
