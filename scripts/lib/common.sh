#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# Common Functions
# -----------------------------------------------------------------------------
# Shared utilities for all scripts: logging, error handling, prerequisites.
# Source this file at the beginning of other scripts.
#
# Usage:
#   source "$(dirname "${BASH_SOURCE[0]}")/lib/common.sh"
# -----------------------------------------------------------------------------

# Exit immediately on error, undefined vars, and pipe failures
set -euo pipefail

# -----------------------------------------------------------------------------
# Directory Paths
# -----------------------------------------------------------------------------
# These are set relative to the repository root for consistency.
# -----------------------------------------------------------------------------
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
CONFIGS_DIR="${REPO_ROOT}/configs"
TALOS_CONFIGS_DIR="${CONFIGS_DIR}/talos"
LIB_DIR="${SCRIPT_DIR}/lib"

# -----------------------------------------------------------------------------
# Configuration
# -----------------------------------------------------------------------------
CLUSTER_NAME="k8s-lab"
SUPPORTED_CLOUDS=("gcp")

# -----------------------------------------------------------------------------
# Cloud Provider Functions
# -----------------------------------------------------------------------------

# Validate cloud provider argument
validate_cloud() {
    local cloud=$1

    if [[ -z "$cloud" ]]; then
        log_error "Cloud provider required"
        echo "Usage: $0 <cloud>" >&2
        echo "Supported clouds: ${SUPPORTED_CLOUDS[*]}" >&2
        exit 1
    fi

    local valid=false
    for c in "${SUPPORTED_CLOUDS[@]}"; do
        [[ "$c" == "$cloud" ]] && valid=true && break
    done

    if [[ "$valid" != "true" ]]; then
        log_error "Unsupported cloud provider: $cloud"
        echo "Supported clouds: ${SUPPORTED_CLOUDS[*]}" >&2
        exit 1
    fi
}

# Source cloud-specific modules and set TF_DIR
source_cloud_modules() {
    local cloud=$1

    case "$cloud" in
        gcp)
            TF_DIR="${REPO_ROOT}/infra/gcp/terraform"
            source "${LIB_DIR}/gcp/infra.sh"
            source "${LIB_DIR}/gcp/tunnel.sh"
            ;;
        *)
            log_error "No modules for cloud: $cloud"
            exit 1
            ;;
    esac
    export TF_DIR
}

# -----------------------------------------------------------------------------
# Colors
# -----------------------------------------------------------------------------
# ANSI color codes for terminal output.
# NC (No Color) resets formatting.
# -----------------------------------------------------------------------------
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# -----------------------------------------------------------------------------
# Logging Functions
# -----------------------------------------------------------------------------
# Consistent log output with colored prefixes.
# All output goes to stderr to keep stdout clean for data.
# -----------------------------------------------------------------------------
log_info()  { echo -e "${GREEN}[INFO]${NC} $1" >&2; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC} $1" >&2; }
log_error() { echo -e "${RED}[ERROR]${NC} $1" >&2; }
log_step()  { echo -e "${BLUE}[STEP]${NC} $1" >&2; }

# -----------------------------------------------------------------------------
# Error Handling
# -----------------------------------------------------------------------------
# Trap EXIT to provide context on failures and cleanup resources.
# Scripts should call setup_error_handling at the start.
# -----------------------------------------------------------------------------
_cleanup_handler() {
    local exit_code=$?
    local line_no=$1

    # Clean up tunnels if tunnel.sh was sourced
    if type tunnel_cleanup_all &>/dev/null; then
        tunnel_cleanup_all
    fi

    if [ $exit_code -ne 0 ]; then
        log_error "Script failed at line ${line_no} with exit code ${exit_code}"
    fi
}

setup_error_handling() {
    trap '_cleanup_handler ${LINENO}' EXIT
}

# -----------------------------------------------------------------------------
# Prerequisites Check
# -----------------------------------------------------------------------------
# Verifies required tools are installed before proceeding.
# Exits with error if any tool is missing, listing all missing tools.
#
# Parameters:
#   $1 - cloud provider (optional, checks cloud-specific tools)
#
# Required tools (cloud-agnostic):
# - terraform: Infrastructure provisioning
# - talosctl: Talos Linux CLI for cluster configuration
# - kubectl: Kubernetes CLI for cluster management
# - jq: JSON parsing for Terraform outputs
# - velero: Backup/restore CLI
#
# Cloud-specific tools:
# - gcloud (GCP): For IAP tunneling
# -----------------------------------------------------------------------------
check_prerequisites() {
    local cloud="${1:-}"
    log_step "Checking prerequisites..."
    local missing=()

    # Cloud-specific tools
    case "$cloud" in
        gcp)
            if ! command -v gcloud &> /dev/null; then
                missing+=("gcloud    - https://cloud.google.com/sdk/docs/install")
            fi
            ;;
    esac

    # Cloud-agnostic tools
    if ! command -v terraform &> /dev/null; then
        missing+=("terraform - brew install terraform")
    fi

    if ! command -v talosctl &> /dev/null; then
        missing+=("talosctl  - brew install talosctl")
    fi

    if ! command -v kubectl &> /dev/null; then
        missing+=("kubectl   - brew install kubectl")
    fi

    if ! command -v jq &> /dev/null; then
        missing+=("jq        - brew install jq")
    fi

    if ! command -v velero &> /dev/null; then
        missing+=("velero    - brew install velero")
    fi

    if [ ${#missing[@]} -gt 0 ]; then
        log_error "Missing required tools:"
        for tool in "${missing[@]}"; do
            echo "  - ${tool}" >&2
        done
        exit 1
    fi

    log_info "All prerequisites satisfied"
}

# -----------------------------------------------------------------------------
# Utility Functions
# -----------------------------------------------------------------------------

# Wait with a simple progress indicator
wait_with_dots() {
    local seconds=$1
    local message=${2:-"Waiting"}

    log_info "${message} (${seconds}s)..."
    for ((i=0; i<seconds; i+=5)); do
        sleep 5
        printf "." >&2
    done
    echo "" >&2
}
