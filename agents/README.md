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

## Protocols

Magic Cake demonstrates three modern AI protocols for different integration patterns:

### A2A (Agent-to-Agent Protocol)

**What is it?**
A2A is Google ADK's protocol for agent-to-agent communication. It enables AI agents in different systems to discover each other's capabilities and communicate via natural language RPC.

**Why we use it:**
Commerce and Supply Chain are independent systems that need to talk. A2A allows them to communicate naturally without tight coupling - Commerce doesn't need to know Supply Chain's internal API structure, just its agent capabilities.

**How it works:**
1. Each system exposes `/.well-known/agent-card.json` describing its capabilities
2. Systems use `RemoteA2aAgent` to create client connections
3. Communication happens via natural language messages, not rigid API contracts
4. ADK handles message serialization, routing, and response parsing

**Implementation in Magic Cake:**

**Commerce → Supply Chain (Customer Flow):**
- Cake Designer checks ingredient availability before offering options
- Checkout deducts inventory after payment
- Checkout creates orders with images in Supply Chain database

**Supply Chain → Commerce (Backoffice):**
- Inventory notifies Commerce when stock reaches zero
- Commerce can update catalog in real-time

**Code example:**
```python
# agents/commerce/a2a/supply_chain_client.py
from google.adk.agents.remote_a2a_agent import RemoteA2aAgent

supply_chain = RemoteA2aAgent(
    name="supply_chain_remote",
    description="Remote Supply Chain Intelligence system",
    agent_card="http://supply-chain.agents.svc.cluster.local:8002/.well-known/agent-card.json"
)

# Natural language call - no API spec needed
response = supply_chain.run("Check stock for chocolate")
```

**Benefits:**
- **Loose coupling:** Systems communicate via natural language, not rigid APIs
- **Discovery:** Agent cards enable runtime capability discovery
- **Evolution:** Change internal implementation without breaking inter-system communication
- **Natural:** Agents reason about requests and provide context-aware responses

**Testing:**
```bash
# Chat with Commerce (internally calls Supply Chain via A2A)
./bin/k8s-lab agent-chat --system commerce --cloud gcp "I want a chocolate cake"

# Chat with Supply Chain directly
./bin/k8s-lab agent-chat --system supply-chain --cloud gcp "Show me current stock"
```

---

### UCP (Universal Commerce Protocol)

**What is it?**
UCP is an emerging standard for making businesses "agent-ready" - discoverable and transactable by external AI agents like Gemini in Search, Google Assistant, or any UCP-compatible agent.

**Why we use it:**
Instead of building a traditional e-commerce API, UCP lets external AI agents discover our catalog, configure products, and complete transactions programmatically. Magic Cake becomes accessible to any AI agent, not just our own UI.

**How it works:**
1. **Discovery:** External agents fetch `/.well-known/ucp` to learn capabilities
2. **Catalog:** Agents query `/ucp/catalog` for available products and pricing
3. **Sessions:** Agents create checkout sessions with product configurations
4. **Completion:** Agents finalize orders, triggering internal workflows

**Implementation in Magic Cake:**

**Discovery endpoint (UCP manifest):**
```bash
curl http://commerce:8001/.well-known/ucp
{
  "name": "Magic Cake Amsterdam",
  "capabilities": ["dev.ucp.shopping", "dev.ucp.checkout"],
  "services": {
    "catalog": {"endpoint": "/ucp/catalog"},
    "checkout": {"endpoint": "/ucp/checkout-sessions"}
  }
}
```

**Catalog endpoint:**
```bash
curl http://commerce:8001/ucp/catalog
{
  "flavors": [{"id": "chocolate", "available": true}, ...],
  "nuts": [{"id": "almond", "available": true}, ...],
  "pricing": {"price_per_person": 5.0, "currency": "EUR"},
  "delivery": {"area": "Amsterdam (postcodes 1000-1109)"}
}
```

**Session flow:**
```bash
# 1. Create session
curl -X POST http://commerce:8001/ucp/checkout-sessions \
  -d '{"customer_name":"AI Agent","cakes":[{"flavor":"chocolate","nuts":"walnut","people_count":12,"concept":"birthday"}]}'
# Returns: session_id, pricing, available_dates

# 2. Update session with delivery details
curl -X PUT http://commerce:8001/ucp/checkout-sessions/{session_id} \
  -d '{"delivery_date":"2026-02-17","postcode":"1012 AB","house_number":"42"}'

# 3. Complete order
curl -X POST http://commerce:8001/ucp/checkout-sessions/{session_id}/complete
# Returns: order_id, image_paths, confirmation
```

**Benefits:**
- **Agent-native commerce:** External AI agents can shop without human UI
- **Standardized:** UCP is a protocol, not a proprietary API
- **Discovery:** Agents learn capabilities at runtime, no hardcoded integrations
- **Future-proof:** Works with Gemini in Search, Google Assistant, and future UCP agents

**Comparison with traditional APIs:**

| Traditional E-commerce API | UCP Agentic Storefront |
|----------------------------|------------------------|
| Rigid endpoints (POST /orders, GET /products) | Capability-based discovery |
| Requires API documentation | Self-describing via manifest |
| Human-centric (cart, checkout pages) | Agent-native (session-based) |
| Single integration per API version | Works with any UCP-compatible agent |

**Use cases:**
- Gemini in Search: "Order a chocolate cake from Magic Cake for tomorrow"
- Google Assistant: "Get me a birthday cake from that Amsterdam bakery"
- Business agents: Automated catering orders for corporate events

---

### MCP (Model Context Protocol)

**What is it?**
MCP enables AI agents to use external tools and services. Think of it as a standardized way for agents to call APIs, databases, and services.

**Why we use it:**
Fulfillment agent needs Google Maps for route optimization. MCP provides a clean interface to Maps APIs without writing custom API clients.

**How it works:**
ADK's `MCPToolset` wraps MCP servers, making them available as agent tools. Agents use MCP tools just like regular Python functions.

**Implementation in Magic Cake:**
```python
# agents/supply_chain/agents/fulfillment.py
from google.adk.mcp import MCPToolset

maps_mcp = MCPToolset(
    server_name="google-maps",
    env={"GOOGLE_MAPS_API_KEY": os.getenv("GOOGLE_MAPS_API_KEY")}
)

fulfillment_agent = adk.Agent(
    name="fulfillment",
    instruction="Plan optimal delivery routes from Danzigerkade 4, Amsterdam",
    tools=[maps_mcp]  # MCP tools available to agent
)
```

**Benefits:**
- **Reusable:** MCP servers work across all agents and systems
- **Standardized:** No custom API client code
- **Maintained:** MCP servers updated independently of our code

---

## Development

See `agent_plan.md` (gitignored, local only) for detailed implementation plan.

Current status: Phase 4 (A2A Integration) - Commerce and Supply Chain communicate via A2A, UCP endpoints functional.
