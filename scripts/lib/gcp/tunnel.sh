#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# IAP Tunnel Functions
# -----------------------------------------------------------------------------
# Manages IAP (Identity-Aware Proxy) TCP tunnels to GCP VMs.
#
# Why IAP tunnels?
# - VMs have no external IPs (security best practice, company policy)
# - IAP authenticates via gcloud credentials
# - Traffic is encrypted and routed through Google's network
# - No need for VPN or bastion hosts
#
# How it works:
# 1. gcloud establishes tunnel from localhost to VM's internal IP
# 2. Local port maps to remote port on VM
# 3. Tools (talosctl, kubectl) connect to localhost
#
# Ports used:
# - 50000: Talos API (machine configuration, bootstrap)
# - 6443: Kubernetes API (kubectl)
# -----------------------------------------------------------------------------

# Track active tunnel PIDs for cleanup
declare -a TUNNEL_PIDS=()

# -----------------------------------------------------------------------------
# Start IAP Tunnel
# -----------------------------------------------------------------------------
# Creates a TCP tunnel to a VM via IAP.
#
# Arguments:
#   $1 - Instance name
#   $2 - Zone
#   $3 - Remote port on VM
#   $4 - Local port to bind
#
# Returns:
#   Tunnel PID (via echo, capture with $())
#
# Example:
#   tunnel_pid=$(tunnel_start "talos-cp-0" "europe-west4-a" 50000 50000)
# -----------------------------------------------------------------------------
tunnel_start() {
    local instance=$1
    local zone=$2
    local remote_port=$3
    local local_port=$4
    local max_retries=5
    local retry_delay=15
    local attempt=1
    log_step "Starting IAP tunnel. tunnel_start: instance=${instance}, zone=${zone}, remote_port=${remote_port}, local_port=${local_port}"

    while [ "${attempt}" -le "${max_retries}" ]; do
        # Start tunnel in background
        # Redirect all output to /dev/null to prevent blocking
        gcloud compute start-iap-tunnel "${instance}" "${remote_port}" \
            --local-host-port="localhost:${local_port}" \
            --zone="${zone}" \
            --project="${PROJECT_ID}" \
            >/dev/null 2>&1 &

        local pid=$!
        TUNNEL_PIDS+=("${pid}")

        # Wait for tunnel to establish
        # IAP needs time to authenticate and set up the connection
        sleep 10

        # Verify tunnel is running
        if kill -0 "${pid}" 2>/dev/null; then
            log_info "IAP Tunnel started pid:${pid}"
            echo "${pid}"
            return 0
        fi

        log_warn "IAP tunnel attempt ${attempt}/${max_retries} failed, retrying in ${retry_delay}s..."
        sleep "${retry_delay}"
        ((attempt++))
    done

    log_error "Failed to start IAP tunnel to ${instance} after ${max_retries} attempts"
    return 1
}

# -----------------------------------------------------------------------------
# Stop IAP Tunnel
# -----------------------------------------------------------------------------
# Terminates a tunnel process gracefully.
#
# Arguments:
#   $1 - Tunnel PID
# -----------------------------------------------------------------------------
tunnel_stop() {
    local pid=$1
    log_step "IAP tunnel_stop: pid=${pid}"

    if kill -0 "${pid}" 2>/dev/null; then
        kill "${pid}" 2>/dev/null || true
        wait "${pid}" 2>/dev/null || true
        log_info "Tunnel (PID ${pid}) stopped"
    fi
}

# -----------------------------------------------------------------------------
# Cleanup All Tunnels
# -----------------------------------------------------------------------------
# Stops all tunnels started during this session.
# Called automatically on script exit via trap.
# -----------------------------------------------------------------------------
tunnel_cleanup_all() {
    log_step "tunnel_cleanup_all: cleaning up ${#TUNNEL_PIDS[@]} tunnel(s)"
    for pid in "${TUNNEL_PIDS[@]:-}"; do
        if [ -n "${pid}" ]; then
            tunnel_stop "${pid}"
        fi
    done
    TUNNEL_PIDS=()
}

# Register cleanup on script exit
trap tunnel_cleanup_all EXIT
