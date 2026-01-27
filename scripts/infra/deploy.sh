#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# Cluster Deployment
# -----------------------------------------------------------------------------
# Creates cloud infrastructure and bootstraps a Talos Kubernetes cluster.
#
# What this does:
# 1. Verifies prerequisites (cloud CLI, terraform, talosctl, kubectl, jq)
# 2. Creates cloud resources (VPC, firewall, VMs) via Terraform
# 3. Generates Talos machine configurations
# 4. Applies configs to VMs via tunnel
# 5. Bootstraps Kubernetes (initializes etcd, starts control plane)
# 6. Fetches kubeconfig for kubectl access
#
# Usage: ./deploy.sh <cloud>
# Run via: make deploy <cloud>
# -----------------------------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"

print_deploy_usage() {
    local cloud=$1
    cat <<EOF

==============================================
  Cluster is ready!
==============================================

To deploy applications:
  make apply ${cloud}

To access the cluster manually:
  make connect ${cloud}

To destroy the cluster:
  make destroy ${cloud}

EOF
}

main() {
    local cloud=$1

    validate_cloud "$cloud"
    source_cloud_modules "$cloud"
    source "${LIB_DIR}/talos.sh"

    echo "=============================================="
    echo "  Kubernetes Lab - Cluster Deployment (${cloud})"
    echo "=============================================="
    echo ""
    log_step "Starting cluster deployment for ${cloud}"

    setup_error_handling
    check_prerequisites

    # Create infrastructure
    tf_create
    tf_get_outputs

    # Wait for VMs to boot into Talos maintenance mode
    log_info "Waiting for VMs to boot into Talos maintenance mode (3 minutes)"
    sleep 180

    # Configure and bootstrap Talos
    talos_generate_configs
    talos_apply_all_configs

    log_info "Waiting for nodes to come back after config apply (1 minute)"
    sleep 60

    talos_bootstrap
    talos_fetch_kubeconfig

    print_deploy_usage "$cloud"
}

main "$@"
