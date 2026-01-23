#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# GCP Resource Verification
# -----------------------------------------------------------------------------
# Verifies GCP resources are properly created or destroyed.
# Used to confirm infrastructure state after terraform operations.
#
# Why verify?
# - Terraform may report success even if resources are in unexpected states
# - Disks may persist if auto_delete is misconfigured
# - Provides clear feedback about actual resource state
# -----------------------------------------------------------------------------

# -----------------------------------------------------------------------------
# Verify VMs Destroyed
# -----------------------------------------------------------------------------
# Checks that no Talos VMs exist in the project.
#
# Returns:
#   0 if no VMs found, 1 if VMs still exist
# -----------------------------------------------------------------------------
gcp_verify_vms_destroyed() {
    log_step "Verifying VMs are destroyed..."

    local vms
    vms=$(gcloud compute instances list \
        --project="${PROJECT_ID}" \
        --filter="name~talos-" \
        --format="value(name)" 2>/dev/null || true)

    if [ -n "${vms}" ]; then
        log_error "VMs still exist:"
        echo "${vms}" | while read -r vm; do
            echo "  - ${vm}" >&2
        done
        return 1
    fi

    log_info "All VMs destroyed"
    return 0
}

# -----------------------------------------------------------------------------
# Verify Disks Destroyed
# -----------------------------------------------------------------------------
# Checks that no Talos disks exist in the project.
# Important because orphaned disks incur costs.
#
# Returns:
#   0 if no disks found, 1 if disks still exist
# -----------------------------------------------------------------------------
gcp_verify_disks_destroyed() {
    log_step "Verifying disks are destroyed..."

    local disks
    disks=$(gcloud compute disks list \
        --project="${PROJECT_ID}" \
        --filter="name~talos-" \
        --format="value(name)" 2>/dev/null || true)

    if [ -n "${disks}" ]; then
        log_error "Disks still exist:"
        echo "${disks}" | while read -r disk; do
            echo "  - ${disk}" >&2
        done
        return 1
    fi

    log_info "All disks destroyed"
    return 0
}

# -----------------------------------------------------------------------------
# Verify All Resources Destroyed
# -----------------------------------------------------------------------------
# Comprehensive check that all GCP resources are cleaned up.
#
# Returns:
#   0 if all resources destroyed, 1 if any remain
# -----------------------------------------------------------------------------
gcp_verify_all_destroyed() {
    log_step "gcp_verify_all_destroyed: verifying all GCP resources are cleaned up"
    local failed=0

    gcp_verify_vms_destroyed || failed=1
    gcp_verify_disks_destroyed || failed=1

    if [ ${failed} -eq 0 ]; then
        log_info "All GCP resources verified destroyed"
    else
        log_warn "Some resources may still exist - check GCP console"
    fi

    return ${failed}
}

# -----------------------------------------------------------------------------
# Verify VMs Running
# -----------------------------------------------------------------------------
# Checks that expected Talos VMs are running.
# Used after infrastructure creation.
#
# Arguments:
#   $1 - Expected number of VMs
#
# Returns:
#   0 if correct number running, 1 otherwise
# -----------------------------------------------------------------------------
gcp_verify_vms_running() {
    local expected_count=${1:-3}

    log_step "Verifying VMs are running..."

    local running_count
    running_count=$(gcloud compute instances list \
        --project="${PROJECT_ID}" \
        --filter="name~talos- AND status=RUNNING" \
        --format="value(name)" 2>/dev/null | wc -l | tr -d ' ')

    if [ "${running_count}" -ne "${expected_count}" ]; then
        log_error "Expected ${expected_count} running VMs, found ${running_count}"
        return 1
    fi

    log_info "All ${expected_count} VMs are running"
    return 0
}