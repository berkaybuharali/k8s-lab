# Project Status

**Current Phase:** Migration to Go CLI (Phase 6d - Data Operations)

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

**Task:** Implement `seed-redis` command (Phase 6d)

**Context:**
- `deploy-applications` verified and working (`./bin/k8s-lab deploy-applications --cloud gcp`)
- Need to populate Redis with test data to verify statefulness and backup capabilities

**Requirements:**
1. **Connect to Redis:**
   - Use `kubectl exec` equivalent via client-go (SPDY executor)
   - Target the `redis` pod in `application` namespace
2. **Seed Data:**
   - Set 100+ keys (e.g., `user:1`, `user:2`...)
   - Verify keys are set (count them)
3. **Implementation Notes:**
   - Command: `./k8s-lab seed-redis --cloud gcp`
   - Use `cli/pkg/k8s/client.go` - might need to add `Exec` method
   - Reuse existing tunnel creation logic
   - Log progress clearly
   - While implementing go functionality, if you have a doubt, look the bash way. Bash way is implemented before and working as expected.

**Files to Create/Modify:**
- `cli/cmd/seed_redis.go` (new command)
- `cli/pkg/k8s/client.go` (add Exec method)

---

# Recent Accomplishments

1. **Implemented `deploy-applications` command** - Deploys NGINX and Redis using Go CLI (Phase 6c complete)
2. **Implemented `deploy-tools gcp` command** - Installs CSI driver, StorageClass, and Velero via Go CLI (Phase 6b complete)
3. **Verified deploy-tools in production** - `./bin/k8s-lab deploy-tools --cloud gcp --verbose` working end-to-end
4. **Fixed Velero deployment** - Correct Terraform output names, kubectl rollout status for wait
5. **Established K8s API tunnel** - Deploy-tools creates IAP tunnel before K8s operations
6. **Created dual-agent coordination files** - status_dev_guideline.md, GEMINI.md for Claude/Gemini workflow

---

# Next Steps

**Phase 6e - Backup/Restore:**
1. Implement `backup gcp` command
   - Create Velero backup of application namespace
   - Wait for backup completion
   - Verify backup in GCS

2. Implement `restore gcp` command
   - Install tools if needed
   - Restore application namespace from backup
   - Verify application data integrity

**Phase 7 - Documentation:**
3. Comprehensive documentation update
   - Create `cli/README.md` (architecture, packages, testing)
   - Update root `README.md` (dual-CLI usage, Quick Start)
   - Update all READMEs with Go CLI examples
   - Ensure terraform.tfvars.example has TODOs

**Phase 8 - Testing & Validation:**
4. Full lifecycle integration test
   - Script: deploy-infra → deploy-tools → deploy-applications → seed-redis → backup → destroy → restore
   - Verify data integrity after restore
   - Test both Makefile and Go CLI paths

**Phase 9 - Multi-Cloud Preparation:**
5. STACKIT provider scaffolding
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