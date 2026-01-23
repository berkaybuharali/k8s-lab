#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# Cluster Destroy
# -----------------------------------------------------------------------------
# Tears down all GCP resources and cleans up local configs.
#
# What this does:
# 1. Destroys Terraform-managed resources (VMs, firewall, VPC)
# 2. Verifies all resources are actually removed (VMs, disks)
# 3. Removes generated configs (talosconfig, kubeconfig, etc.)
#
# Terraform state (in GCS) is preserved for audit/history.
#
# Run via: make down
# -----------------------------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source library modules
source "${SCRIPT_DIR}/lib/common.sh"
source "${SCRIPT_DIR}/lib/gcp/terraform.sh"
source "${SCRIPT_DIR}/lib/gcp/verify.sh"
source "${SCRIPT_DIR}/lib/talos.sh"

# -----------------------------------------------------------------------------
# Main
# -----------------------------------------------------------------------------
main() {
    echo "=============================================="
    echo "  Kubernetes Lab - Cluster Destroy"
    echo "=============================================="
    echo ""
    log_step "main: starting cluster destroy"

    setup_error_handling

    # Get project ID before destroying (needed for verification)
    tf_get_project_id

    # Destroy infrastructure
    tf_destroy

    # Clean up generated configs
    talos_cleanup_configs

    # Verify all resources are actually destroyed
    gcp_verify_all_destroyed

    echo ""
    echo "=============================================="
    echo "  Cluster destroyed"
    echo "=============================================="
}

main "$@"