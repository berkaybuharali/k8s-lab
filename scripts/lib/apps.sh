#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# Application Functions
# -----------------------------------------------------------------------------
# Cloud-agnostic functions for deploying and removing applications.
#
# These functions apply Kubernetes manifests that work across any cloud:
# - Namespace
# - Deployments (NGINX, Redis)
# - Services
# - PersistentVolumeClaims
#
# Cloud-specific resources (StorageClass, CSI driver) are handled by
# the cloud-specific library modules.
# -----------------------------------------------------------------------------

# -----------------------------------------------------------------------------
# Deploy Applications
# -----------------------------------------------------------------------------
# Applies all application manifests.
#
# Arguments:
#   $1 - Cloud provider (gcp, stackit, etc.) for cloud-specific manifests
# -----------------------------------------------------------------------------
apps_deploy() {
    local cloud=$1
    log_step "Deploying applications on top of Talos Kubernetes"

    local apps_dir="${REPO_ROOT}/apps"

    # Apply namespace
    log_info "Creating namespace application"
    kubectl apply -f "${apps_dir}/namespace.yaml"

    # Apply cloud-specific StorageClass
    if [[ -f "${apps_dir}/${cloud}/storageclass.yaml" ]]; then
        log_info "Applying ${cloud} StorageClass"
        kubectl apply -f "${apps_dir}/${cloud}/storageclass.yaml"
    fi

    # Deploy NGINX (stateless, 2 replicas across zones)
    log_info "Deploying NGINX"
    kubectl apply -f "${apps_dir}/nginx.yaml"

    # Deploy Redis (stateful, 1 replica with persistent storage)
    log_info "Deploying Redis with persisten storage"
    kubectl apply -f "${apps_dir}/redis.yaml"

    # Wait for deployments to be ready
    log_info "Waiting for NGINX deployment"
    kubectl rollout status deployment/nginx -n application --timeout=120s

    log_info "Waiting for Redis deployment"
    kubectl rollout status deployment/redis -n application --timeout=300s

    log_info "Applications deployed"
}

# -----------------------------------------------------------------------------
# Remove Applications
# -----------------------------------------------------------------------------
# Deletes all application resources in correct order.
# PVC deletion triggers persistent disk deletion due to reclaimPolicy: Delete.
#
# Arguments:
#   $1 - Cloud provider (gcp, stackit, etc.)
# -----------------------------------------------------------------------------
apps_remove() {
    local cloud=$1
    log_step "Removing applications"

    local apps_dir="${REPO_ROOT}/apps"

    # Check if namespace exists
    if ! kubectl get namespace application &> /dev/null; then
        log_info "Namespace 'application' not found - nothing to remove"
        return 0
    fi

    # Delete deployments first (stops pods from using PVCs)
    log_info "Deleting deployments..."
    kubectl delete deployment --all -n application --ignore-not-found --timeout=60s

    # Delete services
    log_info "Deleting services..."
    kubectl delete svc --all -n application --ignore-not-found

    # Delete PVCs (triggers disk deletion)
    log_info "Deleting PersistentVolumeClaims"
    local pvcs
    pvcs=$(kubectl get pvc -n application -o name 2>/dev/null || true)
    if [[ -n "$pvcs" ]]; then
        kubectl delete pvc --all -n application --timeout=120s
        log_info "Waiting for disks to be deleted..."
        sleep 15
    fi

    # Delete namespace
    log_info "Deleting namespace"
    kubectl delete namespace application --timeout=120s --ignore-not-found

    # Delete cloud-specific StorageClass
    if [[ -f "${apps_dir}/${cloud}/storageclass.yaml" ]]; then
        log_info "Deleting ${cloud} StorageClass..."
        kubectl delete -f "${apps_dir}/${cloud}/storageclass.yaml" --ignore-not-found
    fi

    log_info "Applications removed"
}

# -----------------------------------------------------------------------------
# Print Application Status
# -----------------------------------------------------------------------------
# Shows deployed resources for user verification.
# -----------------------------------------------------------------------------
apps_status() {
    echo ""
    echo "=============================================="
    echo "  Application Status"
    echo "=============================================="
    echo ""

    if ! kubectl get namespace application &> /dev/null; then
        log_info "No applications deployed"
        return 0
    fi

    echo "Deployments:"
    kubectl get deployments -n application
    echo ""
    echo "Pods:"
    kubectl get pods -n application -o wide
    echo ""
    echo "Services:"
    kubectl get svc -n application
    echo ""
    echo "PersistentVolumeClaims:"
    kubectl get pvc -n application
    echo ""
}
