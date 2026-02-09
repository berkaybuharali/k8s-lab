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
- Cloud-agnostic (works across GCP, AWS, Azure, etc.)
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

**Task:** Testing & Validation (Phase 8)

**Context:**
- All implementation phases (6a-6e) are complete and verified
- Documentation (Phase 7) is complete
- Need to validate full lifecycle end-to-end

**Requirements:**
1. **Full Lifecycle Integration Test:**
   - Script or manual test: deploy-infra → deploy-tools → deploy-applications → seed-redis → backup → destroy → restore
   - Verify data integrity after restore (Redis keys)
   - Test both Makefile and Go CLI paths
2. **Edge Case Testing:**
   - Backup/restore with no data
   - Restore to fresh cluster
   - Volume snapshot verification

**Next Steps:**
- Create integration test script or document manual test procedure
- Run full lifecycle with both interfaces
- Document any issues or limitations

---

# Recent Accomplishments

1. **Completed Documentation (Phase 7)** - Created cli/README.md with architecture docs, updated root README with dual-interface Quick Start (Makefile vs Go CLI), installation instructions
2. **Fixed Velero Volume Snapshots** - Removed `IncludeClusterResources: false` from Backup CR to enable PV snapshot creation
3. **Implemented `backup` and `restore` commands** - Full DR capability via Go CLI (Phase 6e complete)
4. **Implemented `seed-redis` command** - Populates Redis with 100+ keys via optimized bulk `Exec` (Phase 6d complete)
5. **Enhanced K8s Client** - Added robust `Exec`, `DeleteNamespace`, and `ApplyManifest` with SSA support
6. **Implemented `deploy-applications` command** - Deploys NGINX and Redis using Go CLI (Phase 6c complete)
7. **Implemented `deploy-tools gcp` command** - Installs CSI driver, StorageClass, and Velero via Go CLI (Phase 6b complete)

---

# Next Steps

**Phase 8 - Testing & Validation:**
1. Full lifecycle integration test
   - Script: deploy-infra → deploy-tools → deploy-applications → seed-redis → backup → destroy → restore
   - Verify data integrity after restore
   - Test both Makefile and Go CLI paths

**Phase 9 - Multi-Cloud Preparation:**
2. Additional cloud provider scaffolding (e.g., AWS, Azure)
   - Define cloud-specific config structure
   - Create `cli/pkg/cloud/<provider>/` package
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