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
# -----------------------------------------------------------------------------

VELERO_BACKUP_NAME="k8s-lab-backup"
VELERO_NAMESPACE="application"

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
# Creates a Velero backup of the application namespace.
# Includes all resources (deployments, services, PVCs) and volume snapshots.
#
# --include-namespaces: Only backs up the application namespace
# --wait: Blocks until backup completes or fails
# --default-volumes-to-fs-backup=false: Uses native volume snapshots (GCE PD)
# -----------------------------------------------------------------------------
velero_backup() {
    log_step "Creating Velero backup: ${VELERO_BACKUP_NAME}"

    # Delete previous backup with same name if exists
    if velero backup get "${VELERO_BACKUP_NAME}" &>/dev/null; then
        log_info "Deleting previous backup: ${VELERO_BACKUP_NAME}"
        velero backup delete "${VELERO_BACKUP_NAME}" --confirm
        # Wait for deletion to complete
        local retries=30
        while velero backup get "${VELERO_BACKUP_NAME}" &>/dev/null && [ "$retries" -gt 0 ]; do
            sleep 2
            ((retries--))
        done
    fi

    velero backup create "${VELERO_BACKUP_NAME}" \
        --include-namespaces "${VELERO_NAMESPACE}" \
        --wait

    velero_verify_backup
}

# -----------------------------------------------------------------------------
# Verify Backup
# -----------------------------------------------------------------------------
# Checks that the backup completed successfully.
# -----------------------------------------------------------------------------
velero_verify_backup() {
    log_step "Verifying backup: ${VELERO_BACKUP_NAME}"

    local status
    status=$(velero backup get "${VELERO_BACKUP_NAME}" -o json | jq -r '.status.phase')

    if [[ "$status" != "Completed" ]]; then
        log_error "Backup failed with status: ${status}"
        velero backup describe "${VELERO_BACKUP_NAME}" --details
        return 1
    fi

    log_info "Backup verified: ${status}"
    velero backup describe "${VELERO_BACKUP_NAME}"
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

    if ! velero backup get "${backup_name}" &>/dev/null; then
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
    backups=$(velero backup get -o json | jq -r '.items[].metadata.name' 2>/dev/null || true)

    if [[ -z "$backups" ]]; then
        log_info "No backups found"
        return 0
    fi

    log_info "Found backups:"
    echo "$backups" >&2

    velero backup delete --all --confirm
    log_info "All backups deleted"
}

velero_restore() {
    log_step "Restoring from backup: ${VELERO_BACKUP_NAME}"

    velero restore create \
        --from-backup "${VELERO_BACKUP_NAME}" \
        --wait

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
    restore_name=$(velero restore get -o json | jq -r '.items[-1].metadata.name')

    if [[ -z "$restore_name" || "$restore_name" == "null" ]]; then
        log_error "No restore found"
        return 1
    fi

    local status
    status=$(velero restore get "${restore_name}" -o json | jq -r '.status.phase')

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
