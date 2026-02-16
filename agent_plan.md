# Agent Plan: Magic Cake Shop

## Current Status

| Phase | Status | Notes |
|-------|--------|-------|
| Phase 0: Cleanup | Complete | scripts/, Makefile removed. Go CLI is sole interface. |
| Phase 1: Foundation | Complete | Python ADK scaffolding, K8s manifests, Dockerfiles (32 files) |
| Phase 2: Supply Chain | Complete (deployed & seeded) | Agents deployed with A2A + health checks. Data seeded. Ready for Phase 3. |
| Phase 3: Commerce + UCP | Ready to Start | Commerce scaffolded. Need: image_gen, address, payment tools + UCP implementation |
| Phase 4: A2A Integration | Scaffolded | Placeholder clients created. Implement after Phase 3. |
| Phase 5: UI Elevation | Scaffolded | UI components created as placeholders. Implement after Phase 4. |
| Phase 6: Deployment | Not Started | Artifact Registry, backup scope, seed data, lifecycle |
| Phase 7: Documentation | Not Started | Docs, project history, lessons learned |

**Last Updated:** Phase 2 deployed (2024-02-16). Supply Chain agents running with A2A protocol. Seed-inventory clears old orders. Commerce tools scaffolded. Learnings added to README.md. Ready to implement Phase 3 (Commerce tools).

---

## Context

The k8s-lab project is a mature Kubernetes lab with Go CLI, React dashboard, Terraform/Talos/Velero on GCP. We're extending it with Google ADK-based AI agents organized into two systems that communicate via A2A protocol. The domain is a cake shop called **Magic Cake** that takes orders conversationally, generates cake images with Imagen, and delivers only in Amsterdam.

**Key principle:** No microservices. Only Agents. Every interaction is conversational and agentic.

**Pricing:** 5 EUR per slice (= per person). Minimum 6 people = 30 EUR minimum cake. Delivery fee: 5 EUR if order total < 50 EUR, free otherwise. An order can contain multiple cakes.

**Phase cleanup rule:** When a phase is completed, replace its detailed steps with a short summary of what was done. Keep it concise.

**Three protocols demonstrated:**
- **A2A** (Agent-to-Agent): Cross-system communication between Commerce and Supply Chain
- **MCP** (Model Context Protocol): Google Maps integration for route optimization
- **UCP** (Universal Commerce Protocol): Agentic storefront -- external AI agents can discover and order cakes

## Architecture

```
            ┌────────────────────────────────────────────────────────┐
            │                   External AI Agents                    │
            │         (Gemini in Search, Gemini App, etc.)            │
            └───────────────────────┬────────────────────────────────┘
                                    │ UCP (/.well-known/ucp)
            ┌───────────────────────┴────────────────────────────────┐
            │                  React UI (SPA)                         │
            │  Infra | Arch | Magic Cake Shop | Backoffice | About    │
            └───────────────────────┬────────────────────────────────┘
                                    │ REST + WebSocket
            ┌───────────────────────┴────────────────────────────────┐
            │             Go CLI (k8s-lab binary)                     │
            │        Cobra commands + HTTP API server                  │
            └───────┬───────────────────────┬────────────────────────┘
                    │                       │
       ┌────────────┴──────┐     ┌──────────┴──────────────┐
       │  Commerce         │     │  Supply Chain            │
       │  Concierge        │◄───►│  Intelligence            │
       │  (System A)       │ A2A │  (System B)              │
       │  Port 8001        │     │  Port 8002               │
       ├───────────────────┤     ├──────────────────────────┤
       │ Translation       │     │ Inventory                │
       │ Cake Designer     │     │ Order Service            │
       │ Checkout          │     │ Fulfillment (Maps MCP)   │
       │ UCP Server        │     │                          │
       └───────────────────┘     └──────────────────────────┘
                │                          │
       ┌────────┴──────────────────────────┴──────────┐
       │                 Shared Redis                   │
       │    (inventory + orders + sessions + carts)     │
       └──────────────────────┬───────────────────────┘
                              │
       ┌──────────────────────┴───────────────────────┐
       │         Existing GCS Bucket                    │
       │   gs://{project}-k8s-lab/cakes/               │
       │   cakes/orders/{order-id}/cake.png            │
       └──────────────────────────────────────────────┘
```

**6 Agents + UCP Server across 2 Systems:**

| # | System | Agent | Model | Key Tools/Tech |
|---|--------|-------|-------|----------------|
| 1 | Commerce | Translation | gemini-2.5-flash | Gemini native multilingual (EN/DE/NL/TR) |
| 2 | Commerce | Cake Designer | gemini-2.5-pro | Redis (inventory check via A2A), Imagen/Banana Pro (cake image) |
| 3 | Commerce | Checkout | gemini-2.5-pro | Address validation, fake payment, A2A to Order Service + Inventory |
| - | Commerce | UCP Server | - | `/.well-known/ucp` discovery, checkout sessions, agentic storefront |
| 4 | Supply Chain | Inventory | gemini-2.5-flash | Redis (stock: chocolate, ananas, banana, walnut, almond) |
| 5 | Supply Chain | Order Service | gemini-2.5-flash | Redis (order CRUD), GCS (cake images) |
| 6 | Supply Chain | Fulfillment | gemini-2.5-pro | Google Maps MCP (route optimization from Danzigerkade 4) |

