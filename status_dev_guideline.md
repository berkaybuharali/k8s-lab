# Project Status

**Current Phase:** Migration to Go CLI (Phase 6c - Application Deployment)

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

**Task:** Implement `deploy-applications` command (Phase 6c)

**Context:**
- `deploy-tools` command verified and working (`./bin/k8s-lab deploy-tools --cloud gcp --verbose`)
- Ready to implement application deployment (NGINX, Redis)

**Requirements:**
1. **Cloud-Agnostic Design:**
   - Applications layer must work across clouds (Layer 4 in architecture)
   - K8s manifests in `apps/` directory (not `apps/gcp/`)
   - Only use cloud provider for infrastructure queries if needed

2. **Go SDK Preference:**
   - Use K8s client-go for deployments (avoid kubectl shell commands)
   - Apply manifests programmatically via K8s API
   - Follow existing pattern from `deploy-tools` (see `cli/pkg/k8s/client.go`)

3. **Testing:**
   - Write sensible unit tests where appropriate
   - Test manifest loading and parsing
   - Mock K8s client for deployment tests

4. **Applications to Deploy:**
   - NGINX: 2 replicas, stateless (apps/nginx.yaml)
   - Redis: 1 replica, stateful with GCE PD persistence (apps/redis.yaml)
   - Create namespace if not exists
   - Wait for deployments to be ready

5. **Implementation Notes:**
   - Command: `./k8s-lab deploy-applications --cloud gcp`
   - Reuse K8s client from `cli/pkg/k8s/client.go`
   - Create IAP tunnel before K8s operations (follow deploy-tools pattern)
   - Use logger for step-by-step feedback

**Documentation:**
- README updates deferred to Phase 7 (comprehensive documentation phase)
- Focus on implementation and testing for now

**Files to Create/Modify:**
- `cli/cmd/deploy_applications.go` (new command)
- `cli/pkg/k8s/client.go` (add ApplyManifest methods if needed)
- Tests: `cli/cmd/deploy_applications_test.go`, `cli/pkg/k8s/client_test.go`

---

# Recent Accomplishments

1. **Implemented `deploy-tools gcp` command** - Installs CSI driver, StorageClass, and Velero via Go CLI (Phase 6b complete)
2. **Verified deploy-tools in production** - `./bin/k8s-lab deploy-tools --cloud gcp --verbose` working end-to-end
3. **Fixed Velero deployment** - Correct Terraform output names, kubectl rollout status for wait
4. **Established K8s API tunnel** - Deploy-tools creates IAP tunnel before K8s operations
5. **Created dual-agent coordination files** - status_dev_guideline.md, GEMINI.md for Claude/Gemini workflow

---

# Next Steps

**Immediate (Phase 6c):**
1. Implement `deploy-applications gcp` command
   - NGINX deployment (2 replicas)
   - Redis deployment (1 replica + PersistentVolumeClaim)
   - Namespace creation and management
   - Deployment readiness checks

**Phase 6d - Data Operations:**
2. Implement `seed-redis gcp` command
   - Connect to Redis pod
   - Insert test data (key-value pairs)
   - Verify data persistence

**Phase 6e - Backup/Restore:**
3. Implement `backup gcp` command
   - Create Velero backup of application namespace
   - Wait for backup completion
   - Verify backup in GCS

4. Implement `restore gcp` command
   - Install tools if needed
   - Restore application namespace from backup
   - Verify application data integrity

**Phase 7 - Documentation:**
5. Comprehensive documentation update
   - Create `cli/README.md` (architecture, packages, testing)
   - Update root `README.md` (dual-CLI usage, Quick Start)
   - Update all READMEs with Go CLI examples
   - Ensure terraform.tfvars.example has TODOs

**Phase 8 - Testing & Validation:**
6. Full lifecycle integration test
   - Script: deploy-infra → deploy-tools → deploy-applications → seed-redis → backup → destroy → restore
   - Verify data integrity after restore
   - Test both Makefile and Go CLI paths

**Phase 9 - Multi-Cloud Preparation:**
7. STACKIT provider scaffolding
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
