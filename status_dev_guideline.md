# Project Status

**Current Phase:** Migration to Go CLI (Phase 6e - Backup/Restore)

**Branch:** `feature/go-migration`

---

# Architecture Decision Record

## Why Talos Linux?
- Immutable OS designed specifically for Kubernetes
- API-driven configuration (no SSH)
- Minimal attack surface, secure by default
- Perfect for reproducible infrastructure

## Why Velero?
- Industry-standard K8s backup/restore solution
- Cloud-agnostic (works across GCP, STACKIT, etc.)
- Supports backup hooks for data consistency (Redis BGSAVE)
- Enables cluster migration and disaster recovery

## Why Go CLI + Bash Scripts?
- **Dual-Agent workflow:** Claude (architect) + Gemini (implementer)
- **User choice:** Developers pick their preferred interface
- **Industry standard:** Both Makefile (bash) and standalone CLI binary
- **Risk mitigation:** Go CLI development doesn't break existing workflows
- **Modern appeal:** Cobra CLI attracts new contributors

## Why Terraform?
- Declarative infrastructure as code
- State management (GCS backend)
- Cloud provider abstraction layer
- Enables reproducible deployments

---

# Active Task

**Task:** Implement `backup` and `restore` commands (Phase 6e)

**Context:**
- `seed-redis` verified and working (`./bin/k8s-lab seed-redis --cloud gcp`)
- Need to implement the core disaster recovery features using Velero Go SDK (or API)

**Requirements:**
1. **Backup Command:**
   - Command: `./k8s-lab backup --cloud gcp`
   - Create a Velero backup of the `application` namespace
   - Use dynamic backup naming (e.g., `lab-backup-YYYYMMDD-HHMMSS`)
   - Wait for backup completion (status: `Completed`)
2. **Restore Command:**
   - Command: `./k8s-lab restore --cloud gcp --backup <name>`
   - Restore the `application` namespace from the specified backup
   - Wait for restore completion
3. **Implementation Notes:**
   - Use Velero Go SDK or Dynamic Client to interact with Velero CRDs
   - Reuse existing tunnel and k8s client logic
   - While implementing go functionality, if you have a doubt, look the bash way. Bash way is implemented before and working as expected.

**Files to Create/Modify:**
- `cli/cmd/backup.go` (new)
- `cli/cmd/restore.go` (new)
- `cli/pkg/velero/client.go` (implement backup/restore logic)

---

# Recent Accomplishments

1. **Implemented `seed-redis` command** - Populates Redis with 100+ keys via optimized bulk `Exec` (Phase 6d complete)
2. **Enhanced K8s Client** - Added robust `Exec` with pod readiness checks and `ApplyManifest` with SSA support
3. **Implemented `deploy-applications` command** - Deploys NGINX and Redis using Go CLI (Phase 6c complete)
4. **Implemented `deploy-tools gcp` command** - Installs CSI driver, StorageClass, and Velero via Go CLI (Phase 6b complete)
5. **Verified deploy-tools in production** - `./bin/k8s-lab deploy-tools --cloud gcp --verbose` working end-to-end
6. **Created dual-agent coordination files** - status_dev_guideline.md, GEMINI.md for Claude/Gemini workflow

---

# Next Steps

**Phase 7 - Documentation:**
1. Comprehensive documentation update
   - Create `cli/README.md` (architecture, packages, testing)
   - Update root `README.md` (dual-CLI usage, Quick Start)
   - Update all READMEs with Go CLI examples
   - Ensure terraform.tfvars.example has TODOs

**Phase 8 - Testing & Validation:**
2. Full lifecycle integration test
   - Script: deploy-infra → deploy-tools → deploy-applications → seed-redis → backup → destroy → restore
   - Verify data integrity after restore
   - Test both Makefile and Go CLI paths

**Phase 9 - Multi-Cloud Preparation:**
3. STACKIT provider scaffolding
   - Define STACKIT config structure
   - Create `cli/pkg/cloud/stackit/` package
   - Update provider interface for multi-cloud

---

# Development Workflow

**CRITICAL:** Both agents (Claude + Gemini) MUST:
1. **Read this file FIRST** before starting any work
2. **Update this file BEFORE exiting** (Active Task → Recent Accomplishments, update Next Steps)
3. **Follow CLAUDE.md rules** (no hardcoded IDs, documentation standards)
4. **Commit frequently** with descriptive messages (no Co-Authored-by lines)

**Agent Handoff Protocol:**
- Claude: Architecture, complex logic, planning, ADR updates
- Gemini: Implementation, scaffolding, refactoring, testing, git operations
- This file = sync mechanism between agents
