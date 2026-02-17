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
| deploy-applications | Apps (NGINX, Redis) + agents (build, push, deploy) |
| deploy-agents | Build + deploy AI agent containers (standalone, also called by deploy-applications) |
| deploy | All-in-one: infra + tools + applications (incl. agents) |
| seed-data | Seed all test data: inventory, fake orders with images |
| backup | Backup namespace to GCS |
| restore | Install tools + restore from backup |
| destroy | Destroy all (apps + infra) |

Daily create/destroy avoids overnight costs. Configs in `configs/` (gitignored).

## Development Rules

**Dual-Agent Workflow:**
- ALWAYS read `agent_plan.md` for work
- ALWAYS update status sections BEFORE exiting
- Claude = Lead Architect (complex logic, planning)
- Gemini = Implementation Engineer (scaffolding, refactoring, docs, git ops)
- After implementing a step: run its verification checks, wait for user code review before pushing

**Public Repo Readiness:**
- No hardcoded project IDs, buckets, or user-specific values
- Use variables/tfvars with *.example files (e.g., terraform.tfvars.example with TODOs)
- Do not put Co-Authored-by type of lines in commit messages

**CLI-First:**
- All ops via `./bin/k8s-lab <command> --cloud <cloud>` (current: gcp only)
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
- Install via brew: talosctl, kubectl, jq, velero, gcloud

**Logging:**
- Go CLI: use `cli/pkg/logger` package
- Start functions with step logging (intent + critical vars, no long configs/yamls)

## Agent Development -- Magic Cake Shop

**Plan:** See `agent_plan.md` (gitignored, local planning doc)

**Domain:** Magic Cake -- conversational cake ordering, Amsterdam-only delivery, Gemini-generated cake images.

**Three Protocols:**
- **A2A** (Agent-to-Agent): Commerce ↔ Supply Chain cross-system calls (check stock, create orders)
- **MCP** (Model Context Protocol): Fulfillment agent uses Google Maps MCP for route optimization
- **UCP** (Universal Commerce Protocol): Agentic storefront -- external AI agents discover and order cakes via REST

**Structure:**
- `agents/commerce/` -- Commerce Concierge (System A, port 8001): Translation, Cake Designer, Checkout + UCP endpoints
- `agents/supply_chain/` -- Supply Chain Intelligence (System B, port 8002): Inventory, Order Service, Fulfillment
- `agents/shared/` -- Shared config + Redis client

**Tech:** Python ADK v1.25, Gemini models (2.5 Flash Image for cake generation)

**Pricing:** 5 EUR per slice (per person). Min 6, max 50 per cake. Delivery: 5 EUR if total < 50 EUR, free otherwise. Multiple cakes per order allowed.

**Inventory items:** chocolate, ananas, banana, walnut, almond (max 5 per type)

**Image storage:** Existing GCS bucket `{project}-k8s-lab` under `cakes/orders/{order-id}/cake_N.png` (cake_1.png, cake_2.png for multi-cake orders)

**Conventions:**
- Each system is an independent Python package with its own pyproject.toml
- A2A: HTTP POST between K8s services (commerce.agents.svc:8001 ↔ supply-chain.agents.svc:8002)
- UCP: REST endpoints on Commerce (/.well-known/ucp, /ucp/catalog, /ucp/checkout-sessions)
- MCP: ADK MCPToolset for Google Maps (requires GOOGLE_API_KEY)
- Tools are in `tools/` subdirectory per system
- Config via environment variables (REDIS_HOST, GCP_PROJECT_ID, GCS_BUCKET, GOOGLE_API_KEY, etc.)
