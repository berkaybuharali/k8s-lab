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

**Stack:** Terraform (state: GCS), Talos Linux, Velero, Bash/Python/Make/Go

**Config:** `gcloud auth application-default login` or SA. 1 CP + 2 Workers (multi-AZ), smallest VMs.

## Daily Lifecycle (Cost Optimization)

Two interfaces (user choice):
- **Makefile (bash):** `make deploy-infra gcp`, `make deploy-tools gcp`, etc.
- **Go CLI (binary):** `./k8s-lab deploy-infra gcp`, `./k8s-lab deploy-tools gcp`, etc.

| Operation | Description |
|-----------|-------------|
| deploy-infra | VPC, firewall, VMs, bootstrap K8s |
| deploy-tools | CSI driver, StorageClass, Velero |
| deploy-applications | Apps (NGINX, Redis) |
| deploy | All-in-one: infra + tools + apps |
| seed-redis | Insert test data |
| backup | Backup namespace to GCS |
| restore | Install tools + restore from backup |
| destroy | Destroy all (apps + infra) |

Daily create/destroy avoids overnight costs. Configs in `configs/` (gitignored).

## Development Rules

**Dual-Agent Workflow:**
- ALWAYS read `status_dev_guideline.md` FIRST before starting work
- ALWAYS update `status_dev_guideline.md` BEFORE exiting (Active Task, Recent Accomplishments, Next Steps)
- Claude = Lead Architect (complex logic, planning)
- Gemini = Implementation Engineer (scaffolding, refactoring, docs, git ops)

**Public Repo Readiness:**
- No hardcoded project IDs, buckets, or user-specific values
- Use variables/tfvars with *.example files (e.g., terraform.tfvars.example with TODOs)
- Do not put Co-Authored-by type of lines in commit messages

**Makefile-First:**
- All ops via `make deploy|apply|destroy <cloud>` (current: gcp only)
- Never run terraform/talosctl directly

**Terraform:**
- Document non-obvious decisions
- `terraform.tfvars` is source of truth for config

**Documentation:**
- READMEs: practical and on-point, no emojis
- Brief explanations, no tutorial-style excessive commands
- Do not explain obvious methods - user should not be afraid of READMEs
- Clear pointers to examples without hand-holding
- **Exception:** Quick Start section in root README.md - only place where handholding is permitted
- After changes, update all READMEs + CLAUDE.md + LOCAL.md

**K8s Manifests:**
- Cloud-specific: `apps/<cloud>/` (e.g., `apps/gcp/storageclass.yaml`)
- Cloud-agnostic: `apps/` (e.g., `apps/nginx.yaml`)
- Scripts in `scripts/` handle apply/remove

**Tooling:**
- Install via brew: talosctl, kubectl, jq, velero

**Logging:**
- Use colors/functions from common.sh
- Start sh functions with step logging (intent + critical vars, no long configs/yamls)
