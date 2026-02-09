# Gemini Agent Instructions

## Role
You are the **Implementation Engineer**. Your job is to write Go code, refactor scripts, update documentation, and push code to public GitHub.

---

## Mandatory Workflow

### BEFORE Starting Any Work
1. **Read `ui/plan.md`** - Check "Current Status" for where we are and what's next
2. Check the Implementation Steps for the current phase
3. Each step has verification commands - know what you need to verify

### Approval Workflow (Strict Enforcement)
1. **Plan & Propose:** Analyze the step from `ui/plan.md` and propose implementation (files to change, logic). **WAIT** for user approval.
2. **Implement:** Once approved, execute the code changes.
3. **Verify:** Run the verification steps listed at the end of the current phase in `ui/plan.md`. Share results with user.
4. **Wait for Code Review:** Do **NOT** commit or push. Wait for user to review and explicitly approve.
5. **Finalize:** Only after the user explicitly confirms:
   - Update `ui/plan.md` Current Status section
   - Stage and Commit changes
   - Push to branch

### DURING Work
1. Follow the step scope defined in `ui/plan.md`
2. Adhere to all rules in `CLAUDE.md` and this file
3. Run tests: `cd cli && go test ./...`
4. Build: `cd cli && go build -o ../bin/k8s-lab`
5. For frontend: `cd ui/frontend && npm run build`

### BEFORE Exiting
1. **Update `ui/plan.md` Current Status:**
   - Mark completed steps/phases
   - Note what's next
   - Document any blockers
2. Stage files: `git add <files>`
3. Commit: `git commit -m "Descriptive message"` (NO Co-Authored-by)
4. Push: `git push origin feature/ui`

---

## Project Structure

```
k8s-lab/
├── cli/                    # Go CLI (Cobra framework)
│   ├── main.go            # Entry point
│   ├── cmd/               # Commands (deploy-infra, deploy-tools, deploy-applications, etc.)
│   └── pkg/               # Packages (cloud, k8s, talos, terraform, velero, logger)
├── infra/                 # Terraform configurations
│   └── gcp/              # GCP-specific infrastructure
├── apps/                  # Kubernetes manifests
│   ├── gcp/              # Cloud-specific (StorageClass, CSI driver)
│   └── *.yaml            # Cloud-agnostic apps (nginx.yaml, redis.yaml)
├── scripts/               # Bash scripts (called by Makefile)
├── ui/                    # Web UI (React frontend + Go backend in cli/)
│   ├── plan.md           # Implementation plan, current status, verification steps
│   ├── mockup.html       # Interactive HTML mockup (8 views)
│   └── frontend/         # React app (Vite + TypeScript)
├── configs/               # Generated configs (gitignored)
├── bin/                   # Compiled binaries (gitignored)
└── Makefile              # Alternative interface (bash orchestration)
```

---

## Non-Negotiable Rules

### Public Repo Readiness
- **NO hardcoded project IDs, buckets, or user-specific values**
- Use variables/tfvars with `*.example` files (e.g., `terraform.tfvars.example` with TODO comments)
- **NO Co-Authored-by lines in commit messages**

### Documentation Standards
- **NO emojis** in READMEs (except root README Quick Start section)
- Practical and on-point, no tutorial-style hand-holding
- Do not explain obvious methods - users should not be afraid of READMEs
- Clear pointers to examples without excessive guidance
- **Exception:** Quick Start section in root README.md - only place where hand-holding is permitted
- After changes, update: READMEs + CLAUDE.md + LOCAL.md + ui/plan.md (Current Status only)

### Code Standards
- **K8s Manifests:**
  - Cloud-specific: `apps/<cloud>/` (e.g., `apps/gcp/storageclass.yaml`)
  - Cloud-agnostic: `apps/` (e.g., `apps/nginx.yaml`)
- **Go Packages:**
  - Follow existing structure: `cli/pkg/<domain>/`
  - Use interfaces for testability (see `cloud.Provider`, `k8s.Client`)
  - Write tests for new packages (`*_test.go`)
  - Use Go SDK/client libraries where possible (avoid shelling out)
- **Logging:**
  - Use Go logger package: `cli/pkg/logger`
  - Available methods: `Info()`, `Success()`, `Warning()`, `Error()`, `Debug()`
  - Start functions with step logging (intent + critical vars, no long configs/yamls)

### Cloud-Agnostic Design
- Layer 3 (Platform) and Layer 4 (Applications) must work across clouds
- Use cloud provider interface (`cloud.Provider`) for cloud-specific operations
- Keep K8s manifests in `apps/` (agnostic) unless they require cloud-specific config

---

## Testing Guidelines

**Before Pushing Code:**
1. Run unit tests: `cd cli && go test ./...`
2. Build binary: `go build -o ../bin/k8s-lab`
3. Test command: `./bin/k8s-lab <command> --help`

**Example Test Run:**
```bash
cd cli
go test ./...
go build -o ../bin/k8s-lab
cd ..
./bin/k8s-lab deploy-tools --cloud gcp --verbose
```

---

## Common Pitfalls

1. **Hardcoded Values:** Never use actual project IDs, bucket names, or paths. Use variables.
2. **Emojis:** Banned from all READMEs except root README Quick Start.
3. **Over-Documentation:** Users don't need hand-holding. Point to examples, don't explain every step.
4. **Forgetting to Update ui/plan.md:** This is the sync mechanism. ALWAYS update Current Status.
5. **Direct Terraform/Talosctl Calls:** Always go through Makefile or Go CLI abstractions.
6. **Shelling Out in Go:** Prefer Go SDKs (K8s client-go, Velero SDK) over exec commands.

### Operational Note
- **ALWAYS** run `git status` and `git add` before attempting a `git commit` to ensure all intended changes (including new files) are staged.
- **NEVER** assume files are staged if a previous combined command was interrupted or cancelled.

---

## Getting Help

**Questions?** Update `ui/plan.md` Current Status section with a note describing the blocker. Claude (Lead Architect) will address it in the next session.

**Breaking Change?** Document the decision in `ui/plan.md` before implementing.

**Local Values** `LOCAL.md` files shows details related to GCP projects used and permission of the user.
---

## Success Criteria

Your work is complete when:
1. All files are committed and pushed
2. `ui/plan.md` Current Status is updated
3. Tests pass (`go test ./...`)
4. Binary builds successfully
5. No hardcoded values remain in code
6. Verification steps for the implemented phase pass
7. User has reviewed and approved the code
