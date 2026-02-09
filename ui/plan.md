# UI Implementation Plan

## Overview

A local web dashboard for k8s-lab that provides a visual interface to all cluster lifecycle operations. Runs as `k8s-lab ui --cloud gcp` - single binary serving React frontend + Go API backend.

**Mockup:** Open `ui/mockup.html` in a browser to see all 8 interactive views.

---

## Current Status

**Phase:** Phase 7 Complete
**Branch:** `feature/ui`
**Next Step:** End-to-end user verification (backup/restore flow)

| Phase | Status |
|-------|--------|
| Phase 1: Foundation | Complete |
| Phase 2: Tunnel Manager + Status | Complete |
| Phase 3: Operations + Log Streaming | Complete |
| Phase 4: Data Panels | Complete |
| Phase 5: Polish & Missing Features | Complete |
| Phase 6: UX Polish (User Testing Feedback) | Complete |
| Phase 7: Logo Integration & Status Fixes | Complete |

---

## Architecture

```
Browser (localhost:3000)
    |
    v
Go HTTP Server (embedded in k8s-lab binary)
    |-- Serves React static files (embedded via go:embed)
    |-- REST API endpoints (/api/*)
    |-- WebSocket endpoint (/ws/logs) for live log streaming
    |-- Manages persistent IAP tunnel lifecycle
    |    \-- gcloud compute start-iap-tunnel (port 6443)
    |-- On shutdown (Ctrl+C): kills tunnel, cleans up env, exits
    \-- Signal handler: SIGINT/SIGTERM -> graceful cleanup
```

### How `k8s-lab ui` works

1. `k8s-lab ui --cloud gcp` starts the Go HTTP server
2. Server starts, opens browser to `http://localhost:3000`
3. **The terminal stays active** (like `npm start` or `make connect`)
4. Ctrl+C triggers graceful shutdown:
   - Kills persistent IAP tunnel subprocess
   - Unsets `K8SLAB_TUNNEL_MANAGED` (only relevant for child processes)
   - Closes HTTP server
   - Process exits cleanly
5. If terminal is closed or process is killed, Go's signal handler catches SIGTERM and cleans up tunnel

### Why single binary?
- React build output embedded into Go binary via `go:embed`
- No separate `npm start` needed in production
- For development: React dev server (port 5173) proxies API to Go (port 8080)

---

## Critical Design Decision: Tunnel Management

### The Problem

Current CLI pattern: every command creates its own IAP tunnel (10s startup), does work, tears down via `defer cleanup()`. If the UI keeps a persistent tunnel on port 6443 and a CLI command tries to open another, port conflicts.

### The Solution: `K8SLAB_TUNNEL_MANAGED` env var

**How it works:**

1. **UI starts** -> checks terraform state -> infra exists? -> starts persistent K8s tunnel (port 6443)
2. **UI calls CLI commands** with `K8SLAB_TUNNEL_MANAGED=true` set on the subprocess
3. **CLI tunnel logic** checks this env var:
   - Set -> skip K8s tunnel creation/cleanup, assume `localhost:6443` works
   - Not set -> current behavior (create + defer cleanup)
4. **Standalone CLI** (no UI) -> env var not set -> works exactly as before

**CLI change (~5 lines in `cli/pkg/cloud/gcp/provider.go`):**

```go
func (p *Provider) CreateK8sEndpoint(ctx context.Context, ...) (string, func(), error) {
    if os.Getenv("K8SLAB_TUNNEL_MANAGED") == "true" {
        p.log.Info("K8s tunnel externally managed, skipping creation")
        return "localhost:6443", func() {}, nil  // no-op cleanup
    }
    // ... existing tunnel code unchanged ...
}
```

**Talos tunnel (port 50000):** Only used during `deploy-infra` bootstrap. Always per-operation, never persistent. No conflict.

**Tunnel health:**
- Ping `localhost:6443` every 30 seconds
- If dead -> auto-restart (same gcloud command), frontend shows "Reconnecting..."
- Operations blocked while reconnecting
- After 5 failed reconnect attempts -> show error, user can manually retry

### Lifecycle

