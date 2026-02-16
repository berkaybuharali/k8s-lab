# Magic Cake Agents

Python ADK-based AI agents for the Magic Cake shop.

## Directory Structure

```
agents/
├── shared/              # Shared config and Redis client
├── commerce/            # Commerce Concierge (System A, port 8001)
└── supply_chain/        # Supply Chain Intelligence (System B, port 8002)
```

## Local Development Setup

### Prerequisites

- Python 3.11+
- GCP project with ADK enabled
- Values from `LOCAL.md` (project ID, bucket, region)

### Installation

1. **Create local config files from examples:**

```bash
# Shared config
cp agents/shared/config.py.example agents/shared/config.py
# Edit config.py with your values from LOCAL.md

# K8s manifests
cp apps/agents/configmap.yaml.example apps/agents/configmap.yaml
cp apps/agents/commerce.yaml.example apps/agents/commerce.yaml
cp apps/agents/supply-chain.yaml.example apps/agents/supply-chain.yaml
# Edit with your project ID and region
```

2. **Install Python packages (optional for local development):**

```bash
# Create virtual environment
python3 -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install shared package
pip install -e agents/shared/

# Install agent packages
pip install -e agents/commerce/
pip install -e agents/supply_chain/
```

**Note:** Installing packages locally is optional. The agents are designed to run in containers where packages are installed during Docker build. Local installation is only needed if you want IDE autocomplete and type checking.

### Fixing Import Errors in IDE

If you see red squiggles on `from google import adk`, you have two options:

**Option 1: Install packages in virtual environment (recommended)**
```bash
python3 -m venv venv
source venv/bin/activate
pip install -e agents/shared/
pip install -e agents/commerce/
```

**Option 2: Accept the red squiggles**
The code will work fine in containers. The imports are only "red" because your local environment doesn't have the packages installed.

## Artifact Registry Setup

**Repository creation:** Managed by Terraform during `deploy-infra` (see `infra/gcp/terraform/artifact_registry.tf`).

**Talos nodes authentication:** Automatically configured during `deploy-infra` via machine config patches (`infra/gcp/talos-patches/artifact-registry.yaml` + `csi.yaml`). The patches are merged into generated Talos machine configs for all nodes.

**Docker authentication (one-time, for local development):**
```bash
# Configure Docker to authenticate with Artifact Registry
gcloud auth configure-docker europe-west4-docker.pkg.dev
```

This only needs to be done once on your local machine. The repository stores built container images for both Commerce and Supply Chain agents.

## Running Agents

Agents are deployed to Kubernetes via the Go CLI:

```bash
# Deploy both agent systems to K8s
./bin/k8s-lab deploy-agents --cloud gcp

# Test via port-forward
kubectl port-forward -n agents svc/commerce 8001:8001
curl -X POST http://localhost:8001/run -d '{"app_name":"commerce","user_id":"test","session_id":"s1","new_message":{"role":"user","parts":[{"text":"Hello"}]}}'
```

See the main README.md for full deployment workflow.

## Architecture

### Commerce Concierge (System A)

- **Port:** 8001
- **Agents:** Translation, Cake Designer, Checkout
- **Responsibilities:** Customer interaction, cake customization, order placement
- **Protocols:** A2A (to Supply Chain), UCP (external agents)

### Supply Chain Intelligence (System B)

- **Port:** 8002
- **Agents:** Inventory, Order Service, Fulfillment
- **Responsibilities:** Inventory management, order storage, delivery routing
- **Protocols:** A2A (to Commerce), MCP (Google Maps)

## Development

See `agent_plan.md` (gitignored, local only) for detailed implementation plan.

Current phase: Phase 1 complete (scaffolding). Next: Phase 2 (Supply Chain tools).
