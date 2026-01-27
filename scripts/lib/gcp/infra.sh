#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# Terraform Functions
# -----------------------------------------------------------------------------
# Handles infrastructure creation and destruction via Terraform.
#
# What Terraform creates:
# - VPC network with custom subnet (10.0.0.0/24)
# - Firewall rules:
#   - IAP SSH access (35.235.240.0/20 -> port 22)
#   - Internal cluster traffic (all ports between nodes)
#   - Talos/Kubernetes APIs (ports 50000, 6443)
# - Compute instances:
#   - 1 control plane (talos-cp-0)
#   - 2 workers (talos-worker-0, talos-worker-1)
#
# VMs boot with Talos Linux but are unconfigured. They expose the Talos API
# on port 50000 and wait for machine configuration to be applied.
# -----------------------------------------------------------------------------

# -----------------------------------------------------------------------------
# Create Infrastructure
# -----------------------------------------------------------------------------
# Runs terraform init and apply to create all GCP resources.
# Uses -auto-approve for non-interactive execution.
# -----------------------------------------------------------------------------
tf_create() {
    log_step "Creating infrastructure with Terraform directory: ${TF_DIR}"

    cd "${TF_DIR}"

    # Ensure state bucket exists before terraform init
    tf_ensure_state_bucket

    # Initialize Terraform (downloads providers, configures backend)
    # -upgrade ensures we have latest provider versions
    terraform init -upgrade

    # Apply configuration
    # -auto-approve skips interactive confirmation (required for automation)
    terraform apply -auto-approve

    log_info "Infrastructure created"
}

# -----------------------------------------------------------------------------
# Destroy Infrastructure
# -----------------------------------------------------------------------------
# Destroys all Terraform-managed resources.
# This removes VMs, firewall rules, and VPC.
# Terraform state in GCS bucket is preserved.
# -----------------------------------------------------------------------------
tf_destroy() {
    log_step "Destroying infrastructure with Terraform... directory: ${TF_DIR}"

    cd "${TF_DIR}"

    # Check if there's anything to destroy
    if ! terraform state list &>/dev/null; then
        log_warn "No Terraform state found - nothing to destroy"
        return 0
    fi

    terraform destroy -auto-approve

    log_info "Infrastructure destroyed"
}

# -----------------------------------------------------------------------------
# Get Project ID from tfvars
# -----------------------------------------------------------------------------
# Reads project_id directly from terraform.tfvars file.
# Use this when terraform state may not exist (e.g., verification after destroy).
# -----------------------------------------------------------------------------
tf_get_project_id() {
    log_step "tf_get_project_id: reading from ${TF_DIR}/terraform.tfvars"
    PROJECT_ID=$(grep -E "^project_id" "${TF_DIR}/terraform.tfvars" | cut -d'"' -f2)
    export PROJECT_ID
    log_info "Project ID: ${PROJECT_ID}"
}

# -----------------------------------------------------------------------------
# Ensure State Bucket Exists
# -----------------------------------------------------------------------------
# Reads bucket name from backend.tf and creates it if it doesn't exist.
# Called before terraform init to ensure remote state backend is available.
# -----------------------------------------------------------------------------
tf_ensure_state_bucket() {
    log_step "tf_ensure_state_bucket: checking GCS bucket for Terraform state"

    # Read project_id from terraform.tfvars
    local project_id
    project_id=$(grep -E "^project_id" "${TF_DIR}/terraform.tfvars" | cut -d'"' -f2)

    # Read bucket name from backend.tf
    local bucket_name
    bucket_name=$(grep -E "bucket\s*=" "${TF_DIR}/backend.tf" | sed 's/.*"\(.*\)".*/\1/')

    # Read region from terraform.tfvars
    local region
    region=$(grep -E "^region" "${TF_DIR}/terraform.tfvars" | cut -d'"' -f2)

    log_info "State bucket: gs://${bucket_name} (project: ${project_id}, region: ${region})"

    # Check if bucket exists
    if gcloud storage buckets describe "gs://${bucket_name}" --project="${project_id}" &>/dev/null; then
        log_info "State bucket already exists"
        return 0
    fi

    # Create bucket
    log_info "Creating state bucket..."
    gcloud storage buckets create "gs://${bucket_name}" \
        --project="${project_id}" \
        --location="${region}" \
        --uniform-bucket-level-access

    # Enable versioning for state recovery
    gcloud storage buckets update "gs://${bucket_name}" --versioning

    log_info "State bucket created"
}

# -----------------------------------------------------------------------------
# Get Terraform Outputs
# -----------------------------------------------------------------------------
# Retrieves resource information from Terraform state.
# Sets global variables used by other scripts:
# - PROJECT_ID: GCP project
# - CP_NAME, CP_ZONE, CP_IP: Control plane details
# - WORKER_NAMES[], WORKER_ZONES[], WORKER_IPS[]: Worker details
#
# These values are needed to:
# - Construct IAP tunnel commands (name, zone, project)
# - Generate Talos configs (internal IPs for cluster endpoint)
# -----------------------------------------------------------------------------
tf_get_outputs() {
    log_step "Reading Terraform outputs..."

    cd "${TF_DIR}"

    # Single-value outputs
    PROJECT_ID=$(terraform output -raw project_id)
    CP_NAME=$(terraform output -raw control_plane_name)
    CP_ZONE=$(terraform output -raw control_plane_zone)
    CP_IP=$(terraform output -raw control_plane_ip)

    # Array outputs (parsed from JSON)
    # Using word splitting intentionally here to populate arrays
    # shellcheck disable=SC2207
    WORKER_NAMES=($(terraform output -json worker_names | jq -r '.[]'))
    # shellcheck disable=SC2207
    WORKER_ZONES=($(terraform output -json worker_zones | jq -r '.[]'))
    # shellcheck disable=SC2207
    WORKER_IPS=($(terraform output -json worker_ips | jq -r '.[]'))

    # Export for use in other scripts
    export PROJECT_ID CP_NAME CP_ZONE CP_IP
    export WORKER_NAMES WORKER_ZONES WORKER_IPS

    log_info "Control plane: ${CP_NAME} (${CP_IP}) in ${CP_ZONE}"
    for i in "${!WORKER_NAMES[@]}"; do
        log_info "Worker ${i}: ${WORKER_NAMES[$i]} (${WORKER_IPS[$i]}) in ${WORKER_ZONES[$i]}"
    done
}

# -----------------------------------------------------------------------------
# Talos Machine Config Patches
# -----------------------------------------------------------------------------
# GCP-specific patches for Talos machine configuration.
# These are applied during talos_generate_configs() in talos.sh.
#
# Current patches:
# - csi.yaml: Adds kubelet extraMounts for GCE PD CSI driver compatibility
#   (Talos has different udev paths than standard Linux)
# -----------------------------------------------------------------------------
TALOS_PATCH_FILES=(
    "${REPO_ROOT}/infra/gcp/talos-patches/csi.yaml"
)
export TALOS_PATCH_FILES