**Language:** Python ADK v1.25 for agents, Go for CLI/UI.

## Protocol Summary

### A2A (Phase 4) -- Inter-System Communication
Commerce and Supply Chain talk via A2A protocol:
- Cake Designer → Inventory: check ingredient availability before offering options
- Checkout → Inventory: deduct stock after payment
- Checkout → Order Service: create order with all details + image
- Mechanically: HTTP POST to other system's `/run` endpoint with ADK message format

### MCP (Phase 2) -- External Tool Integration
Fulfillment Agent uses Google Maps MCP via ADK's `MCPToolset`:
- Calculates optimal delivery route from Danzigerkade 4, 1013 AP Amsterdam
- Multi-stop route optimization for all deliveries on a given day
- Wraps Google Maps Directions/Distance Matrix/Geocoding APIs
- Requires `GOOGLE_MAPS_API_KEY` env var

### UCP (Phase 3) -- Agentic Storefront
Magic Cake publishes a UCP business profile so external AI agents can discover and order cakes:
- `/.well-known/ucp` manifest: declares capabilities (cake-ordering, delivery-scheduling, flavor-catalog)
- `POST /checkout-sessions`: external agent creates an order session with line items (flavor, nuts, people, concept)
- `PUT /checkout-sessions/{id}`: modify session (apply delivery date, address)
- Session completion triggers the same internal flow: inventory check → image gen → order creation
- NOT just payment -- covers full commerce lifecycle: discovery, configuration, ordering, fulfillment status
- Demonstrates how a small business becomes "agent-ready" for Gemini in Search, Gemini App, etc.

## Pricing

| Item | Price |
|------|-------|
| Per slice (= per person) | 5 EUR |
| Minimum cake | 6 people = 30 EUR |
| Maximum cake | 50 people = 250 EUR |
| Delivery fee | 5 EUR if order total < 50 EUR, free if >= 50 EUR |
| Multiple cakes | Same order can have multiple cakes (>50 people = suggest 2 cakes) |

Example: 1 chocolate cake for 8 people + 1 banana cake for 12 people = (8*5) + (12*5) = 100 EUR, free delivery.
Example: 1 cake for 6 people = 30 EUR + 5 EUR delivery = 35 EUR.

## Customer Flow (Conversational)

```
Customer opens Magic Cake Shop chat
    │
    ▼
Translation Agent: "Welcome! Choose language: English, German, Dutch, Turkish"
    │
    ▼ (all subsequent messages in chosen language)
    │
Cake Designer Agent (per cake, can repeat for multiple cakes):
    ├─ "What flavor? Chocolate, Ananas, Banana" (checks inventory via A2A first, hides out-of-stock)
    ├─ "Any nuts? Almond, Walnut, No nuts" (checks inventory via A2A)
    ├─ "How many people? (6-50)" (>50 suggests splitting into 2 cakes)
    ├─ "Any concept? Birthday text, Star Wars theme, etc."
    ├─ Generates cake image via Imagen → shows to customer
    ├─ "Do you approve this design?"
    └─ "Would you like to add another cake to this order?"
    │
    ▼
Checkout Agent:
    ├─ Shows order summary: each cake with price (people × 5 EUR)
    ├─ Shows delivery fee (5 EUR if total < 50 EUR, free otherwise)
    ├─ "Delivery address? (Amsterdam only)" → validates postcode 1000-1109
    ├─ "Delivery date?" → only next 3 days
    ├─ "Proceed to payment?" → fake payment (always succeeds)
    ├─ A2A → Inventory: deduct ingredients for all cakes
    ├─ A2A → Order Service: create order with all cake images (cake_1.png, cake_2.png, ...)
    └─ "Order confirmed! Your cake(s) will be delivered on [date]"
```

## UCP Flow (Programmatic -- External Agent)

```
External AI Agent (e.g., Gemini in Search)
    │
    ▼
GET /.well-known/ucp → discovers Magic Cake capabilities + catalog
    │
    ▼
POST /checkout-sessions → {cakes: [{flavor: "chocolate", nuts: "walnut", people: 12, concept: "birthday"}]}
    │ Returns: session_id, available_dates, pricing (12 × 5 = 60 EUR, free delivery)
    ▼
PUT /checkout-sessions/{id} → {delivery_date: "2026-02-15", postcode: "1013 AP", house_number: "42"}
    │ Returns: updated session with total, delivery confirmation
    ▼
POST /checkout-sessions/{id}/complete → triggers internal flow
    │ Inventory check → Image generation → Order creation → Stock deduction
    ▼
Returns: order confirmation with cake image URL, delivery details
```

## Image Storage (Existing GCS Bucket)

