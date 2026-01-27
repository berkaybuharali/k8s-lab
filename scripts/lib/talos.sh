#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# Talos Functions
# -----------------------------------------------------------------------------
# Handles Talos Linux configuration and Kubernetes cluster bootstrap.
#
# Talos Linux overview:
# - Immutable OS designed for Kubernetes
# - No SSH, no shell - only API access (port 50000)
# - Configuration applied via talosctl, not config files
# - Kubernetes components are built-in, just need to bootstrap
#
# Workflow:
# 1. Generate configs (controlplane.yaml, worker.yaml, talosconfig)
# 2. Apply configs to each node via Talos API
# 3. Bootstrap etcd on control plane (one-time, initializes cluster)
# 4. Fetch kubeconfig for kubectl access
# -----------------------------------------------------------------------------

# -----------------------------------------------------------------------------
# Generate Talos Configs
# -----------------------------------------------------------------------------
# Creates machine configuration files using talosctl.
#
# Generated files:
# - controlplane.yaml: Config for control plane nodes
#   - Runs: etcd, kube-apiserver, controller-manager, scheduler
# - worker.yaml: Config for worker nodes
#   - Runs: kubelet only
# - talosconfig: Client authentication config
#   - Contains endpoints and credentials for talosctl
#
# The endpoint URL (https://<CP_IP>:6443) is the Kubernetes API address.
# Workers use this to find and join the cluster.
# -----------------------------------------------------------------------------
talos_generate_configs() {
    log_step "Generating Talos machine configurations..."

    mkdir -p "${CONFIGS_DIR}"

    # Build patch flags from cloud-specific patches (set by cloud modules)
    # TALOS_PATCH_FILES is an array of patch file paths, may be unset for some clouds
    local patch_args=()
    if [[ ${#TALOS_PATCH_FILES[@]:-0} -gt 0 ]]; then
        for patch_file in "${TALOS_PATCH_FILES[@]}"; do
            if [[ -f "${patch_file}" ]]; then
                log_info "Applying patch: ${patch_file}"
                patch_args+=("--config-patch=@${patch_file}")
            else
                log_warn "Patch file not found: ${patch_file}"
            fi
        done
    fi

    # Generate configs with control plane IP as Kubernetes endpoint
    # --force: Overwrite existing (needed for daily recreate workflow)
    # --additional-sans localhost: Required for IAP tunnel access (we connect via localhost)
    talosctl gen config "${CLUSTER_NAME}" "https://${CP_IP}:6443" \
        --output-dir "${CONFIGS_DIR}" \
        --additional-sans localhost \
        --force \
        "${patch_args[@]}"

    log_info "Generated:"
    log_info "  ${CONFIGS_DIR}/controlplane.yaml"
    log_info "  ${CONFIGS_DIR}/worker.yaml"
    log_info "  ${CONFIGS_DIR}/talosconfig"
}

# -----------------------------------------------------------------------------
# Apply Config to Node
# -----------------------------------------------------------------------------
# Sends machine configuration to a Talos node via IAP tunnel.
#
# After receiving config, Talos will:
# 1. Configure networking
# 2. Start containerd and kubelet
# 3. Wait for bootstrap (control plane) or join cluster (workers)
#
# Arguments:
#   $1 - Config file path
#   $2 - Local tunnel port
# -----------------------------------------------------------------------------
talos_apply_config() {
    local config_file=$1
    local port=$2
    log_step "talos_apply_config: port=${port}"

    # --insecure: Node doesn't trust us yet (no config applied)
    talosctl apply-config \
        --nodes "localhost" \
        --endpoints "localhost:${port}" \
        --file "${config_file}" \
        --insecure
}

# -----------------------------------------------------------------------------
# Apply Config to Single Node
# -----------------------------------------------------------------------------
# Applies configuration to a single node, managing IAP tunnel lifecycle.
# Nodes are in maintenance mode on first boot - API responds immediately.
#
# Arguments:
#   $1 - Node name (instance name)
#   $2 - Zone
#   $3 - Config file path
#   $4 - Node type label (for logging, e.g., "control-plane", "worker-0")
# -----------------------------------------------------------------------------
talos_apply_node_config() {
    local node_name=$1
    local zone=$2
    local config_file=$3
    local node_label=$4
    log_step "talos_apply_node_config: node=${node_name}, zone=${zone}, label=${node_label}"

    local tunnel_pid
    tunnel_pid=$(tunnel_start "${node_name}" "${zone}" 50000 50000)

    talos_apply_config "${config_file}" 50000

    log_info "${node_label} (${node_name}) configured"
    tunnel_stop "${tunnel_pid}"
}

# -----------------------------------------------------------------------------
# Apply Config to Control Plane
# -----------------------------------------------------------------------------
# Configures the control plane node via IAP tunnel.
# -----------------------------------------------------------------------------
talos_apply_cp_config() {
    log_step "talos_apply_cp_config: CP_NAME=${CP_NAME}, CP_ZONE=${CP_ZONE}"

    local cp_config="${CONFIGS_DIR}/controlplane.yaml"
    talos_apply_node_config "${CP_NAME}" "${CP_ZONE}" "${cp_config}" "control-plane"
}

# -----------------------------------------------------------------------------
# Apply Config to All Workers
# -----------------------------------------------------------------------------
# Configures all worker nodes via IAP tunnels.
# Opens one tunnel at a time to avoid port conflicts.
# -----------------------------------------------------------------------------
talos_apply_worker_configs() {
    local worker_count=${#WORKER_NAMES[@]}
    log_step "talos_apply_worker_configs: worker_count=${worker_count}"

    local worker_config="${CONFIGS_DIR}/worker.yaml"

    for i in "${!WORKER_NAMES[@]}"; do
        talos_apply_node_config "${WORKER_NAMES[$i]}" "${WORKER_ZONES[$i]}" "${worker_config}" "worker-${i}"
    done
}

# -----------------------------------------------------------------------------
# Apply Configs to All Nodes
# -----------------------------------------------------------------------------
# Configures control plane and all workers via IAP tunnels.
# Opens one tunnel at a time to avoid port conflicts.
# -----------------------------------------------------------------------------
talos_apply_all_configs() {
    log_step "talos_apply_all_configs: applying to control plane and workers"

    talos_apply_cp_config
    talos_apply_worker_configs

    log_info "All nodes configured"
}

# -----------------------------------------------------------------------------
# Wait for Talos API (Authenticated Mode)
# -----------------------------------------------------------------------------
# After config is applied, node reboots into running mode.
# This waits for the API to be accessible with proper authentication.
#
# Arguments:
#   $1 - talosconfig path
#   $2 - Max attempts (default: 30, ~5 minutes with 10s interval)
# -----------------------------------------------------------------------------
talos_wait_for_api_ready() {
    local talosconfig=$1
    local max_attempts=${2:-30}
    local attempt=1
    log_step "Waiting Talos API (authenticated mode) talos_wait_for_api_ready: max_attempts=${max_attempts}"

    while [ "${attempt}" -le "${max_attempts}" ]; do
        # Try to get version with authentication - this confirms node is running
        if talosctl version \
            --talosconfig="${talosconfig}" \
            --nodes "localhost" \
            --endpoints "localhost:50000" &>/dev/null; then
            log_info "Talos API ready (authenticated)"
            return 0
        fi

        printf "." >&2
        sleep 10
        ((attempt++))
    done

    echo "" >&2
    log_error "Talos API not ready after $((max_attempts * 10)) seconds"
    return 1
}

# -----------------------------------------------------------------------------
# Bootstrap Cluster
# -----------------------------------------------------------------------------
# Initializes Kubernetes by bootstrapping etcd on control plane.
#
# What bootstrap does:
# 1. Initializes etcd (distributed key-value store for Kubernetes)
# 2. Generates cluster PKI (certificates for secure communication)
# 3. Starts kube-apiserver, controller-manager, scheduler
# 4. Workers automatically join via kubelet once API is available
#
# IMPORTANT: Run only ONCE on ONE control plane node.
# Running again will corrupt the cluster state.
# -----------------------------------------------------------------------------
talos_bootstrap() {
    log_step "Bootstrapping Kubernetes cluster..."

    local talosconfig="${CONFIGS_DIR}/talosconfig"

    # Start tunnel to control plane
    local tunnel_pid
    tunnel_pid=$(tunnel_start "${CP_NAME}" "${CP_ZONE}" 50000 50000)

    # Wait for Talos to finish rebooting after config was applied
    # Node goes: maintenance mode -> reboot -> running mode
    talos_wait_for_api_ready "${talosconfig}"

    # Bootstrap - initializes etcd and starts Kubernetes
    log_info "Initiating cluster bootstrap..."
    talosctl bootstrap \
        --talosconfig="${talosconfig}" \
        --nodes "localhost" \
        --endpoints "localhost:50000"

    log_info "Bootstrap initiated, waiting for cluster health..."

    # Poll health endpoint with longer timeout
    # Checks: etcd, kubelet, apiserver, controller-manager, scheduler
    local max_attempts=60
    local attempt=1

    while [ "${attempt}" -le "${max_attempts}" ]; do
        if talosctl health \
            --talosconfig="${talosconfig}" \
            --nodes "localhost" \
            --endpoints "localhost:50000" \
            --wait-timeout=30s &>/dev/null; then
            log_info "Control plane is healthy"
            tunnel_stop "${tunnel_pid}"
            return 0
        fi

        printf "." >&2
        sleep 10
        ((attempt++))
    done

    echo "" >&2
    log_warn "Health check timed out - cluster may still be initializing"
    tunnel_stop "${tunnel_pid}"
}

# -----------------------------------------------------------------------------
# Wait for Kubernetes API
# -----------------------------------------------------------------------------
# Waits for the Kubernetes API to be accessible.
#
# Arguments:
#   $1 - kubeconfig path
#   $2 - Max attempts (default: 30, ~5 minutes with 10s interval)
# -----------------------------------------------------------------------------
talos_wait_for_kubernetes_api() {
    local kubeconfig=$1
    local max_attempts=${2:-30}
    local attempt=1
    log_step "talos_wait_for_kubernetes_api: max_attempts=${max_attempts}"

    log_info "Waiting for Kubernetes API to be ready..."

    while [ "${attempt}" -le "${max_attempts}" ]; do
        if kubectl --kubeconfig="${kubeconfig}" get nodes &>/dev/null; then
            log_info "Kubernetes API is ready"
            return 0
        fi

        printf "." >&2
        sleep 10
        ((attempt++))
    done

    echo "" >&2
    log_error "Kubernetes API not ready after $((max_attempts * 10)) seconds"
    return 1
}

# -----------------------------------------------------------------------------
# Wait for All Nodes
# -----------------------------------------------------------------------------
# Waits for expected number of nodes to be Ready in the cluster.
#
# Arguments:
#   $1 - kubeconfig path
#   $2 - Expected node count
#   $3 - Max attempts (default: 60, ~10 minutes with 10s interval)
# -----------------------------------------------------------------------------
talos_wait_for_all_nodes() {
    local kubeconfig=$1
    local expected_nodes=$2
    local max_attempts=${3:-60}
    local attempt=1
    log_step "talos_wait_for_all_nodes: expected_nodes=${expected_nodes}, max_attempts=${max_attempts}"

    while [ "${attempt}" -le "${max_attempts}" ]; do
        # Count nodes that are Ready
        local ready_nodes
        ready_nodes=$(kubectl --kubeconfig="${kubeconfig}" get nodes \
            --no-headers 2>/dev/null | grep -c " Ready" || echo "0")

        if [ "${ready_nodes}" -ge "${expected_nodes}" ]; then
            log_info "All ${expected_nodes} nodes are Ready"
            return 0
        fi

        printf "." >&2
        sleep 10
        ((attempt++))
    done

    echo "" >&2
    log_warn "Only ${ready_nodes}/${expected_nodes} nodes Ready after $((max_attempts * 10)) seconds"
    return 1
}

# -----------------------------------------------------------------------------
# Fetch Kubeconfig
# -----------------------------------------------------------------------------
# Retrieves kubeconfig from cluster for kubectl access.
#
# The kubeconfig contains:
# - Cluster CA certificate
# - Admin client certificate
# - API server endpoint (internal IP - needs tunnel for access)
# -----------------------------------------------------------------------------
talos_fetch_kubeconfig() {
    log_step "Fetching kubeconfig..."

    local talosconfig="${CONFIGS_DIR}/talosconfig"
    local kubeconfig="${CONFIGS_DIR}/kubeconfig"

    local tunnel_pid
    tunnel_pid=$(tunnel_start "${CP_NAME}" "${CP_ZONE}" 50000 50000)

    talosctl kubeconfig "${kubeconfig}" \
        --talosconfig="${talosconfig}" \
        --nodes "localhost" \
        --endpoints "localhost:50000" \
        --force

    tunnel_stop "${tunnel_pid}"

    log_info "Kubeconfig saved: ${kubeconfig}"

    # Modify kubeconfig to use localhost (for IAP tunnel access)
    # The generated kubeconfig points to the internal IP, but we access via tunnel
    log_info "Updating kubeconfig to use localhost endpoint..."
    sed -i.bak "s|server: https://${CP_IP}:6443|server: https://localhost:6443|g" "${kubeconfig}"
    rm -f "${kubeconfig}.bak"
}

# -----------------------------------------------------------------------------
# Cleanup Configs
# -----------------------------------------------------------------------------
# Removes generated config files.
# Called during cluster destruction.
# -----------------------------------------------------------------------------
talos_cleanup_configs() {
    log_step "talos_cleanup_configs: CONFIGS_DIR=${CONFIGS_DIR}"
    if [ -d "${CONFIGS_DIR}" ]; then
        log_info "Removing generated configs..."
        rm -rf "${CONFIGS_DIR}"
    fi
}
