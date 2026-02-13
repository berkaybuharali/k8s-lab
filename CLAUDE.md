# Project Knowledge

## Overview
K8s lab on cloud VMs with backup, now extended with AI agent platform. Focus: reproducibility, cost-efficiency (PoC), documentation.

## Architecture

| Layer | Description | Cloud-Agnostic |
|-------|-------------|----------------|
| 1. Infrastructure | VMs, networking (Terraform) | No |
| 2. Cluster | Talos Linux (K8s) | No |
| 3. Platform | Velero (backup/restore) | Yes |
| 4. Applications | User workloads, PostgreSQL | Yes |
| 5. Agents | AI agents (Python ADK) | Yes |

Layers 3-5 reusable across clouds.

**Stack:** Terraform (state: GCS), Talos Linux, Velero, Go CLI, Python ADK

**Config:** `gcloud auth application-default login` or SA. 1 CP + 2 Workers (multi-AZ), smallest VMs.

## Daily Lifecycle (Cost Optimization)

**Interface:** Go CLI binary (`./k8s-lab` or `k8s-lab` if on PATH)

| Operation | Description |
|-----------|-------------|
| deploy-infra | VPC, firewall, VMs, bootstrap K8s |
| deploy-tools | CSI driver, StorageClass, Velero |
| deploy-applications | Apps (NGINX, Redis) |
| deploy-agents | Build + deploy AI agent containers |
| deploy | All-in-one: infra + tools + apps + agents |
| seed-redis | Insert test data |
| seed-inventory | Insert product inventory data |
| backup | Backup namespace to GCS |
| restore | Install tools + restore from backup |
| destroy | Destroy all (apps + infra) |

Daily create/destroy avoids overnight costs. Configs in `configs/` (gitignored).

## Development Rules

**Dual-Agent Workflow:**
- ALWAYS read `agent_plan.md` for agent work, `ui/plan.md` for UI work
- ALWAYS update status sections BEFORE exiting
- Claude = Lead Architect (complex logic, planning)
- Gemini = Implementation Engineer (scaffolding, refactoring, docs, git ops)
- After implementing a step: run its verification checks, wait for user code review before pushing

**Public Repo Readiness:**
- No hardcoded project IDs, buckets, or user-specific values
- Use variables/tfvars with *.example files (e.g., terraform.tfvars.example with TODOs)
- Do not put Co-Authored-by type of lines in commit messages

**CLI-First:**
- All ops via `k8s-lab <command> --cloud <cloud>` (current: gcp only)
- Never run terraform/talosctl directly
- UI build: `./build-ui.sh`

**Terraform:**
- Document non-obvious decisions
- `terraform.tfvars` is source of truth for config

**Documentation:**
- READMEs: practical and on-point, no emojis
- Brief explanations, no tutorial-style excessive commands
- Do not explain obvious methods - user should not be afraid of READMEs
- Clear pointers to examples without hand-holding
- **Exception:** Quick Start section in root README.md - only place where handholding is permitted
- After changes, update all READMEs + CLAUDE.md + LOCAL.md + ui/plan.md (status)

**K8s Manifests:**
- Cloud-specific: `apps/<cloud>/` (e.g., `apps/gcp/storageclass.yaml`)
- Cloud-agnostic: `apps/` (e.g., `apps/nginx.yaml`)
- Agent manifests: `apps/agents/`

**Tooling:**
- Install via brew: talosctl, kubectl, jq, velero

**Logging:**
- Go CLI: use `cli/pkg/logger` package
- Start functions with step logging (intent + critical vars, no long configs/yamls)

## Agent Development

**Plan:** See `agent_plan.md` (gitignored, local planning doc)

**Structure:**
- `agents/commerce/` -- Commerce Concierge (System A, port 8001)
- `agents/supply_chain/` -- Supply Chain Intelligence (System B, port 8002)
- `agents/shared/` -- Shared config + Redis client

**Tech:** Python ADK v1.25, Gemini models, A2A protocol between systems

**Conventions:**
- Each system is an independent Python package with its own pyproject.toml
- Agents communicate via A2A (HTTP POST between services)
- Tools are in `tools/` subdirectory per system
- Config via environment variables (REDIS_HOST, GCP_PROJECT_ID, etc.)
