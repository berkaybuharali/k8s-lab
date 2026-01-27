#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# GCE PD CSI Driver Functions
# -----------------------------------------------------------------------------
# Handles installation of the GCE Persistent Disk CSI driver.
#
# The CSI driver enables:
# - Dynamic provisioning of GCE Persistent Disks
# - Automatic disk attachment/detachment to nodes
# - Disk cleanup when PVCs are deleted (reclaimPolicy: Delete)
#
# Prerequisites:
# - Kubernetes cluster running
# - VMs have compute service account with disk permissions
# -----------------------------------------------------------------------------

# CSI driver kustomize overlay
# Using 'noauth' overlay - uses VM's service account directly (no workload identity)
# This is simpler for lab environments where VMs already have disk permissions
GCP_CSI_DRIVER_OVERLAY="https://github.com/kubernetes-sigs/gcp-compute-persistent-disk-csi-driver/deploy/kubernetes/overlays/noauth?ref=v1.16.1"

# -----------------------------------------------------------------------------
# Install GCE PD CSI Driver
# -----------------------------------------------------------------------------
# Installs the CSI driver using kubectl's built-in kustomize support.
# Uses the 'noauth' overlay which relies on the VM's service account.
# -----------------------------------------------------------------------------
gcp_csi_install() {
    log_step "Installing GCE PD CSI driver"

    # Check if already installed and running
    local controller_ready
    controller_ready=$(kubectl get deployment csi-gce-pd-controller -n gce-pd-csi-driver -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo "0")
    if [[ "$controller_ready" -ge 1 ]]; then
        log_info "GCE PD CSI driver already running"
        return 0
    fi

    # Create namespace with privileged label (CSI drivers need host access)
    log_info "Creating gce-pd-csi-driver namespace with privileged policy"
    kubectl apply -f - <<EOF
apiVersion: v1
kind: Namespace
metadata:
  name: gce-pd-csi-driver
  labels:
    pod-security.kubernetes.io/enforce: privileged
EOF

    # Apply CSI driver using kustomize (built into kubectl)
    log_info "Applying CSI driver manifests"
    kubectl apply -k "${GCP_CSI_DRIVER_OVERLAY}"

    # Wait for controller to be ready
    log_info "Waiting for CSI driver controller to be ready"
    kubectl rollout status deployment/csi-gce-pd-controller -n gce-pd-csi-driver --timeout=180s

    # Wait for node driver to be ready on all nodes
    log_info "Waiting for CSI driver nodes to be ready"
    kubectl rollout status daemonset/csi-gce-pd-node -n gce-pd-csi-driver --timeout=180s

    log_info "GCE PD CSI driver installed"
}

# -----------------------------------------------------------------------------
# Uninstall GCE PD CSI Driver
# -----------------------------------------------------------------------------
# Removes the CSI driver. Called during destroy if needed.
# Note: Usually not needed as cluster destruction removes everything.
# -----------------------------------------------------------------------------
gcp_csi_uninstall() {
    log_step "Uninstalling GCE PD CSI driver"

    if ! kubectl get namespace gce-pd-csi-driver &> /dev/null; then
        log_info "GCE PD CSI driver not installed"
        return 0
    fi

    kubectl delete -k "${GCP_CSI_DRIVER_OVERLAY}" --ignore-not-found

    log_info "GCE PD CSI driver uninstalled"
}
