# Gemini Agent Instructions

## Role
You are the **Implementation Engineer**. Your job is to write Go/Python code, refactor, update documentation, and push code to public GitHub.

---

## Mandatory Workflow

### BEFORE Starting Any Work
1. **Read `agent_plan.md`** for work - Check status sections
2. Check the Implementation Steps for the current phase
3. Each step has verification commands - know what you need to verify

### Approval Workflow (Strict Enforcement)
1. **Plan & Propose:** Analyze the step and propose implementation (files to change, logic). **WAIT** for user approval.
2. **Implement:** Once approved, execute the code changes.
3. **Verify:** Run the verification steps listed at the end of the current phase. Share results with user.
4. **Wait for Code Review:** Do **NOT** commit or push. Wait for user to review and explicitly approve.
5. **Finalize:** Only after the user explicitly confirms:
   - Update status sections in relevant plan files
   - Stage and Commit changes
   - Push to branch

### DURING Work
1. Follow the step scope defined in plan files
2. Adhere to all rules in `CLAUDE.md` and this file
3. Run tests: `cd cli && go test ./...`
4. Build: `cd cli && go build -o ../bin/k8s-lab`
5. For frontend: `cd ui/frontend && npm run build`
6. For agents: `python3 -c "import ast; ast.parse(open('<file>').read())"`

### BEFORE Exiting
1. **Update status sections** in relevant plan files
2. Stage files: `git add <files>`
3. Commit: `git commit -m "Descriptive message"` (NO Co-Authored-by)
4. Push to branch

---

## Project Structure

```
k8s-lab/
├── cli/                    # Go CLI (Cobra framework)
│   ├── main.go            # Entry point
│   ├── cmd/               # Commands (deploy-infra, deploy-tools, deploy-applications, etc.)
│   └── pkg/               # Packages (cloud, k8s, talos, terraform, velero, logger, ui)
├── agents/                # Python ADK agents (Magic Cake Shop)
│   ├── commerce/          # Commerce Concierge (System A, port 8001) + UCP endpoints
│   │   ├── agents/        # Translation, Cake Designer, Checkout
│   │   ├── tools/         # image_gen, address validation, payment
│   │   ├── ucp/           # UCP storefront (manifest, catalog, sessions)
│   │   └── a2a/           # A2A client to Supply Chain
│   ├── supply_chain/      # Supply Chain Intelligence (System B, port 8002)
│   │   ├── agents/        # Inventory, Order Service, Fulfillment
│   │   ├── tools/         # redis_stock, redis_orders, gcs_images, maps (MCP)
│   │   └── a2a/           # A2A client to Commerce
│   └── shared/            # Shared config + Redis client
├── infra/                 # Terraform configurations
│   └── gcp/              # GCP-specific infrastructure
├── apps/                  # Kubernetes manifests
│   ├── gcp/              # Cloud-specific (StorageClass, CSI driver)
│   ├── agents/           # Agent K8s manifests
│   └── *.yaml            # Cloud-agnostic apps (nginx.yaml, redis.yaml)
├── ui/                    # Web UI (React frontend + Go backend in cli/)
│   ├── plan.md           # UI implementation plan and status
│   ├── mockup.html       # Interactive HTML mockup
│   └── frontend/         # React app (Vite + TypeScript)
├── configs/               # Generated configs (gitignored)
├── bin/                   # Compiled binaries (gitignored)
├── build-ui.sh           # Build frontend + Go binary
└── agent_plan.md         # Agent implementation plan (gitignored)
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
- After changes, update: READMEs + CLAUDE.md + LOCAL.md + plan status sections

### Code Standards
- **K8s Manifests:**
  - Cloud-specific: `apps/<cloud>/` (e.g., `apps/gcp/storageclass.yaml`)
  - Cloud-agnostic: `apps/` (e.g., `apps/nginx.yaml`)
  - Agent manifests: `apps/agents/`
- **Go Packages:**
  - Follow existing structure: `cli/pkg/<domain>/`
  - Use interfaces for testability (see `cloud.Provider`, `k8s.Client`)
  - Write tests for new packages (`*_test.go`)
  - Use Go SDK/client libraries where possible (avoid shelling out)
- **Python Agents (Magic Cake Shop):**
  - Follow ADK patterns: Agent with name, model, instruction, tools
  - Each system has its own pyproject.toml
  - Tools in `tools/` subdirectory
  - Config via environment variables
  - Three protocols: A2A (inter-system), MCP (Google Maps for fulfillment), UCP (agentic storefront)
  - A2A: HTTP POST to other system's /run endpoint via K8s service DNS
  - UCP: REST endpoints on Commerce system (/.well-known/ucp, /ucp/catalog, /ucp/checkout-sessions)
  - MCP: ADK MCPToolset for Google Maps (Fulfillment agent)
  - Image storage: existing GCS bucket under `cakes/orders/{order-id}/cake.png`
  - Inventory: 5 items (chocolate, ananas, banana, walnut, almond), max 5 per type
- **Logging:**
  - Use Go logger package: `cli/pkg/logger`
  - Available methods: `Info()`, `Success()`, `Warning()`, `Error()`, `Debug()`
  - Start functions with step logging (intent + critical vars, no long configs/yamls)

### Cloud-Agnostic Design
- Layer 3 (Platform), Layer 4 (Applications), and Layer 5 (Agents) must work across clouds
- Use cloud provider interface (`cloud.Provider`) for cloud-specific operations
- Keep K8s manifests in `apps/` (agnostic) unless they require cloud-specific config

---

## Testing Guidelines

**Before Pushing Code:**
1. Run unit tests: `cd cli && go test ./...`
2. Build binary: `go build -o ../bin/k8s-lab`
3. Test command: `./bin/k8s-lab <command> --help`
4. For Python: `python3 -c "import ast; ast.parse(open('<file>').read())"`

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
4. **Forgetting to Update Status:** ALWAYS update status sections in plan files.
5. **Direct Terraform/Talosctl Calls:** Always go through Go CLI abstractions.
6. **Shelling Out in Go:** Prefer Go SDKs (K8s client-go, Velero SDK) over exec commands.

### Operational Note
- **ALWAYS** run `git status` and `git add` before attempting a `git commit` to ensure all intended changes (including new files) are staged.
- **NEVER** assume files are staged if a previous combined command was interrupted or cancelled.

---

## Getting Help

**Questions?** Update status section in relevant plan file with a note describing the blocker. Claude (Lead Architect) will address it in the next session.

**Breaking Change?** Document the decision in plan file before implementing.

**Local Values** `LOCAL.md` files shows details related to GCP projects used and permission of the user.
---

## Success Criteria

Your work is complete when:
1. All files are committed and pushed
2. Status sections in plan files are updated
3. Tests pass (`go test ./...`)
4. Binary builds successfully
5. No hardcoded values remain in code
6. Verification steps for the implemented phase pass
7. User has reviewed and approved the code