| Aspect | Detail |
|--------|--------|
| Bucket | Existing `{project}-k8s-lab` (same as Terraform state) |
| Prefix | `cakes/orders/{order-id}/cake_N.png` |
| Order ID | `CAKE-{YYYYMMDD}-{4-char}` (e.g., `CAKE-20260213-A3F2`) |
| Single cake | `gs://{project}-k8s-lab/cakes/orders/CAKE-20260213-A3F2/cake_1.png` |
| Multiple cakes | `cake_1.png`, `cake_2.png`, etc. per order |
| Redis ref | Order hash has `image_paths` field → comma-separated GCS paths |
| Cleanup | `k8s-lab cleanup-cakes --cloud gcp` cross-refs Redis orders vs GCS `/cakes/` prefix, removes orphans |
| Destroy | `destroy` does NOT touch GCS → images survive for restore |
| No new bucket needed | Reuses existing infra, `/cakes/` prefix keeps it organized |

## Directory Structure (Final State)

```
k8s-lab/
├── agents/                          # Python ADK agents
│   ├── commerce/                    # System A - Commerce Concierge
│   │   ├── pyproject.toml
│   │   ├── Dockerfile
│   │   ├── main.py                  # A2A server on :8001 + UCP endpoints
│   │   ├── agents/
│   │   │   ├── translation.py       # Language selection (EN/DE/NL/TR)
│   │   │   ├── cake_designer.py     # Cake preferences + Imagen generation
│   │   │   └── checkout.py          # Address, delivery, payment, order creation
│   │   ├── tools/
│   │   │   ├── image_gen.py         # Imagen / Banana Pro → GCS
│   │   │   ├── address.py           # Amsterdam postcode validation
│   │   │   └── payment.py           # Fake payment processor
│   │   ├── ucp/
│   │   │   ├── manifest.py          # /.well-known/ucp capability declaration
│   │   │   ├── sessions.py          # Checkout session management (create/update/complete)
│   │   │   └── catalog.py           # Flavor catalog from inventory
│   │   └── a2a/
│   │       └── supply_chain_client.py  # A2A calls to System B
│   ├── supply_chain/                # System B - Supply Chain Intelligence
│   │   ├── pyproject.toml
│   │   ├── Dockerfile
│   │   ├── main.py                  # A2A server on :8002
│   │   ├── agents/
│   │   │   ├── inventory.py         # Stock management (5 ingredients)
│   │   │   ├── order_service.py     # Order CRUD + image references
│   │   │   └── fulfillment.py       # Route planning from Danzigerkade 4
│   │   ├── tools/
│   │   │   ├── redis_stock.py       # Redis inventory ops
│   │   │   ├── redis_orders.py      # Redis order ops
│   │   │   ├── gcs_images.py        # GCS image upload/retrieval
│   │   │   └── maps.py             # Google Maps MCP wrapper
│   │   └── a2a/
│   │       └── commerce_client.py   # A2A calls to System A
│   └── shared/                      # Shared config + Redis client
│       ├── pyproject.toml
│       ├── config.py
│       └── redis_client.py
├── apps/agents/                     # K8s manifests for agents
│   ├── namespace.yaml
│   ├── configmap.yaml
│   ├── commerce.yaml
│   └── supply-chain.yaml
├── cli/cmd/                         # Extended Go CLI
│   ├── deploy_agents.go             # NEW
│   ├── seed_inventory.go            # NEW
│   ├── agent_chat.go                # NEW
│   ├── cleanup_cakes.go             # NEW
│   └── ... (existing)
├── cli/pkg/ui/
│   ├── handlers_agents.go           # NEW: Agent + shop + backoffice API endpoints
│   └── ... (existing)
├── ui/frontend/src/components/
│   ├── ShopPage.tsx                 # NEW: Magic Cake Shop (chat)
│   ├── BackofficePage.tsx           # NEW: Orders, map, inventory, revenue, agent log
│   ├── AgentChat.tsx                # NEW: Reusable chat widget
│   └── ... (existing)
├── build-ui.sh
├── agent_plan.md                    # This plan
└── ... (existing: infra/, configs/, cli/, etc.)
```

## Phase Dependencies

```
Phase 0 ──► Phase 1 ──► Phase 2 ──► Phase 4 ──► Phase 5 ──► Phase 6 ──► Phase 7
                    └──► Phase 3 ──┘
```

---

## Phase 0: Cleanup -- Remove Bash

**Status:** Complete

**What was done:**
- Removed `scripts/` directory (all bash scripts)
- Removed `Makefile`
- Made Go CLI (`k8s-lab` binary) the sole interface for all operations
- Created `build-ui.sh` for UI build workflow
- Fixed `findRepoRoot()` in Go CLI to properly locate repository root
- Updated all documentation to reference Go CLI commands instead of make/scripts
- Removed references to old bash interface from README.md and CLAUDE.md

**Result:** Clean, single-interface project. All operations via `./k8s-lab <command>`

---

## Phase 1: Foundation

**Status:** Complete

**What was done:**
- Created directory structure for two agent systems:
  - `agents/commerce/` - System A (Commerce Concierge)
  - `agents/supply_chain/` - System B (Supply Chain Intelligence)
  - `agents/shared/` - Shared configuration and Redis client
- Implemented Python ADK scaffolding:
  - Agent definitions: translation, cake_designer, checkout (Commerce); inventory, order_service, fulfillment (Supply Chain)
  - Tool directories: `tools/`, `a2a/`, `ucp/` (Commerce only)
  - Package configuration: `pyproject.toml` per system with setuptools
  - Main entry points: `main.py` for A2A server setup
