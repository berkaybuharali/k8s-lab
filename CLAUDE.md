# Project Knowledge

## Overview
Lab environment for Kubernetes on cloud VMs with backup capabilities.
Focus on reproducibility, cost-efficiency (PoC), and clear documentation.

## Architecture Layers

| Layer | Description | Infra-Agnostic |
|-------|-------------|----------------|
| 1. Infrastructure | VMs, networking (Terraform) | No |
| 2. Cluster | Talos Linux (Kubernetes) | No |
| 3. Platform | Velero (backup/restore) | Yes |
| 4. Applications | User workloads, PostgreSQL | Yes |

Layers 3-4 are cloud-agnostic and reusable across infrastructures.

## Tech Stack
- **Infrastructure:** Terraform (state in GCS)
- **Cluster:** Talos Linux
- **Backup:** Velero
- **Scripting:** Bash/Python, Makefiles

## Configuration
- **Auth:** `gcloud auth application-default login` or Service Account
- **Cluster Spec:** 1 Control Plane, 2 Workers (different zones)
- **Resources:** Smallest/cheapest VMs suitable for PoC

## Daily Lifecycle

The cluster is designed for daily create/destroy cycles to minimize cloud costs.

| Command | Action |
|---------|--------|
| `make up` | Create VPC, firewall, VMs, bootstrap Kubernetes |
| `make down` | Destroy all resources |

This approach:
- Avoids running VMs overnight/weekends
- Keeps costs minimal for PoC/lab usage
- Ensures reproducibility (cluster is rebuilt from scratch daily)

Generated configs (talosconfig, kubeconfig) are stored in `configs/` (gitignored).

## Roadmap

### Phase 1: Foundation [COMPLETE]
- Project structure
- Terraform infrastructure
- Basic automation scripts

### Phase 2: Cluster [IN PROGRESS]
- Talos image setup [COMPLETE]
  - Image: Talos v1.12.1 uploaded to GCP as `talos-v1-12-1`
  - Source: Talos Image Factory (vanilla schematic)
- Kubernetes bootstrap
  - Generate talosctl configs (controlplane.yaml, worker.yaml)
  - Apply configs via IAP tunnel
  - Bootstrap cluster
- IAP access verification
  - Verify talosctl connectivity through IAP TCP tunnel

### Phase 3: Backup
- Velero installation
- Backup/restore verification
- Evaluate CRD design for backup policies

### Phase 4: Applications
- Deploy test services with PersistentVolumes
- PostgreSQL deployment
- Backup verification with stateful workloads

### Phase 5: Expansion
- Multi-cloud support (STACKIT)
- Restore from backup to new cluster

## Rules

These are policies to follow when developing this project:

### Code Readiness for Public Repository
- GCP Project ID, bucket names, and user-specific details must be configurable
- Use variables/tfvars, avoid hardcoding sensitive values
- Minimize repetition of environment-specific values

### Makefile Usage
- All operations should be done via Makefile targets
- Never run terraform/talosctl commands directly in normal workflow
- `make up` and `make down` are the primary entry points

### Terraform
- Document non-obvious code decisions
- All configurable values go in `terraform.tfvars`
- Variables can have defaults, but tfvars is the source of truth

### Documentation
- READMEs: Clear, no emojis, step-by-step explanations
- Explain Talos and Velero concepts when implementing
- Document customizable options

### Kubernetes Manifests
- Store all YAML files in this repository under `k8s/`
- Scripts to apply/manage manifests

### Reference
- Always consult this file when making design decisions
- This file is the source of truth for project policies

### Tool Installations
- Some tools may be needed to install like talosctl, kubectl, jq. Use brew to install

### Git Integration
- There will be public git integration. Keep a list of values which should not be pushed like GCP Project ID, GCS Bucket name, etc. 
Those values should have placeholder with TODO comments and explained in guidelines. 