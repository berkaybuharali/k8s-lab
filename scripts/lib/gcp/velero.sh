#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# GCP-Specific Velero Installation
# -----------------------------------------------------------------------------
# Installs Velero with the GCP plugin for:
# - Backup storage in GCS (same bucket as Terraform state, velero/ prefix)
# - Volume snapshots via GCE Persistent Disk snapshots
#
# Authentication: Uses the VM service account (--no-secret).
# No JSON key file needed - VMs already have storage.objectAdmin and
# compute.storageAdmin roles attached via Terraform.
# -----------------------------------------------------------------------------

# Plugin version pinned for reproducibility
GCP_VELERO_PLUGIN="velero/velero-plugin-for-gcp:v1.11.0"

# -----------------------------------------------------------------------------
# Install Velero with GCP Plugin
# -----------------------------------------------------------------------------
# Uses bucket name, project ID, and service account from Terraform outputs.
# Uses --no-secret for workload identity via VM SA.
# -----------------------------------------------------------------------------
gcp_velero_install() {
    log_step "Installing Velero with GCP plugin"

    # Check if already installed and running
    local velero_ready
    velero_ready=$(kubectl get deployment velero -n velero -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    if [[ "$velero_ready" -ge 1 ]]; then
        log_info "Velero already running"
        return 0
    fi

    # Use bucket name from Terraform outputs (same bucket used for state)
    log_info "Using GCS bucket: ${BUCKET_NAME} (prefix: velero)"

    # Get service account email from Terraform
    local sa_email
    sa_email=$(cd "${TF_DIR}" && terraform output -raw node_service_account_email 2>/dev/null || true)

    # Build install command
    local install_args=(
        --provider gcp
        --plugins "${GCP_VELERO_PLUGIN}"
        --bucket "${BUCKET_NAME}"
        --prefix velero
        --no-secret
        --snapshot-location-config "project=${PROJECT_ID}"
    )

    # Add SA config if available (for GCS access via workload identity)
    if [[ -n "$sa_email" ]]; then
        install_args+=(--backup-location-config "serviceAccount=${sa_email}")
        log_info "Using service account: ${sa_email}"
    fi

    velero install "${install_args[@]}"

    velero_wait_ready
    log_info "Velero installed with GCP plugin"
}