- Created Dockerfiles for both systems
- Created Kubernetes manifests in `apps/agents/`:
  - `namespace.yaml` - agents namespace
  - `configmap.yaml.example` - environment variables (project ID, region, bucket, API keys)
  - `commerce.yaml.example` - Commerce deployment + service (port 8001)
  - `supply-chain.yaml.example` - Supply Chain deployment + service (port 8002)
  - `gcr-credential-sync.yaml` - DaemonSet for image pull secrets
- All `.example` files are templates - actual configs gitignored for public repo
- Created agents/README.md with local development guide

**Total files:** 32 files across scaffolding, tools, agents, manifests

**Result:** Full ADK project structure ready for implementation. Public repo safe (no secrets).

---

## Phase 2: Supply Chain Intelligence (System B) -- Build First

**Goal:** 3 working agents with real tools, testable via Go CLI.

### Steps

**2.1 Inventory Agent tools** -- `agents/supply_chain/tools/redis_stock.py`:
- `check_stock(item)` -- HGET inventory:{item} quantity. Items: chocolate, ananas, banana, walnut, almond
- `update_stock(item, quantity_change, reason)` -- HINCRBY inventory:{item} quantity + RPUSH inventory:log
- `list_all_stock()` -- HGETALL for all 5 items, return availability overview
- `list_low_stock(threshold)` -- Filter items with quantity <= threshold
- Max stock per item: 5. Agent instruction enforces this.

**2.2 Order Service Agent tools** -- `agents/supply_chain/tools/redis_orders.py`:
- `create_order(customer_name, cakes: [{flavor, nuts, people_count, concept}], address, postcode, delivery_date, image_paths) -- Price and delivery fee calculated internally.` -- HSET order:{CAKE-YYYYMMDD-XXXX} with all fields. Price = sum(people × 5 EUR per cake). Delivery fee = 5 EUR if total < 50, else 0.
- `get_order(order_id)` -- HGETALL order:{id}
- `list_orders(delivery_date?)` -- SCAN order:CAKE-*, optionally filter by delivery_date
- `delete_order(order_id)` -- DEL order:{id} + delete GCS image
- `get_order_stats()` -- Count orders, sum revenue, average price

**2.2b Order Service image tools** -- `agents/supply_chain/tools/gcs_images.py`:
- `upload_cake_image(order_id, cake_number, image_bytes)` -- Upload to gs://{bucket}/cakes/orders/{order_id}/cake_{N}.png
- `get_cake_image_urls(order_id)` -- Return signed URLs for all cake images in order
- `delete_cake_images(order_id)` -- Delete all images for order from GCS
- `list_orphan_images()` -- Compare GCS objects under /cakes/ vs Redis orders, return orphans

**2.3 Fulfillment Agent** -- Google Maps MCP integration:
- Use ADK's `MCPToolset` to connect to Google Maps MCP server
- Agent instruction: "You are a delivery route planner for Magic Cake. Our fulfillment center is at Danzigerkade 4, 1013 AP Amsterdam. Plan optimal routes visiting all delivery addresses for a given day."
- Tools via MCP: route calculation, delivery time estimation, multi-stop optimization
- Custom tool: `get_orders_for_date(date)` -- Fetches all orders for a delivery date from Redis
- Agent reasons about optimal visiting order and uses Maps MCP to compute the actual route
- Requires `GOOGLE_MAPS_API_KEY` environment variable

**2.4 Wire tools into agent definitions** -- Update inventory.py, order_service.py, fulfillment.py to import and register tools

**2.5 Go CLI: `deploy-agents` command** -- `cli/cmd/deploy_agents.go`:
- Docker build from repo root (needs agents/shared/ context)
- Docker push to Artifact Registry (`us-central1-docker.pkg.dev/{project}/k8s-lab/{name}:latest`)
- Patch configmap with actual project ID, region, bucket name, maps API key
- Apply K8s manifests (namespace, configmap, commerce, supply-chain)
- Wait for pods ready
- Follow pattern from `deploy_applications.go`

**2.6 Go CLI: `seed-inventory` command** -- `cli/cmd/seed_inventory.go`:
- Follow pattern from `seed_redis.go` (get pod, exec redis-cli via stdin)
- Seed inventory (5 items, max 5 per type, some deliberately low):
  - chocolate: qty 4
  - ananas: qty 1 (LOW -- forces unavailability in conversation)
  - banana: qty 3
  - walnut: qty 2 (LOW -- forces limited availability)
  - almond: qty 4
- Seed 7 fake orders across next 3 days (4 + 2 + 1):
  - Valid Amsterdam addresses (real streets: Herengracht 502, Prinsengracht 263, Keizersgracht 174, Damrak 1, Rokin 92, Singel 140, Amstel 51)
  - Customer names (Dutch-style: Jan de Vries, Maria van den Berg, etc. or internaltional John Doe, Mary Steling)
  - Cake details: random flavor + nuts + people count (6-50) + concept (birthday, wedding, baby shower, etc.)
  - Prices calculated from formula
  - Pre-generated cake images uploaded to GCS during seed
  - Delivery dates computed from current day: today+1 (4 orders), today+2 (2 orders), today+3 (1 order)
  - Order IDs: CAKE-{YYYYMMDD}-{XXXX}

