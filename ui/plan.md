# UI Implementation Plan

## Overview

A local web dashboard for k8s-lab that provides a visual interface to all cluster lifecycle operations. Runs as `k8s-lab ui --cloud gcp` - single binary serving React frontend + Go API backend.

**Mockup:** Open `ui/mockup.html` in a browser to see all 8 interactive views.

---

## Current Status

**Phase:** Phase 3: Operations + Log Streaming
**Branch:** `feature/ui`
**Next Step:** Phase 3, Step 3.1 (WebSocket server)

| Phase | Status |
|-------|--------|
| Phase 1: Foundation | Complete |
| Phase 2: Tunnel Manager + Status | Complete |
| Phase 3: Operations + Log Streaming | Not started |
| Phase 4: Data Panels | Not started |
| Phase 5: Polish | Not started |

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
|       |   |-- components/
|       |   |   |-- Header.tsx
|       |   |   |-- CloudInfoPanel.tsx
|       |   |   |-- StatusPanel.tsx
|       |   |   |-- ActionsPanel.tsx
|       |   |   |-- NodesPanel.tsx
|       |   |   |-- PodTable.tsx
|       |   |   |-- PodDetail.tsx
|       |   |   |-- PersistentDisks.tsx
|       |   |   |-- BackupList.tsx
|       |   |   |-- RedisExplorer.tsx
|       |   |   |-- TerraformResources.tsx
|       |   |   |-- LogStream.tsx
|       |   |   |-- RestoreDialog.tsx
|       |   |   |-- Banner.tsx
|       |   |   \-- ThemeToggle.tsx
|       |   |-- hooks/
|       |   |   |-- useWebSocket.ts
|       |   |   \-- useApi.ts
|       |   |-- types/
|       |   |   \-- index.ts
|       |   \-- theme/
|       |       \-- index.ts
|       |-- index.html
|       |-- package.json
|       |-- tsconfig.json
|       \-- vite.config.ts
|
\-- cli/
    |-- cmd/
    |   \-- ui.go             # NEW: k8s-lab ui command
    \-- pkg/
        \-- ui/               # NEW: UI backend package
            |-- server.go     # HTTP server, static file serving, signal handler
            |-- handlers.go   # API endpoint handlers
            |-- tunnel.go     # Persistent tunnel manager with health checks
            \-- websocket.go  # Log streaming via WebSocket
```

Go backend code lives inside `cli/` so it compiles into the same `k8s-lab` binary.

---

## Where APIs are Implemented

All API endpoints are Go HTTP handlers in `cli/pkg/ui/handlers.go`. The routing is set up in `cli/pkg/ui/server.go`. Here's the mapping:

| File | What it does |
|------|-------------|
| `cli/cmd/ui.go` | Cobra command definition, flag parsing, starts server |
| `cli/pkg/ui/server.go` | HTTP router, static file embedding, CORS, signal handler |
| `cli/pkg/ui/handlers.go` | All `/api/*` endpoint handlers (auth, status, pods, redis, operations) |
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

### Phase 5: Polish

**Step 5.1: Theme toggle**
- Dark/light with system preference detection, persist in localStorage

**Step 5.2: Banner system**
- `Banner.tsx`: red (not auth, deploy failed), yellow (tunnel lost)

**Step 5.3: Error handling**
- API errors -> toast, operation failures -> banner + log, tunnel disconnect -> grey out panels

**Step 5.4: Build integration**
- Makefile target: `make build-ui` (npm build + go build)
- Dev mode: `--dev` flag proxies to Vite dev server

**Step 5.5: Documentation**
- `ui/README.md`: architecture, dev setup, build
- Node.js (npm) as build-time only dependency
- Fully self-contained binary explanation (go:embed)
- One-time setup (`npm install`) and build (`make build-ui`) instructions
- Tooling list update (node/npm)
- Explanation of the `dist/` copy step for `go:embed` requirements

> **Phase 5 Verification:**
> 1. Theme toggle works, persists across page refresh
> 2. Banner shows on auth failure / deploy failure / tunnel disconnect
> 3. Panels grey out when tunnel lost, show "stale" warning
> 4. `make build-ui` produces working binary
> 5. `./bin/k8s-lab ui --help` shows usage

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
