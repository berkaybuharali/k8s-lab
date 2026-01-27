# Project Knowledge

## Overview
K8s lab on cloud VMs with backup. Focus: reproducibility, cost-efficiency (PoC), documentation.

## Architecture

| Layer | Description | Cloud-Agnostic |
|-------|-------------|----------------|
| 1. Infrastructure | VMs, networking (Terraform) | No |
| 2. Cluster | Talos Linux (K8s) | No |
| 3. Platform | Velero (backup/restore) | Yes |
| 4. Applications | User workloads, PostgreSQL | Yes |

Layers 3-4 reusable across clouds.

**Stack:** Terraform (state: GCS), Talos Linux, Velero, Bash/Python/Make

**Config:** `gcloud auth application-default login` or SA. 1 CP + 2 Workers (multi-AZ), smallest VMs.

## Daily Lifecycle (Cost Optimization)

| Command | Action |
|---------|--------|
| `make deploy-infra gcp` | Create VPC, firewall, VMs, bootstrap K8s |
| `make deploy-tools gcp` | Install CSI driver, StorageClass, Velero |
| `make deploy-applications gcp` | Deploy apps (NGINX, Redis) |
| `make deploy gcp` | All-in-one: infra + tools + apps |
| `make seed-redis gcp` | Insert test data into Redis |
| `make backup gcp` | Backup application namespace to GCS |
| `make restore gcp` | Install tools + restore apps from backup |
| `make destroy gcp` | Destroy all (apps + infra) |

Daily create/destroy avoids overnight costs. Configs in `configs/` (gitignored).

## Status

**Phase 1-2: COMPLETE** - Infra, Talos v1.12.1 (vanilla), K8s bootstrap via IAP tunnel  
**Phase 3: COMPLETE** - Velero installation, backup/restore via `make backup|restore gcp`
**Phase 4: IN PROGRESS** - NGINX (2 replicas, stateless), Redis (1 replica + GCE PD), PostgreSQL deployment  
**Phase 5: PLANNED** - Multi-cloud (STACKIT), cross-cluster restore

## Development Rules

**Public Repo Readiness:**
- No hardcoded project IDs, buckets, or user-specific values
- Use variables/tfvars with *.example files (e.g., terraform.tfvars.example with TODOs)

**Makefile-First:**
- All ops via `make deploy|apply|destroy <cloud>` (current: gcp only)
- Never run terraform/talosctl directly

**Terraform:**
- Document non-obvious decisions
- `terraform.tfvars` is source of truth for config

**Documentation:**
- READMEs: clear, no emojis, step-by-step
- Explain Talos/Velero concepts when implementing
- After changes, update all READMEs + CLAUDE.md + LOCAL.md

**K8s Manifests:**
- Cloud-specific: `apps/<cloud>/` (e.g., `apps/gcp/storageclass.yaml`)
- Cloud-agnostic: `apps/` (e.g., `apps/nginx.yaml`)
- Scripts in `scripts/` handle apply/remove

**Tooling:**
- Install via brew: talosctl, kubectl, jq

**Logging:**
- Use colors/functions from common.sh
- Start sh functions with step logging (intent + critical vars, no long configs/yamls)