**2.7 Register new operations** in `cli/pkg/ui/server.go` and `cli/pkg/ui/handlers.go`:
- Add `deploy-agents`, `seed-inventory`, `cleanup-cakes` to allowed operations map

**2.8 Go CLI: `cleanup-cakes` command** -- `cli/cmd/cleanup_cakes.go`:
- List all GCS objects under `cakes/` prefix in existing bucket
- List all order IDs in Redis
- Delete GCS objects not referenced by any order
- Report: "Cleaned X orphan images, Y images retained"

### Verification
**Status:** ✅ Phase 2 Complete - agents deployed and seeded

**Deployed:**
- Supply Chain agents running in `agents` namespace
- Commerce agents scaffolded (tools not implemented)
- Health checks passing (`/health` endpoint via Starlette)
- Data seeded (5 ingredients, 7 orders with GCS images)

**A2A Testing Note:** A2A agents cannot be tested with curl `/run` endpoint. They use `RemoteA2aAgent` for inter-agent communication. Testing deferred to Phase 4 when Commerce calls Supply Chain.

### TODOs / Known Issues

**Phase 2 Improvements (defer to later phases):**
- **Logging:** deploy-agents has too many DEBUG logs. Should promote 8 key milestones to INFO (namespace applied, deployment created, agents ready, etc.) and add deployment summary with access instructions.
- **Redis Namespace:** Currently cross-namespace (Redis in `application`, agents in `agents`). Consider moving Redis to `agents` namespace for cleaner isolation. Decision needed before Phase 4 (A2A will intensify Redis usage).
- **GCS Cleanup:** `destroy` command does not clean GCS bucket (intentional for backup/restore). Consider adding `--clean-gcs` flag as opt-in.

**Phase 3 Requirements (next phase):**
- Implement Commerce tools: `image_gen.py` (Imagen API), `address.py`, `payment.py`
- Implement UCP endpoints: manifest, catalog, sessions
- Add `google-cloud-aiplatform` and `fastapi` dependencies to commerce pyproject.toml

### Critical Files
- `cli/cmd/deploy_applications.go` -- Pattern for deploy_agents.go
- `cli/cmd/seed_redis.go` -- Pattern for seed_inventory.go
- `cli/pkg/ui/handlers.go` -- Add to allowed operations

---

## Phase 3: Commerce Concierge + UCP (System A) -- Build Second

**Goal:** 3 working agents + UCP agentic storefront. Customer-facing conversational cake ordering and programmatic agent access.

### Steps

**3.1 Translation Agent** -- `agents/commerce/agents/translation.py`:
- No custom tools. Gemini Flash handles translation natively.
- Agent instruction: "You are the first point of contact for Magic Cake shop. Greet the customer and ask them to choose a language: English, German (Deutsch), Dutch (Nederlands), or Turkish (Turkce). Once chosen, ALL subsequent messages in the conversation must be in that language. Pass the customer to the Cake Designer agent after language is set."
- Stores language choice in session state (Redis key: session:{id}:language)

**3.2 Cake Designer Agent** -- `agents/commerce/agents/cake_designer.py`:
- Agent instruction: "You help customers design their dream cake. Ask questions one by one in the customer's chosen language. Check ingredient availability before offering options. Do not offer out-of-stock ingredients."
- Tools:
  - `check_ingredient_available(item)` -- A2A call to Inventory agent (Phase 4 wires this, stub in Phase 3 returns mock data)
  - `generate_cake_image(description)` -- Calls Imagen/Banana Pro API with cake description, uploads to GCS, returns image URL
- Flow enforced by instruction:
  1. Ask flavor (only offer in-stock: chocolate/ananas/banana)
  2. Ask nuts (only offer in-stock: almond/walnut/none)
  3. Ask people count (6-50, >50 suggest 2 cakes)
  4. Ask concept/theme (free text: birthday, Star Wars, etc.)
  5. Generate image, show to customer, ask approval
  6. If approved → hand off to Checkout

**3.2b Image generation tool** -- `agents/commerce/tools/image_gen.py`:
- `generate_cake_image(flavor, nuts, people_count, concept, order_id, cake_number)`:
  - Constructs prompt: "A beautiful {flavor} cake for {people_count} people with {concept} theme, {nuts} decoration, professional bakery photo"
  - Calls Imagen API (Vertex AI) to generate image
  - Uploads to GCS: `gs://{bucket}/cakes/orders/{order_id}/cake_{N}.png`
  - Returns signed URL for display
  - Called once per cake in the order (cake_1, cake_2, etc.)

**3.3 Checkout Agent** -- `agents/commerce/agents/checkout.py`:
- Agent instruction: "You handle delivery and payment for Magic Cake. Only deliver in Amsterdam. Validate the address. Collect delivery date (next 3 days only). Show order summary with price. Process fake payment."
- Tools:
  - `validate_amsterdam_address(postcode, house_number)` -- Checks postcode in range 1000-1109, format NNNN XX
  - `get_available_delivery_dates()` -- Returns next 3 days as options
  - `calculate_price(cakes: [{people_count}])` -- Sum of (people_count × 5 EUR) per cake. Delivery fee: 5 EUR if total < 50 EUR, free otherwise.
  - `process_payment(order_id, amount)` -- Always succeeds, returns fake transaction ID (PAY-{timestamp})
  - A2A tools (Phase 4): `deduct_stock(items[])`, `create_order(details)`

