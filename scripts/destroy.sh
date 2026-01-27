#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# Cluster Destruction
# -----------------------------------------------------------------------------
# Tears down all cloud resources and cleans up local configs.
#
# What this does:
# 1. Removes applications (if cluster is accessible)
# 2. Deletes PersistentVolumeClaims (triggers disk deletion)
# 3. Destroys Terraform-managed resources (VMs, firewall, VPC)
# 4. Verifies all resources are actually removed
# 5. Removes generated configs (talosconfig, kubeconfig, etc.)
#
# Usage: ./destroy.sh <cloud>
# Run via: make destroy <cloud>
# -----------------------------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib/common.sh"

K8S_API_PORT=6443

# Attempts to gracefully remove applications before destroying infrastructure
try_remove_apps() {
    local cloud=$1
    log_step "Attempting to remove applications"

    if [[ ! -f "${CONFIGS_DIR}/kubeconfig" ]]; then
        log_info "No kubeconfig found - skipping app removal"
        return 0
    fi

    if ! tf_get_outputs 2>/dev/null; then
        log_info "Cannot read Terraform outputs - skipping app removal"
        return 0
    fi

    log_info "Starting tunnel to remove apps gracefully"
    local tunnel_pid
    tunnel_pid=$(tunnel_start "${CP_NAME}" "${CP_ZONE}" "${K8S_API_PORT}" "${K8S_API_PORT}" 2>/dev/null) || {
        log_info "Cannot start tunnel - skipping app removal"
        return 0
    }

    export KUBECONFIG="${CONFIGS_DIR}/kubeconfig"

    if ! kubectl cluster-info &>/dev/null; then
        log_info "Cannot connect to cluster - skipping app removal"
        tunnel_stop "$tunnel_pid" 2>/dev/null || true
        return 0
    fi

    source "${LIB_DIR}/apps.sh"
    apps_remove "$cloud" || log_warn "App removal encountered errors (continuing)"

    tunnel_stop "$tunnel_pid" 2>/dev/null || true
    log_info "Application removal completed"
}

verify_destruction() {
    local cloud=$1
    case "$cloud" in
        gcp)
            source "${LIB_DIR}/gcp/verify.sh"
            gcp_verify_all_destroyed
            ;;
    esac
}

main() {
    local cloud=$1

    validate_cloud "$cloud"
    source_cloud_modules "$cloud"
    source "${LIB_DIR}/talos.sh"

    echo "=============================================="
    echo "  Kubernetes Lab - Cluster Destruction (${cloud})"
    echo "=============================================="
    echo ""
    log_step "Starting cluster destruction for ${cloud}"

    setup_error_handling

    tf_get_project_id
    try_remove_apps "$cloud"
    tf_destroy
    talos_cleanup_configs
    verify_destruction "$cloud"

    echo ""
    echo "=============================================="
    echo "  Cluster destroyed"
    echo "=============================================="
}

main "$@"