```
k8s-lab ui --cloud gcp
|
|-- Startup
|   |-- Check gcloud auth -> show user or error banner
|   |-- Read terraform.tfvars -> extract project_id, region
|   |-- Check terraform state -> infra exists?
|   |   |-- YES -> start persistent tunnel (6443) -> status: Connected
|   |   \-- NO  -> status: Idle (no tunnel)
|   \-- Start HTTP server on :3000, open browser
|
|-- User clicks "Deploy Infra"
|   |-- Spawns: K8SLAB_TUNNEL_MANAGED=true k8s-lab deploy-infra --cloud gcp
|   |-- deploy-infra manages its own Talos tunnel (50000) internally
|   |-- Streams stdout/stderr to WebSocket
|   \-- On success -> UI starts persistent K8s tunnel (6443)
|
|-- User clicks "Deploy Tools" (tunnel already alive)
|   |-- Spawns: K8SLAB_TUNNEL_MANAGED=true k8s-lab deploy-tools --cloud gcp
|   \-- CLI skips tunnel setup, uses existing localhost:6443
|
|-- Tunnel dies
|   |-- Health check detects failure
|   |-- Auto-restart -> status: Reconnecting...
|   \-- Reconnected -> status: Connected
|
\-- Ctrl+C
    |-- Signal handler fires
    |-- Kill tunnel subprocess (PID)
    |-- Close HTTP server
    \-- Exit 0
```

---

## UI States and Behavior

### State-Aware UI

The UI adapts based on three conditions: **auth**, **tunnel**, and **deployment stage**.

| State | Auth Check | Tunnel | Deploy Stage | UI Behavior |
|-------|-----------|--------|--------------|-------------|
| Not Authenticated | Failed | N/A | N/A | Red banner, ALL buttons disabled, error in log |
| Idle | OK | Idle | No infra | Only "Deploy All" and "Deploy Infra" enabled |
| Deploying | OK | Starting | In progress | Spinner in header, Cancel button, all else disabled |
| Running | OK | Connected | Full deploy | All features active |
| Tunnel Lost | OK | Disconnected | Was running | Yellow banner, K8s-dependent buttons disabled, data panels show "stale" |
| Deploy Failed | OK | Idle | Error | Red banner, "Retry" and "Destroy" buttons shown |