**3.3b Address validation tool** -- `agents/commerce/tools/address.py`:
- `validate_amsterdam_address(postcode, house_number)`:
  - Dutch format: 4 digits + space + 2 uppercase letters (e.g., "1013 AP")
  - Amsterdam range: 1000-1109
  - Returns: valid/invalid + formatted address string
  - Out of Amsterdam: "Sorry, Magic Cake only delivers in Amsterdam"

**3.3c Payment tool** -- `agents/commerce/tools/payment.py`:
- `process_payment(order_id, amount, customer_name)`:
  - Generates fake transaction ID: `PAY-{timestamp}`
  - Returns: {success: true, transaction_id, amount, message: "Payment processed"}
  - Always succeeds (PoC)

**3.4 UCP Agentic Storefront** -- `agents/commerce/ucp/`:
This is the non-conversational, programmatic interface. External AI agents (Gemini in Search, Gemini App, any UCP-compatible agent) discover and order cakes without chat.

**3.4a `manifest.py`** -- UCP capability declaration:
- Serves `/.well-known/ucp` endpoint on Commerce system
- JSON manifest:
```json
{
  "name": "Magic Cake Amsterdam",
  "description": "Custom cake ordering and delivery in Amsterdam",
  "capabilities": ["dev.ucp.shopping", "dev.ucp.checkout"],
  "services": {
    "catalog": {"endpoint": "/ucp/catalog", "version": "1.0"},
    "checkout": {"endpoint": "/ucp/checkout-sessions", "version": "1.0"}
  }
}
```

**3.4b `catalog.py`** -- Flavor catalog for discovery:
- `GET /ucp/catalog` -- Returns available flavors, nuts, people range, pricing formula
- Queries Inventory (same as Cake Designer) to only show in-stock items
- Response includes: items with availability, pricing rules, delivery area (Amsterdam only), delivery dates (next 3 days)

**3.4c `sessions.py`** -- Checkout session management:
- `POST /ucp/checkout-sessions` -- Create session with: cakes[{flavor, nuts, people_count, concept}], customer_name
  - Validates ingredients available, calculates price (sum of people × 5 EUR + delivery fee), returns session_id + pricing + available delivery dates
- `PUT /ucp/checkout-sessions/{id}` -- Update with: delivery_date, postcode, house_number
  - Validates Amsterdam address, calculates final price, returns updated session
- `POST /ucp/checkout-sessions/{id}/complete` -- Finalize order
  - Triggers: inventory check → image generation → fake payment → order creation → stock deduction
  - Returns: order confirmation with image URL, delivery details, transaction ID
- `GET /ucp/checkout-sessions/{id}` -- Check session status

**3.5 Wire all tools** into translation.py, cake_designer.py, checkout.py
- A2A tools are stubs in this phase (return mock data)
- Real A2A wiring happens in Phase 4
- UCP endpoints wire to the same underlying tools as conversational flow

### Verification
```bash
cd agents/commerce && python -c "from agents.translation import translation_agent; print(translation_agent.name)"
cd agents/commerce && python -c "from agents.cake_designer import cake_designer_agent; print(cake_designer_agent.name)"

# With running cluster:
./bin/k8s-lab deploy-agents --cloud gcp

# Test conversational flow:
kubectl port-forward -n agents svc/commerce 8001:8001
curl -X POST http://localhost:8001/run -d '{"app_name":"commerce","user_id":"test","session_id":"s1","new_message":{"role":"user","parts":[{"text":"I want to order a cake"}]}}'

# Test UCP discovery:
curl http://localhost:8001/.well-known/ucp
curl http://localhost:8001/ucp/catalog

# Test UCP checkout session:
curl -X POST http://localhost:8001/ucp/checkout-sessions \
  -d '{"flavor":"chocolate","nuts":"walnut","people_count":12,"concept":"birthday","customer_name":"Test User"}'
```

---

## Phase 4: A2A Integration

**Goal:** Wire Commerce and Supply Chain systems together via A2A. Add agent-chat CLI.

### Steps

**4.1 Commerce → Supply Chain A2A client** -- `agents/commerce/a2a/supply_chain_client.py`:
- `check_stock(item)` -- POST to Supply Chain /run, ask Inventory agent about item availability
- `deduct_stock(items: list[str])` -- POST to Supply Chain /run, tell Inventory to deduct items
- `create_order(customer_name, cake_details, address, delivery_date, price, image_path)` -- POST to Supply Chain /run, tell Order Service to create order
- Register as tools on Cake Designer (check_stock) and Checkout (deduct_stock, create_order)
- Replace stub implementations from Phase 3
- UCP endpoints also use these A2A clients (same flow, different entry point)

**4.2 Supply Chain → Commerce A2A client** -- `agents/supply_chain/a2a/commerce_client.py`:
- `notify_out_of_stock(item)` -- POST to Commerce /run, notify that item is unavailable
- Register as tool on Inventory agent
- Used when stock hits 0 after deduction

