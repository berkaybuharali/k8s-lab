#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# Cluster Setup
# -----------------------------------------------------------------------------
# Creates GCP infrastructure and bootstraps a Talos Kubernetes cluster.
#
# What this does:
# 1. Verifies prerequisites (gcloud, terraform, talosctl, kubectl, jq)
# 2. Creates GCP resources (VPC, firewall, VMs) via Terraform
# 3. Generates Talos machine configurations
# 4. Applies configs to VMs via IAP tunnels
# 5. Bootstraps Kubernetes (initializes etcd, starts control plane)
# 6. Fetches kubeconfig for kubectl access
#
# Run via: make deploy
# -----------------------------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source library modules
source "${SCRIPT_DIR}/lib/common.sh"
source "${SCRIPT_DIR}/lib/gcp/terraform.sh"
source "${SCRIPT_DIR}/lib/gcp/tunnel.sh"
source "${SCRIPT_DIR}/lib/talos.sh"

# -----------------------------------------------------------------------------
# Print Usage Instructions
# -----------------------------------------------------------------------------
print_usage() {
    cat <<EOF

==============================================
  Cluster is ready!
==============================================

To access the cluster, start an IAP tunnel to the Kubernetes API:

  # Terminal 1 - Start tunnel (keep running)
  gcloud compute start-iap-tunnel ${CP_NAME} 6443 \\
    --local-host-port=localhost:6443 \\
    --zone=${CP_ZONE} \\
    --project=${PROJECT_ID}

  # Terminal 2 - Use kubectl
  export KUBECONFIG=${CONFIGS_DIR}/kubeconfig
  kubectl get nodes

To destroy the cluster:
  make down

EOF
}

# -----------------------------------------------------------------------------
# Main
# -----------------------------------------------------------------------------
main() {
    echo "=============================================="
    echo "  Kubernetes Lab - Cluster Setup"
    echo "=============================================="
    echo ""
    log_step "main: starting cluster setup"

    setup_error_handling
    check_prerequisites

    # Create infrastructure
    tf_create
    tf_get_outputs

    # Wait for VMs to boot into Talos maintenance mode
    # IAP service needs time to register newly created instances (~30s)
    # Talos needs time to fully boot into maintenance mode (~2-3 min for VM + Talos init)
    log_info "Waiting for VMs to boot into Talos maintenance mode (3 minutes)..."
    sleep 180

    # Configure and bootstrap Talos
    talos_generate_configs
    talos_apply_all_configs

    log_info "Waiting for Nodes to comeback after config apply (1 minute)..."
    sleep 60

    talos_bootstrap
    talos_fetch_kubeconfig

    print_usage
}

main "$@"
