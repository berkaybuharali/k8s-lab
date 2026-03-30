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

# Interactive chat with Commerce Concierge (handles tunnel + port-forward automatically)
./bin/k8s-lab agent-chat --system commerce --cloud gcp

# Resume a previous conversation
./bin/k8s-lab agent-chat --system commerce --cloud gcp --session <contextId>

# Single-turn query to Supply Chain
./bin/k8s-lab agent-chat --system supply-chain --cloud gcp "Show me current stock"
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
A2A is Google ADK's protocol for agent-to-agent communication. It enables AI agents in different systems to discover each other's capabilities and communicate via natural language messages over JSON-RPC 2.0 over HTTP.

**Why we use it:**
Commerce and Supply Chain are independent systems that need to talk. A2A allows them to communicate naturally without tight coupling - Commerce doesn't need to know Supply Chain's internal API structure, just its agent capabilities.

**How it works:**
1. Each system exposes `/.well-known/agent-card.json` describing its capabilities
2. Communication happens via `POST /` with JSON-RPC `message/send` method
3. Messages are natural language, not rigid API contracts
4. A2A servers use `contextId` for multi-turn sessions

**Implementation in Magic Cake:**

**Commerce → Supply Chain (Customer Flow):**
- Cake Designer checks ingredient availability before offering options
- Checkout deducts inventory after payment
- Checkout creates orders with images in Supply Chain database

**Code example (direct HTTP A2A):**
```python
# agents/commerce/a2a/supply_chain_client.py
import httpx, uuid

def _call_supply_chain(message: str) -> str:
    payload = {
        "jsonrpc": "2.0",
        "method": "message/send",
        "id": "1",
        "params": {
            "message": {
                "role": "user",
                "parts": [{"kind": "text", "text": message}],
                "messageId": str(uuid.uuid4()),
            }
        },
    }
    response = httpx.post("http://supply-chain.agents.svc.cluster.local:8002/",
                          json=payload, timeout=15.0)
    response.raise_for_status()
    return _extract_text(response.json().get("result", {}))
```

**Why direct HTTP instead of RemoteA2aAgent:**

ADK provides `RemoteA2aAgent` as a convenience for wiring remote agents as top-level sub-agents in a parent agent. However, calling `remote_agent.run()` inside a **tool function** starts a nested ADK event loop that corrupts the parent agent's active session — causing the agent to loop back to the beginning of the conversation or return empty responses.

The A2A protocol is simply JSON-RPC 2.0 over HTTP. `RemoteA2aAgent` is just a wrapper around it. Using `httpx.post` directly to `POST /` with a `message/send` payload achieves the exact same result without involving ADK's session management at all.

Rule of thumb:
- `RemoteA2aAgent` → use for top-level sub-agent wiring in `sub_agents=[...]`
- Direct HTTP → use for tool-level cross-system calls

**Benefits:**
- **Loose coupling:** Systems communicate via natural language, not rigid APIs
- **Discovery:** Agent cards enable runtime capability discovery
- **Reliable:** No nested ADK event loops, no session corruption

**Testing:**
```bash
# Chat with Commerce (internally calls Supply Chain via A2A)
./bin/k8s-lab agent-chat --system commerce --cloud gcp

# Chat with Supply Chain directly
./bin/k8s-lab agent-chat --system supply-chain --cloud gcp
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
    env={"GOOGLE_API_KEY": os.getenv("GOOGLE_API_KEY")}
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

## ADK Lessons Learned

Practical pitfalls encountered building Magic Cake on ADK v1.25.

### sub_agents routing is unreliable for multi-turn conversational flows

ADK's `sub_agents` pattern lets a root agent delegate messages to child agents. The routing
decision is made by the root agent's LLM on **every single turn** — there is no built-in
mechanism to stay in a sub-agent across turns (ADK issue [#3878](https://github.com/google/adk-python/issues/3878)).

In a multi-turn conversation this caused repeated mid-conversation restarts:

- User: "lets go with ananas" → root LLM saw "ananas" (Dutch/German for pineapple), thought
  it was language input, routed back to the Translation sub-agent
- User: "walnuts" → Cake Designer forgot the previously chosen flavor and restarted from scratch

The root agent re-evaluates routing independently each turn. Explicit CRITICAL rules in the
instruction ("NEVER go back to Translation") did not reliably prevent misdelegation.

**The fix:** For a multi-turn conversational flow where context must survive many exchanges,
use a **single root agent** with all tools. The conversation history in the session is the
state — the agent reads it and follows a phased instruction without any routing machinery
between sub-agents.

```
Sub-agents remain the right choice for:
- Single-shot delegations (one request, one response, done)
- Truly independent parallel tasks
- A2A calls to external systems (Supply Chain, etc.)

Sub-agents are the wrong choice for:
- Multi-turn conversation phases (language → design → checkout)
- Any flow where you need "stay here until done"
```

`SequentialAgent` is not a solution for multi-turn flows — it runs all steps in a single
HTTP request, not across multiple user messages.

---

### Gemini API rejects `list[Dict]` / `list[dict]` tool parameters

ADK auto-generates JSON schema from Python type hints when registering tools. Generic
`dict` or `Dict` produces `"additionalProperties": {}` in the schema, which Gemini's
Generative Language API rejects:

```
400 INVALID_ARGUMENT: Unknown name "additional_properties" at
'tools[0].function_declarations[4].parameters.properties[0].value.items'
```

The failure comes back as `status.state: "failed"` in the A2A JSON response (not as
an HTTP error code), so callers that only check HTTP status receive an empty response
instead of a useful error message — silent failure.

**The fix:** Redesign list-of-dict parameters into parallel typed arrays:

```python
# Bad — list[dict] generates additionalProperties, bare list has no items
def calculate_price(cakes: list[dict]) -> dict: ...   # 400 INVALID_ARGUMENT
def calculate_price(cakes: list) -> dict: ...         # 400 missing field

# Good — use only primitively-typed arrays
def calculate_price(people_counts: list[int]) -> dict: ...

# For multi-field objects, use parallel lists instead of list-of-dicts
def create_order_remote(
    flavors: list[str],
    nuts_choices: list[str],
    people_counts: list[int],
    concepts: list[str],
    image_paths: list[str],
    ...
) -> dict: ...
```

The LLM reads the docstring to understand intent, not the type hint. Parallel arrays are
slightly less ergonomic but work reliably with Gemini's schema validation.

To surface these errors instead of returning silence, the A2A client must check
`result.status.state` and extract text from `result.status.message.parts` when
the state is `"failed"`.

---

### A2A `contextId` must be inside the Message object

In the A2A JSON-RPC `message/send` request, the `Message` type (not `MessageSendParams`)
carries the `contextId` field. Putting `contextId` at the params level is silently ignored
— the server generates a fresh session on every turn, breaking multi-turn conversations.

**Broken (contextId ignored):**
```json
{
  "method": "message/send",
  "params": {
    "contextId": "a5653ed2-...",
    "message": {"role": "user", "parts": [...], "messageId": "..."}
  }
}
```

**Correct (contextId read by server):**
```json
{
  "method": "message/send",
  "params": {
    "message": {
      "role": "user",
      "parts": [...],
      "messageId": "...",
      "contextId": "a5653ed2-..."
    }
  }
}
```

This misplacement causes the same symptom as sub_agent routing loops: the agent resets
to the beginning of the conversation on every turn (because it truly is starting fresh).
The A2A server internally maps `message.contextId → session_id` and `A2A_USER_{contextId} → user_id`.

---

## Development

See `agent_plan.md` (gitignored, local only) for detailed implementation plan.