**4.3 Go CLI: `agent-chat` command** -- `cli/cmd/agent_chat.go`:
- `k8s-lab agent-chat --system commerce --cloud gcp "I want to order a cake"`
- `k8s-lab agent-chat --system supply-chain --cloud gcp "What is current stock?"`
- Opens tunnel to agent pod, HTTP POST to /run endpoint
- Supports multi-turn: `--session <id>` flag to continue conversation
- Prints agent response to stdout

### Verification
```bash
# Cake Designer checks inventory via A2A:
./bin/k8s-lab agent-chat --cloud gcp --system commerce "I want a chocolate cake with walnuts for 10 people"
# Should offer chocolate (in stock) and walnuts (low stock, still available)
# Should NOT offer ananas (qty=1, may be unavailable)

# Full order flow with A2A:
./bin/k8s-lab agent-chat --cloud gcp --system commerce --session order1 "I want to order a cake"
# Follow conversation: language → flavor → nuts → people → concept → image → address → date → payment
# Verify: order created in Redis, stock deducted, image in GCS

# UCP flow with real A2A:
curl -X POST http://localhost:8001/ucp/checkout-sessions \
  -d '{"flavor":"chocolate","nuts":"almond","people_count":8,"concept":"star wars","customer_name":"Agent Test"}'
# Verify: inventory checked via A2A, session created

# Supply Chain direct:
./bin/k8s-lab agent-chat --cloud gcp --system supply-chain "Show me all orders for tomorrow"
./bin/k8s-lab agent-chat --cloud gcp --system supply-chain "Plan delivery route for tomorrow"
```

---

## Phase 5: UI Elevation

**Goal:** Magic Cake Shop page (customer chat), Backoffice page (orders, map, inventory, revenue, agent log). Rename existing dashboard to Infrastructure.

### Steps

**5.1 Update navigation in App.tsx**:
- Rename current dashboard view to "Infrastructure"
- Navigation tabs next to k8s-lab logo: `Infrastructure | Architecture | Magic Cake Shop | Magic Cake Backoffice | About`
```typescript
type View = 'infrastructure' | 'pod-detail' | 'tf-detail' | 'about' | 'architecture'
           | 'shop' | 'backoffice'
```

**5.2 New Go API endpoints** -- `cli/pkg/ui/handlers_agents.go`:
- `POST /api/agent/chat` -- Send message to agent system, return response (body: {system, message, session_id})
- `GET /api/agent/status` -- Agent pod status for both systems
- `GET /api/inventory` -- All 5 ingredient stock levels
- `GET /api/orders?date=YYYY-MM-DD` -- Orders, optionally filtered by delivery date
- `DELETE /api/orders/:id` -- Delete an order (+ GCS image cleanup)
- `GET /api/orders/:id/image` -- Proxy cake image from GCS (signed URL redirect)
- `GET /api/fulfillment/route?date=YYYY-MM-DD` -- Get optimized delivery route for date (proxies to Fulfillment agent)
- `GET /api/orders/stats` -- Order count, total revenue, average price
- `GET /api/agent/activity` -- Recent agent interactions log

**5.3 ShopPage.tsx** -- Magic Cake Shop (customer-facing):
- Fancy landing section: shop name "Magic Cake", tagline, cake imagery/branding
- Chat interface (AgentChat component) connected to Commerce system
- Shows cake image inline when agent generates one
- Clean, modern bakery aesthetic

**5.4 AgentChat.tsx** -- Reusable chat widget:
- Message input, response display with markdown rendering
- Image display support (for cake previews)
- System selector prop (commerce/supply-chain)
- Session management (generates session ID, supports multi-turn)
- Loading states, error handling

**5.5 BackofficePage.tsx** -- Magic Cake Backoffice (5 components):

**5.5a Map View Component**:
- Dropdown to select date (next 3 days)
- Interactive map showing delivery route from Danzigerkade 4 to all delivery addresses
- Route data from Fulfillment agent via `/api/fulfillment/route`
- Map library: Leaflet.js (open source, no API key for tiles, route polyline overlay)
- Markers for each delivery address with order details tooltip

**5.5b Order Table Component**:
- Table columns: Order ID, Customer Name, Cakes (count + summary), Address, Delivery Date, Price (cakes + delivery fee), Image thumbnails
- Delete button per row with confirmation
- Filter dropdown by delivery date
- Click row to expand: full cake details (flavor, nuts, people per cake), full-size images (cake_1, cake_2, ...), delivery fee breakdown

**5.5c Inventory Dashboard Component**:
- Visual stock levels for all 5 ingredients (chocolate, ananas, banana, walnut, almond)
- Color-coded progress bars: green (3-5), yellow (2), red (0-1)
- Shows: current quantity / max (5) per ingredient
- Auto-refreshes every 30 seconds

**5.5d Revenue Summary Component**:
- Stats cards: Total orders count, Total revenue, Average order value
- Simple data from `/api/orders/stats`
- Compact display at top of backoffice

**5.5e Agent Activity Log Component**:
- Recent agent interactions: timestamp, system (commerce/supply-chain), user query, agent action
- Shows A2A calls between systems
- Scrollable feed, max 50 recent entries
- Helps visualize the agentic flow in real-time

