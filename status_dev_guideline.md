# Project Status

**Current Phase:** Migration to Go CLI (Phase 7 - Documentation)

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

**Task:** Documentation Update (Phase 7)

**Context:**
- All implementation phases (6a-6e) are complete and verified.
- The Go CLI now supports the full lifecycle: infra -> tools -> apps -> data -> backup -> restore.
- Need to document the new Go CLI usage alongside the existing Makefile/Bash commands.

**Requirements:**
1. **New `cli/README.md`:**
   - Architecture overview (Cobra, packages)
   - Testing guide (`go test`, mocks)
   - Directory structure explanation
2. **Update root `README.md`:**
   - Dual-CLI usage guide (Bash vs Go)
   - Quick Start section using Go CLI
   - Prerequisites
3. **Update `apps/README.md` & `infra/README.md`:**
   - Add Go CLI examples for relevant commands
4. **Housekeeping:**
   - Ensure `terraform.tfvars.example` has clear TODOs
   - Verify `CLAUDE.md` is up to date

**Files to Create/Modify:**
- `cli/README.md` (new)
- `README.md` (update)
- `apps/README.md` (update)
- `infra/README.md` (update)

---

# Recent Accomplishments

1. **Implemented `backup` and `restore` commands** - Full DR capability via Go CLI (Phase 6e complete)
2. **Fixed Velero Volume Snapshots** - Correctly configured Backup CR to capture GCP Persistent Disks
3. **Implemented `seed-redis` command** - Populates Redis with 100+ keys via optimized bulk `Exec` (Phase 6d complete)
4. **Enhanced K8s Client** - Added robust `Exec`, `DeleteNamespace`, and `ApplyManifest` with SSA support
5. **Implemented `deploy-applications` command** - Deploys NGINX and Redis using Go CLI (Phase 6c complete)
6. **Implemented `deploy-tools gcp` command** - Installs CSI driver, StorageClass, and Velero via Go CLI (Phase 6b complete)

---

# Next Steps

**Phase 8 - Testing & Validation:**
1. Full lifecycle integration test
   - Script: deploy-infra → deploy-tools → deploy-applications → seed-redis → backup → destroy → restore
   - Verify data integrity after restore
   - Test both Makefile and Go CLI paths

**Phase 9 - Multi-Cloud Preparation:**
2. STACKIT provider scaffolding
   - Define STACKIT config structure
   - Create `cli/pkg/cloud/stackit/` package
   - Update provider interface for multi-cloud

---

# Post-Merge Tasks

**Bash Script Alignment (After feature/go-migration Merge):**
- **Add `--clean` flag to bash restore script** (`scripts/backup/restore.sh`)
  - Implementation: `if [[ "$2" == "--clean" ]]; then kubectl delete namespace application --timeout=5m; fi`
  - Purpose: Align bash restore behavior with Go CLI (both support optional clean restore)
  - Currently: Bash never deletes namespace, Go supports `--clean` flag (default: true)
  - After: Both CLIs will support `--clean` flag for disaster recovery testing

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