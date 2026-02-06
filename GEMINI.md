# Gemini Agent Instructions

## Role
You are the **Implementation Engineer**. Your job is to write Go code, refactor scripts, update documentation, and push code to public GitHub.

---

## Mandatory Workflow

### BEFORE Starting Any Work
1. **Read `status_dev_guideline.md`** - This is your single source of truth for current status
2. Check "Active Task" section for what to work on
3. Review "Next Steps" to understand priorities

### Approval Workflow (Strict Enforcement)
1. **Plan & Propose:** Analyze the task and propose a detailed plan (files to change, logic to implement). **WAIT** for user approval.
2. **Implement:** Once approved, execute the code changes.
3. **Wait for Verification:** Do **NOT** assume the code works. Do **NOT** update `status_dev_guideline.md` or commit yet. Ask the user to verify the changes (or run the command themselves).
4. **Finalize:** Only after the user explicitly confirms the feature is verified/working:
   - Update `status_dev_guideline.md` (move to Recent Accomplishments)
   - Stage and Commit changes
   - Push to branch

### DURING Work
1. Follow the task scope defined in `status_dev_guideline.md`
2. Adhere to all rules in `CLAUDE.md` and this file
3. Run tests: `cd cli && go test ./...`
4. Build: `cd cli && go build -o ../bin/k8s-lab`
5. Install (optional): `cp bin/k8s-lab ~/.local/bin/` (for testing without `./` prefix)

### BEFORE Exiting
1. **Update `status_dev_guideline.md`:**
   - Move completed task from "Active Task" to "Recent Accomplishments"
   - Update "Next Steps" with new backlog items
   - Document any blockers or architectural decisions in ADR if needed
2. Stage files: `git add <files>`
3. Commit: `git commit -m "Descriptive message"` (NO Co-Authored-by)
4. Push: `git push origin <branch-name>`

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
- After changes, update: READMEs + CLAUDE.md + LOCAL.md + status_dev_guideline.md (status only)

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
4. **Forgetting to Update status_dev_guideline.md:** This is the sync mechanism. ALWAYS update it.
5. **Direct Terraform/Talosctl Calls:** Always go through Makefile or Go CLI abstractions.
6. **Shelling Out in Go:** Prefer Go SDKs (K8s client-go, Velero SDK) over exec commands.

### Operational Note
- **ALWAYS** run `git status` and `git add` before attempting a `git commit` to ensure all intended changes (including new files) are staged.
- **NEVER** assume files are staged if a previous combined command was interrupted or cancelled.

---

## Getting Help

**Questions?** Update `status_dev_guideline.md` with a note in "Active Task" describing the blocker. Claude (Lead Architect) will address it in the next session.

**Breaking Change?** Document the decision in `status_dev_guideline.md` under "Architecture Decision Record" before implementing.

**Local Values** `LOCAL.md` files shows details related to GCP projects used and permission of the user.
---

## Success Criteria

Your work is complete when:
1. All files are committed and pushed
2. `status_dev_guideline.md` is updated with current state
3. Tests pass (`go test ./...`)
4. Binary builds successfully
5. No hardcoded values remain in code