**5.5f Agent Chat Panel**:
- Embedded AgentChat widget connected to Supply Chain system
- For ad-hoc backoffice queries: "How many orders for tomorrow?", "Plan route for Friday", "What ingredients are low?"
- Collapsed by default, expandable

**5.6 Update ActionsPanel** -- Add "Deploy Agents" and "Seed Inventory" buttons to Infrastructure page

**5.7 Map integration**: Leaflet.js with OpenStreetMap tiles. Route polyline from Fulfillment agent data. No extra API key needed for map rendering (Google Maps API only needed server-side for route calculation via MCP).

### Verification
```bash
./build-ui.sh
./bin/k8s-lab ui --cloud gcp
# Test: Navigate to Magic Cake Shop, order a cake through chat
# Test: Navigate to Backoffice, view orders in table
# Test: Check inventory dashboard colors
# Test: Select delivery date, see route on map
# Test: Delete an order from table, verify removal
# Test: Check revenue stats
# Test: View agent activity log
# Test: Use backoffice agent chat for ad-hoc queries
# Test: Infrastructure tab still works as before
```

---

## Phase 6: Deployment and Lifecycle

**Goal:** Production-ready pipeline, unified CLI commands, seed data with images, full lifecycle.

### Steps

**6.1** Add Artifact Registry to Terraform (`infra/gcp/terraform/`):
```hcl
resource "google_artifact_registry_repository" "agents" {
  repository_id = "k8s-lab"
  location      = var.region
  format        = "DOCKER"
}
```

**6.2** Update backup scope -- Add `agents` namespace to default Velero backup in `cli/cmd/backup.go`

**6.3** Merge `deploy-agents` into `deploy-applications`:
- `deploy-applications` now deploys: NGINX + Redis + agent containers (build, push, apply manifests)
- `deploy-agents` stays as standalone for dev/testing but `deploy-applications` calls it internally
- `deploy` (all-in-one) = infra + tools + applications (which includes agents)

**6.4** Update `destroy` command to clean up agents namespace

**6.5** Replace `seed-redis` + `seed-inventory` with unified `seed-data`:
- `k8s-lab seed-data --cloud gcp` does everything:
  - Seeds Redis test data (existing seed-redis logic)
  - Seeds inventory (5 ingredients with deliberate low stock)
  - Seeds 7 fake orders with valid Amsterdam addresses, computed delivery dates
  - Generates cake images via Imagen and uploads to GCS (cake_1.png per order)
- Makes backoffice immediately useful after seeding
- Old `seed-redis` and `seed-inventory` deprecated (kept as aliases initially, removed later)

**6.6** Full daily lifecycle:
```
deploy → seed-data → [use shop + backoffice] → backup → destroy
restore (next day) → [agents + orders + images all restored/available]
```

### CLI Command Evolution

| Before (Phase 2-4) | After (Phase 6) | Notes |
|---------------------|-----------------|-------|
| `deploy-agents` | Part of `deploy-applications` | Still available standalone for dev |
| `seed-redis` + `seed-inventory` | `seed-data` | Single command, all test data |
| `deploy` | infra + tools + apps (incl. agents) | Unchanged intent, agents included |

### Verification
```bash
./bin/k8s-lab deploy --cloud gcp
./bin/k8s-lab seed-data --cloud gcp
./bin/k8s-lab backup --cloud gcp
./bin/k8s-lab destroy --cloud gcp
# Next day
./bin/k8s-lab deploy-infra --cloud gcp
./bin/k8s-lab restore --cloud gcp
kubectl get pods -n agents  # Agents restored
# Verify: orders in Redis, images in GCS, backoffice shows data
```

---

## Phase 7: Documentation

**Goal:** Comprehensive docs, project history, lessons learned.

### Steps

**7.1** Update `README.md`:
- Add Magic Cake Shop section with architecture overview
- Update Quick Start for agent deployment
- Add to Lessons Learned table
- Update architecture diagram
- Document all three protocols: A2A, MCP, UCP

**7.2** Create `agents/README.md`:
- How-to guide for agent development
- Local testing workflow
- A2A, MCP, UCP reference
- Environment variables reference

**7.3** Final pass on all docs: `CLAUDE.md`, `GEMINI.md`, `LOCAL.md`, `ui/plan.md`

**7.4** Create `project-history.md` -- Timeline of project evolution. This should be HTML file and part of the UI. It should be visual timeline with lessons learned included. 

### Verification
```bash
grep -r "scripts/\|Makefile\|Sales Tracker\|BigQuery\|Vertex AI Search" README.md CLAUDE.md GEMINI.md  # Should be empty
./bin/k8s-lab --help  # All commands documented
```

---

## Implementation Order

1. Phase 0: Cleanup (DONE)
2. Phase 1: Foundation + commit agent_plan.md
3. Phase 2: Supply Chain agents -- **test alone via CLI before proceeding**
4. Phase 3: Commerce agents + UCP -- **test alone via CLI + curl for UCP**
5. Phase 4: A2A wiring -- **test both systems talking via CLI**
6. Phase 5: UI -- **test full flow via browser**
7. Phase 6: Deployment pipeline + seed images
8. Phase 7: Documentation

User requested: "first agent is done we test alone, then do second agent. all tests are via go cli then we will work on frontend."