**Button enable/disable rules:**
- No auth -> everything disabled
- No infra -> only Deploy Infra / Deploy All
- No tunnel -> only Destroy All (gcloud/terraform don't need tunnel)
- No tools -> only Deploy Tools, Destroy
- No apps -> only Deploy Apps, Destroy
- Full deploy -> Seed Redis, Backup, Restore, Destroy enabled

---

## Folder Structure

```
k8s-lab/
|-- ui/
|   |-- plan.md              # This file
|   |-- mockup.html          # Interactive HTML mockup (7 views)
|   \-- frontend/            # React app (Vite + TypeScript)
|       |-- src/
|       |   |-- App.tsx
|       |   |-- main.tsx
|       |   |-- index.css           # Tailwind + dark/light theme vars
|       |   |-- components/
|       |   |   |-- CloudInfoPanel.tsx      # Phase 2
|       |   |   |-- StatusPanel.tsx         # Phase 2
|       |   |   |-- ActionsPanel.tsx        # Phase 3
|       |   |   |-- LogStream.tsx           # Phase 3
|       |   |   |-- NodesPanel.tsx          # Phase 4
|       |   |   |-- PodTable.tsx            # Phase 4
|       |   |   |-- PersistentDisks.tsx     # Phase 4
|       |   |   |-- BackupList.tsx          # Phase 4
|       |   |   |-- PodDetail.tsx           # Phase 5.6
|       |   |   |-- TerraformResources.tsx  # Phase 5.7
|       |   |   |-- RestoreDialog.tsx       # Phase 5.8
|       |   |   |-- RedisExplorer.tsx       # Phase 5.9
|       |   |   \-- Banner.tsx              # Phase 5.2
|       |   |-- hooks/                    # Created when needed
|       |   |   |-- useWebSocket.ts       # Phase 3
|       |   |   \-- useApi.ts             # Phase 3
|       |   |-- lib/
|       |   |   \-- utils.ts              # cn() utility
|       |   \-- types/
|       |       \-- index.ts              # Shared TS interfaces
|       |-- index.html
|       |-- package.json
|       |-- tailwind.config.js
|       |-- postcss.config.js
|       |-- tsconfig.json
|       |-- tsconfig.app.json
|       |-- tsconfig.node.json
|       \-- vite.config.ts
|
\-- cli/
    |-- cmd/
    |   \-- ui.go             # k8s-lab ui command
    \-- pkg/
        \-- ui/               # UI backend package
            |-- server.go        # HTTP server, static file serving, signal handler
            |-- handlers.go      # Auth, status, operation handlers
            |-- handlers_data.go # Data handlers (nodes, pods, pvcs, backups, terraform, namespaces)
            |-- tunnel.go        # Persistent tunnel manager with health checks
            \-- websocket.go     # Log streaming via WebSocket
```

Go backend code lives inside `cli/` so it compiles into the same `k8s-lab` binary.

---

## Where APIs are Implemented

All API endpoints are Go HTTP handlers in `cli/pkg/ui/handlers.go`. The routing is set up in `cli/pkg/ui/server.go`. Here's the mapping:

| File | What it does |
|------|-------------|
| `cli/cmd/ui.go` | Cobra command definition, flag parsing, starts server |
| `cli/pkg/ui/server.go` | HTTP router, static file embedding, CORS, signal handler |
| `cli/pkg/ui/handlers.go` | Auth, status, and operation endpoint handlers |
| `cli/pkg/ui/handlers_data.go` | Data endpoint handlers (nodes, pods, pvcs, backups, terraform, namespaces) |
| `cli/pkg/ui/tunnel.go` | `TunnelManager` struct: Start/Stop/HealthCheck/Reconnect |
| `cli/pkg/ui/websocket.go` | WebSocket upgrade, log broadcasting, connection management |

Each handler in `handlers.go` calls existing packages (`cli/pkg/k8s`, `cli/pkg/cloud`, `cli/pkg/terraform`, `cli/pkg/velero`) or shells out to CLI tools (`gcloud`, `kubectl`, `terraform`).

---

## API Endpoints

### Status & Info

| Method | Path | Description | Needs Tunnel |
|--------|------|-------------|:---:|
| GET | `/api/auth` | Cloud auth: user, project, region | No |
| GET | `/api/status` | Overall: infra, k8s, tools, apps, tunnel | Partial |
| GET | `/api/nodes` | K8s node list with roles, zones, IPs | Yes |
| GET | `/api/pods?ns=application` | Pods in namespace (default: application) | Yes |
| GET | `/api/pods/:name?ns=application` | Pod detail (spec, containers) | Yes |
| GET | `/api/pods/:name/logs?ns=application` | Pod logs | Yes |
| GET | `/api/pods/:name/deployment?ns=application` | Owner deployment YAML | Yes |
| GET | `/api/pods/:name/service?ns=application` | Associated service YAML | Yes |
| GET | `/api/pvcs?ns=application` | PVCs with usage info | Yes |
| GET | `/api/backups` | Velero backup list with volume info | Yes |
| GET | `/api/terraform/resources` | Terraform state resource list | No |
| GET | `/api/namespaces` | List all namespaces | Yes |

### Redis Data

| Method | Path | Description | Needs Tunnel |
|--------|------|-------------|:---:|
| GET | `/api/redis/keys?pattern=*&limit=50` | List keys (supports glob pattern) | Yes |
| GET | `/api/redis/get/:key` | Get key value | Yes |
| POST | `/api/redis/set` | Set key-value `{key, value}` | Yes |
| DELETE | `/api/redis/del/:key` | Delete a key | Yes |
| POST | `/api/redis/flush` | Flush entire DB (FLUSHDB) | Yes |

### Operations

| Method | Path | Description | Needs Tunnel |
|--------|------|-------------|:---:|
| POST | `/api/deploy-infra` | Start deploy-infra | No (creates own) |
| POST | `/api/deploy-tools` | Start deploy-tools | Yes |
| POST | `/api/deploy-applications` | Start deploy-applications | Yes |
| POST | `/api/deploy` | Full deploy (all-in-one) | No (starts with infra) |
| POST | `/api/destroy` | Destroy all | No |
| POST | `/api/seed-redis` | Seed Redis test data | Yes |
| POST | `/api/backup` | Create backup | Yes |
| POST | `/api/restore` | Restore `{backup?: name, clean?: bool}` | Yes |
| DELETE | `/api/backups/:name` | Delete a backup | Yes |

### WebSocket

| Path | Description |
|------|-------------|
| `/ws/logs` | Real-time operation output streaming |

### How operations execute:

1. Frontend POSTs to e.g. `/api/deploy-tools`
2. Backend checks: auth OK? tunnel alive? -> returns 4xx if not
3. Backend spawns: `K8SLAB_TUNNEL_MANAGED=true k8s-lab deploy-tools --cloud gcp`
4. stdout/stderr piped to WebSocket as `{type: "log", tab: "deploy-tools", data: "...", timestamp: "..."}`
5. Each operation gets its own log tab in the frontend
6. On completion: `{type: "done", tab: "deploy-tools", exitCode: 0}`
7. Only one operation at a time (mutex). Return 409 if busy.

---

## Implementation Steps

Each phase has verification steps. **Do not move to the next phase until verification passes and user confirms.**

### Phase 1: Foundation

**Step 1.1: CLI tunnel gate**
- File: `cli/pkg/cloud/gcp/provider.go`
- Add `K8SLAB_TUNNEL_MANAGED` env var check to `CreateK8sEndpoint()`
- If set -> return `"localhost:6443"` with no-op cleanup

**Step 1.2: Scaffold React frontend**
- Run `npm create vite@latest frontend -- --template react-ts` inside `ui/`
- Install: `tailwindcss`, `@shadcn/ui`, `lucide-react` (icons)
- Setup dark/light theme with CSS variables matching mockup
- Create `App.tsx` with basic layout (header + grid)

**Step 1.3: Go HTTP server + UI command**
- Create `cli/cmd/ui.go`: Cobra command `k8s-lab ui --cloud <cloud> [--port 3000]`
- Create `cli/pkg/ui/server.go`: embed React `dist/`, serve static + API routes, signal handler
- Auto-open browser on start, CORS for dev mode

**Step 1.4: Auth endpoint**
- File: `cli/pkg/ui/handlers.go`
- `GET /api/auth`: GCP auth check via `gcloud auth list`, project/region from tfvars
- Frontend: Header shows auth badge (`Authenticated` with green dot or `Not Authenticated` with red dot)
- If not authenticated: display red banner, disable all action buttons

> **Phase 1 Verification:**
> 1. `cd cli && go build -o ../bin/k8s-lab .` -- builds without errors
> 2. `cd cli && go test ./...` -- tests pass
> 3. `K8SLAB_TUNNEL_MANAGED=true ./bin/k8s-lab deploy-tools --cloud gcp --verbose` -- logs show "tunnel externally managed, skipping creation" (will fail on actual deploy since no tunnel, but the skip message must appear)
> 4. `cd ui/frontend && npm run build` -- produces `dist/` folder
> 5. `./bin/k8s-lab ui --cloud gcp` -- opens browser, shows header with auth badge and cloud badge
> 6. `curl http://localhost:3000/api/auth` -- returns JSON with account, project, region (or error if not authenticated)
> 7. Ctrl+C stops the server cleanly

### Phase 2: Tunnel Manager + Status

**Step 2.1: Persistent tunnel manager**
- File: `cli/pkg/ui/tunnel.go`
- `TunnelManager` struct: Start/Stop/HealthCheck/IsConnected
- Starts `gcloud compute start-iap-tunnel` subprocess, 10s readiness wait
- Health check every 30s via TCP connect to `localhost:6443`
- Auto-reconnect on failure (max 5 attempts)
- Starts only if terraform state shows infra exists

**Step 2.2: Status endpoint**
- File: `cli/pkg/ui/handlers.go`
- `GET /api/status`: checks infra (terraform state), K8s (tunnel + API), tools (velero ns), apps (pods)
- Returns JSON with per-layer status + tunnel status

**Step 2.3: Cloud info sidebar (frontend)**
- File: `ui/frontend/src/components/CloudInfoPanel.tsx`
- Left sidebar: provider, account, project, region, TF state (clickable if resources exist), backend
- Fetches from `/api/auth` on load

**Step 2.4: Status panel + tunnel badge (frontend)**
- File: `ui/frontend/src/components/StatusPanel.tsx`
- 4 status cards with colored indicators, auto-refresh 30s
- Header tunnel badge: Connected/Reconnecting/Disconnected
- State-aware: grey out panels when tunnel lost, show stale warnings

> **Phase 2 Verification (requires running infrastructure):**
> 1. `./bin/k8s-lab ui --cloud gcp` -- starts with tunnel auto-connecting (if infra exists)
> 2. `curl http://localhost:3000/api/status` -- returns JSON with infra/k8s/tools/apps status
> 3. Browser shows: Cloud Environment sidebar on left with project details
> 4. Browser shows: Status panel with correct state for each layer
> 5. Header shows tunnel status badge (Connected/Idle)
> 6. **Without infra:** tunnel shows "Idle", status shows "Not Created"
> 7. **With infra:** tunnel auto-connects, status shows "Running"

### Phase 3: Operations + Log Streaming

**Step 3.1: WebSocket server**
- File: `cli/pkg/ui/websocket.go`
- WebSocket upgrade at `/ws/logs`, broadcasts operation output to all connected clients
- Message format: `{type: "log"|"done"|"error", tab: "deploy-infra", data: "...", ts: "..."}`

**Step 3.2: Operation executor**
- File: `cli/pkg/ui/handlers.go`
- POST handlers for each operation (deploy-infra, deploy-tools, etc.)
- Pre-flight: 401 if no auth, 503 if tunnel needed but down, 409 if busy
- Spawns CLI with `K8SLAB_TUNNEL_MANAGED=true`, pipes output to WebSocket
- After deploy-infra succeeds: trigger tunnel start

**Step 3.3: Actions panel (frontend)**
- File: `ui/frontend/src/components/ActionsPanel.tsx`
- Step-by-step buttons with state-aware enable/disable
- Spinner during operation, Cancel button

**Step 3.4: Log stream panel (frontend)**
- File: `ui/frontend/src/components/LogStream.tsx`
- WebSocket to `/ws/logs`, log tabs per operation, ANSI color, auto-scroll
- Height: 380px, Clear button

> **Phase 3 Verification (requires running infrastructure):**
> 1. Click "Deploy Infra" in browser -> operation starts, log tab appears with streaming output
> 2. WebSocket messages visible in browser DevTools (Network > WS)
> 3. While deploying: all buttons disabled except Cancel
> 4. After deploy completes: tunnel auto-connects, next buttons enable
> 5. Click "Deploy Tools" -> uses existing tunnel (no 10s delay), log streams in new tab
> 6. Log tabs are switchable (click previous operation tabs to see their output)
> 7. `curl -X POST http://localhost:3000/api/deploy-tools` while busy -> returns 409

### Phase 4: Data Panels

**Step 4.1: Nodes panel**
- Backend: `GET /api/nodes` via `kubectl get nodes -o json`
- Frontend: `NodesPanel.tsx` -- card per node (name, role, zone, type, IP, status)

**Step 4.2: Pod table with namespace dropdown**
- Backend: `GET /api/namespaces`, `GET /api/pods?ns=<namespace>`
- Frontend: `PodTable.tsx` -- namespace dropdown (default: application), clickable rows
- Auto-refresh 15s

**Step 4.3: Pod detail view**
- Backend: `GET /api/pods/:name?ns=...`, `GET /api/pods/:name/logs?ns=...`
- Frontend: `PodDetail.tsx` -- breadcrumb, info grid, volumes/disks section
- **Three tabs**: deployment.yaml, service.yaml, Pod Logs
- Shows PVC mount paths, disk size/usage

**Step 4.4: Persistent disks panel**
- Backend: `GET /api/pvcs?ns=application`
- Frontend: `PersistentDisks.tsx` -- PVC table with usage bar

**Step 4.5: Backup list with volume info**
- Backend: `GET /api/backups`
- Frontend: `BackupList.tsx` -- table with items, volumes, size, Restore/Delete buttons (proper spacing)

**Step 4.6: Restore dialog**
- Frontend: `RestoreDialog.tsx` -- backup picker, clean checkbox, prerequisites checklist

**Step 4.7: Redis data explorer**
- Backend: `GET /api/redis/keys`, `GET /api/redis/get/:key`, `POST /api/redis/set`, `DELETE /api/redis/del/:key`, `POST /api/redis/flush`
- Frontend: `RedisExplorer.tsx` -- SET (key+value), Search (glob), Refresh, DEL per key, Flush DB button
- Executes via `kubectl exec` on redis pod

**Step 4.8: Terraform detail view**
- Backend: `GET /api/terraform/resources` via `terraform show -json`
- Frontend: `TerraformResources.tsx` -- accessed by clicking "N resources" in Cloud Environment sidebar
- Grouped by category (Network, Compute, IAM)

> **Phase 4 Verification (requires fully deployed cluster):**
> 1. Nodes panel shows 3 node cards with correct roles/zones
> 2. Pod table shows pods, namespace dropdown switches between namespaces
> 3. Click a pod -> detail view shows info, volumes section with PVC/size
> 4. Pod detail tabs: deployment.yaml renders correctly, service.yaml renders, Pod Logs stream
> 5. Back button returns to dashboard
> 6. Persistent Disks table shows PVC with usage bar
> 7. Backups table shows backup list with volume names and sizes
> 8. Click Restore -> dialog opens with backup list and prerequisites
> 9. Redis explorer: search for `user:*`, SET a new key, verify it appears, DEL it, verify gone
> 10. Flush DB button works (with confirmation)
> 11. Click "12 resources" in Cloud Environment -> Terraform detail view with grouped resources
> 12. Back button returns to dashboard

### Phase 5: Polish & Missing Features

Everything in `ui/mockup.html` is in scope. Gaps identified by comparing current implementation against the mockup.

**Step 5.1: Complete ActionsPanel (mockup: Idle, Running, Deploying views)**
- Add "Deploy All" button (backend `deploy` op already wired in server.go:78)
- Add "Create Backup" button (backend `backup` op already wired)
- Add "Restore from Backup" button (backend `restore` op already wired)
- Layout: group buttons as mockup shows — "Quick" (Deploy All, Destroy All), "Step by Step" (1-4 with step numbers), "Backup & Restore"
- Step number badges: numbered circles on step-by-step buttons (1, 2, 3, 4). Done state = green checkmark. Active state = pulsing blue. Failed state = red "!".
- Disable logic per mockup: no auth = all disabled; no infra = only Deploy Infra/Deploy All; full deploy = Seed Redis/Backup/Restore/Destroy enabled

**Step 5.2: Banner system (mockup: Not Authenticated, Tunnel Lost, Deploy Failed views)**
- Create `Banner.tsx`: full-width bar below header
- Red banner (not authenticated): "Run `gcloud auth application-default login` and restart the UI."
- Red banner (deploy failed): "deploy-infra failed. Check the operation log for details."
- Yellow banner (tunnel lost): "Tunnel lost. Reconnecting in 5s... Cluster data may be stale."
- App.tsx renders banner conditionally based on auth/status/tunnel/last-operation state

**Step 5.3: Theme toggle (mockup: all views have moon/sun button in header)**
- Dark/light mode with CSS variables matching mockup `.light-theme` class
- Toggle button in header (Moon/Sun icon from lucide-react)
- Detect system preference on first load, persist choice in localStorage

**Step 5.4: Header operation indicator (mockup: Deploying view)**
- Show spinner + operation name in header-left during active operations (e.g., "Deploying Infrastructure...")
- When `isRunning` is true, display next to the logo

**Step 5.5: Panel warnings for degraded states (mockup: Tunnel Lost view)**
- When tunnel disconnected: yellow footer on data panels ("Data may be stale. Last updated X minutes ago.")
- When not authenticated: error footer on Actions panel ("Authentication required to perform any operations")
- When tunnel disconnected: Actions panel footer ("Tunnel disconnected. Only Destroy All works without tunnel.")
- Grey out / reduce opacity on stale data panels
- Disable namespace dropdown on Pods when tunnel is down

**Step 5.6: Pod Detail view (mockup: Pod Detail view)**
- Backend endpoints already exist: `GET /api/pods/:name?ns=...` and `GET /api/pods/:name/logs?ns=...`
- Create `PodDetail.tsx`: breadcrumb ("← Pods / pod-name"), pod info grid (name, namespace, node, pod IP, image, age, restarts, service IP), volumes table
- Tab bar with "Pod Logs" tab showing logs from backend
- Make PodTable rows clickable -> show PodDetail (use React state in App.tsx, no router needed)
- Back button returns to main dashboard

**Step 5.7: Terraform detail view (mockup: Terraform Detail view)**
- Backend endpoint already exists: `GET /api/terraform/resources`
- Create `TerraformResources.tsx`: breadcrumb ("← Dashboard / Terraform Resources"), grouped resource list (Network, Compute, IAM)
- Make CloudInfoPanel "TF State" value clickable when resources exist -> navigates to detail view
- Show resource count ("12 managed resources") in header

**Step 5.8: Restore dialog (mockup: Restore Dialog overlay)**
- Create `RestoreDialog.tsx`: modal overlay with backdrop
- Backup picker: radio button list with backup name, age, volume info
- "Clean restore" checkbox (delete application namespace first)
- Prerequisites checklist: infra deployed, tools installed, tunnel connected (check marks or X)
- Cancel / Restore buttons in footer
- Triggered from ActionsPanel "Restore from Backup" button and from BackupList per-row "Restore" button

**Step 5.9: Redis Data Explorer (mockup: Running view, Tunnel Lost view)**
- Backend (new): Add to `handlers_data.go`:
  - `GET /api/redis/keys?pattern=*&limit=50` — `kubectl exec` redis pod, run `KEYS` or `SCAN`
  - `GET /api/redis/get/:key` — `kubectl exec` redis pod, run `GET`
  - `POST /api/redis/set` — `kubectl exec` redis pod, run `SET` with `{key, value}` body
  - `DELETE /api/redis/del/:key` — `kubectl exec` redis pod, run `DEL`
  - `POST /api/redis/flush` — `kubectl exec` redis pod, run `FLUSHDB`
  - All endpoints use `kubectl exec -n application deploy/redis -- redis-cli <cmd>` with `--kubeconfig`
- Register routes in server.go
- Frontend: Create `RedisExplorer.tsx`:
  - Toolbar: Key input, Value input, SET button | Search input (glob pattern), Search button, Refresh button | Flush DB button
  - Key-Value grid: 3 columns (Key, Value, Del button per row)
  - Show "N keys" count in card header
  - When tunnel disconnected: show "Cannot reach Redis. Waiting for reconnection..."

**Step 5.10: Wire BackupList buttons**
- BackupList restore/delete buttons currently render but have no onClick handlers
- Restore button -> opens RestoreDialog with that backup pre-selected
- Delete button -> `DELETE /api/backups/:name` (needs new backend endpoint in handlers_data.go)
- Add delete backup handler: `velero backup delete <name> --confirm --kubeconfig ...`
- Register `DELETE` method handling on `/api/backups/` route in server.go

**Step 5.11: Documentation**
- `ui/README.md`: architecture, dev setup, build
- Node.js (npm) as build-time only dependency (not needed at runtime)
- One-time setup: `cd ui/frontend && npm install`
- Build: `make build-ui` (runs npm build, copies dist to cli/pkg/ui/dist/, runs go build)
- Explanation of `go:embed` limitation requiring the dist copy step
- Dev mode: Vite dev server (port 5173) with proxy to Go backend (port 8080)

> **Phase 5 Verification:**
> 1. ActionsPanel shows all buttons from mockup: Deploy All, Destroy All, step-by-step (1-4) with step badges, Create Backup, Restore from Backup
> 2. Button enable/disable matches mockup state rules (no auth, no infra, no tools, etc.)
> 3. Banner shows on auth failure / deploy failure / tunnel disconnect with correct colors and messages
> 4. Theme toggle switches dark/light, persists across refresh
> 5. Header shows spinner + operation name during deploys
> 6. Data panels show "stale" warning footer when tunnel lost, reduced opacity
> 7. Click a pod row -> PodDetail opens with breadcrumb, info grid, logs tab. Back returns to dashboard.
> 8. Click "N resources" in Cloud sidebar -> Terraform detail view with grouped resources. Back returns.
> 9. Click "Restore from Backup" -> Restore dialog with backup picker, clean checkbox, prerequisites
> 10. Redis Explorer: search keys, SET new key, DEL key, Flush DB
> 11. BackupList: Restore button opens dialog, Delete button removes backup
> 12. `make build-ui` produces working binary
> 13. `./bin/k8s-lab ui --help` shows usage

### Phase 6: UX Polish (User Testing Feedback)

Feedback from first real end-to-end test. Issues found by comparing live UI with `ui/mockup.html`.

**Already fixed by Claude:**
- Auth badge blue -> green, "Authenticated" text (was showing email in blue)
- Tunnel Idle badge gray instead of red
- Status refresh timing after operations (deploy-tools stayed disabled after deploy-infra)

**Step 6.1: Logo styling**
- Current: "Kubernetes Lab" in plain font with a "K" blue square
- Mockup: `k8s<span style="color:accent">-lab</span>` — "k8s" in white, "-lab" in accent blue, no square icon
- File: `App.tsx` header section (lines 76-81)
- Change to: `<span className="font-bold text-lg">k8s</span><span className="font-bold text-lg text-primary">-lab</span>`

**Step 6.2: Visual warmth — rounded boxes, warmer palette, spacing**
- Mockup uses `border-radius: 10px` on cards, current uses `rounded-lg` (8px). Change to `rounded-xl`.
- Mockup cards have `box-shadow: 0 1px 3px var(--shadow)` — current cards have `shadow-sm` which is fine.
- Status panel items in mockup have colored icon backgrounds (blue for infra, purple for k8s, cyan for tools, green for apps). Current just has a small icon. See mockup CSS: `.status-icon.infra { background: rgba(59,130,246,0.15); }` etc.
- File: `StatusPanel.tsx` — wrap each icon in a colored background square (36x36px, rounded-lg). Each card gets its own accent color:
  - Infrastructure: blue (`bg-blue-500/15 text-blue-500`)
  - Kubernetes: purple (`bg-purple-500/15 text-purple-500`)
  - Tools: cyan (`bg-cyan-500/15 text-cyan-500`)
  - Applications: green (`bg-green-500/15 text-green-500`)
- Status items in mockup are 2x2 grid inside a card with "Cluster Status" header. Current implementation renders 4 separate cards in a row. Change to: single card with "Cluster Status" header, 2x2 grid of status items inside, each item has colored icon box + label + value with status dot.
- Increase `gap-6` between major sections to feel more spacious

**Step 6.3: Cloud provider display with logo**
- Current CloudInfoPanel shows "GCP" plain text for provider
- Change to: "Google Cloud" with a small cloud icon or the Google Cloud SVG
- Since we use lucide-react, use the `Cloud` icon already imported but make it more prominent
- Provider value: `Google Cloud (GCP)` to match mockup
- File: `CloudInfoPanel.tsx` line 13 — change `auth.provider.toUpperCase()` to a display name mapping: `{gcp: 'Google Cloud (GCP)', aws: 'AWS', azure: 'Azure'}[auth.provider] || auth.provider`

**Step 6.4: Actions panel visual improvements**
- Mockup actions are horizontal button rows (`flex-wrap`), current uses vertical stacked full-width buttons
- Step buttons in mockup are compact inline buttons with small step-number circles, not full-width rows
- Change step-by-step section: horizontal row of compact buttons like mockup (`.actions-row { display: flex; gap: 8px; flex-wrap: wrap; }`)
- Each step button: `<button><span class="step-number">1</span> Deploy Infra</button>` — compact, not full-width
- Quick actions (Deploy All, Destroy All): keep side-by-side but make them `flex: 1` within a row
- Backup & Restore: same horizontal row, not stacked
- File: `ActionsPanel.tsx`

**Step 6.5: Log persistence across page refresh (CRITICAL)**
- Current: logs stored in React state (`useState<LogMessage[]>`), lost on refresh and on Ctrl+C server restart
- Problem: user refreshes page mid-deploy, loses all log output
- Solution (backend-side):
  1. In `websocket.go`: add a `logHistory []LogMessage` ring buffer (last 1000 messages) in `WebSocketHub`
  2. When a new WebSocket client connects, send the full history first
  3. When broadcasting, also append to the ring buffer
  4. Add `isRunning bool` to the hub — set on "start" message, clear on "done"/"error"
  5. When client connects, if `isRunning` is true, set their `isRunning` state immediately
- This way:
  - Page refresh: reconnects WebSocket, gets full log history + current running state
  - Ctrl+C: logs are gone (acceptable — server is dead), but...
  - New session after restart: clean slate (acceptable)
- Files: `cli/pkg/ui/websocket.go` (add ring buffer + replay on connect), `useWebSocket.ts` (no change needed, messages are already appended)
- No localStorage needed — backend is the source of truth

**Step 6.6: Tunnel status — when does it turn green?**
- Clarification for the user: Tunnel turns green after deploy-infra completes. The UI auto-starts the tunnel, which takes ~10-15s to connect. During this time it shows "Starting..." (yellow).
- The status refresh improvements (6.5-related, already fixed by Claude) will make status update faster after the tunnel connects.
- No code change needed, just documentation. But verify the tunnel Starting state shows yellow correctly (fixed in App.tsx).

> **Phase 6 Verification:**
> 1. Logo shows "k8s-lab" with accent color styling
> 2. Cards have rounded-xl, Status panel is a single "Cluster Status" card with 2x2 grid and colored icon backgrounds
> 3. CloudInfoPanel shows "Google Cloud (GCP)" instead of "GCP"
> 4. Actions panel has compact horizontal button layout matching mockup
> 5. Page refresh during an operation: reconnect gets full log history, spinner resumes
> 6. After deploy-infra completes, tunnel shows "Starting..." (yellow) then "Connected" (green), and deploy-tools becomes enabled within 15s
> 7. Auth badge shows green "Authenticated", tunnel Idle shows gray

---

## Tech Stack

| Component | Technology | Why |
|-----------|------------|-----|
| Frontend | React 18+ (TypeScript) | Industry standard, best AI assistance |
| Build tool | Vite | Fast, simple |
| Components | shadcn/ui + Tailwind CSS | Full control, dark/light themes |
| Icons | Lucide React | Clean, consistent icon set |
| Backend | Go (in k8s-lab binary) | Single binary, reuses existing packages |
| Embedding | `go:embed` | No separate file serving |
| Real-time | WebSocket | Low-latency log streaming |
| Tunnel | `gcloud` subprocess | Same proven pattern as CLI |

---

## What's NOT in v1

- No auth on the UI itself (localhost only, no login)
- No multi-cloud switching at runtime (pass `--cloud` at startup)
- No resource cost estimates
- No persistent operation history across sessions
- No mobile layout (desktop-only)
- No in-UI editing of terraform.tfvars or manifests

---

## Questions Resolved

| Question | Decision |
|----------|----------|
| Framework? | React + Vite + TypeScript |
| Deploy vs localhost? | Localhost (`k8s-lab ui`) |
| Single binary? | Yes, `go:embed` for React build |
| Cloud selector? | CLI flag: `k8s-lab ui --cloud gcp` |
| Tunnel strategy? | Persistent + `K8SLAB_TUNNEL_MANAGED` env var |
| How to stop UI? | Ctrl+C in terminal, graceful cleanup (kill tunnel, close server) |
| Pre-built frontend? | Embedded at compile time (`make build-ui`). Dev mode uses Vite proxy. |
