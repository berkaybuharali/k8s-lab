# k8s-lab Web UI

A local web dashboard for managing the Kubernetes lab environment. Built with React (TypeScript) frontend and Go HTTP backend, served as a single binary.

## Architecture

```
Browser <──> Go HTTP Server (port 3000)
               ├── REST API ──> kubectl / velero / terraform CLI
               ├── WebSocket ──> Real-time log streaming
               └── Embedded SPA ──> React (go:embed)
                     │
                     └── IAP Tunnel ──> Kubernetes API (auto-managed)
```

- **Frontend**: React 18, Vite, Tailwind CSS, Lucide icons
- **Backend**: Go HTTP server (`cli/pkg/ui/`) embedded in the `k8s-lab` binary via `go:embed`
- **Communication**: REST for data/actions, WebSocket for live log streaming
- **Tunnel**: Persistent IAP tunnel to Kubernetes API, auto-managed with health checks and reconnection

## Project Structure

```
ui/
└── frontend/               # React application
    ├── src/
    │   ├── App.tsx          # Root layout, view routing, state management
    │   ├── components/
    │   │   ├── ActionsPanel.tsx      # Deploy/destroy/backup/restore buttons
    │   │   ├── BackupList.tsx        # Velero backup list with delete/restore
    │   │   ├── Banner.tsx            # Auth/tunnel warning banners
    │   │   ├── CloudInfoPanel.tsx    # Cloud provider info sidebar
    │   │   ├── LogStream.tsx         # Real-time operation log viewer
    │   │   ├── NodesPanel.tsx        # Kubernetes node status
    │   │   ├── PersistentDisks.tsx   # PVC list
    │   │   ├── PodDetail.tsx         # Single pod info + logs (drill-down view)
    │   │   ├── PodTable.tsx          # Pod list with click-to-detail
    │   │   ├── RedisExplorer.tsx     # Redis key browser with GET/SET/DEL
    │   │   ├── RestoreDialog.tsx     # Backup restore modal
    │   │   ├── StatusPanel.tsx       # Infra/K8s/Tools/Apps status cards
    │   │   ├── TerraformResources.tsx# Terraform state viewer (grouped)
    │   │   └── ThemeToggle.tsx       # Dark/light mode
    │   ├── hooks/
    │   │   ├── useWebSocket.ts       # WebSocket connection + log state
    │   │   └── useApi.ts             # Operation trigger hook
    │   └── types/index.ts            # Shared TypeScript interfaces

cli/pkg/ui/                  # Go backend
    ├── server.go            # HTTP server, route registration, go:embed
    ├── handlers.go          # Auth, status, operation execution
    ├── handlers_data.go     # kubectl/velero/redis data endpoints
    ├── tunnel.go            # IAP tunnel lifecycle + health checks
    └── websocket.go         # WebSocket hub for log broadcasting
```

## Prerequisites

- Go 1.21+
- Node.js 18+ and npm 9+ (build-time only)
- Standard lab tools: `gcloud`, `kubectl`, `terraform`, `talosctl`, `velero`

## Running

```bash
# Production: build and run single binary
make build-ui
./bin/k8s-lab ui --cloud gcp

# Custom port
./bin/k8s-lab ui --cloud gcp --port 8080
```

The browser opens automatically at `http://localhost:3000`.

## Development

Run backend and frontend separately for hot-reloading:

```bash
# Terminal 1: Go backend (port 3000)
cd cli && go run main.go ui --cloud gcp

# Terminal 2: React dev server (port 5173, proxies to 3000)
cd ui/frontend && npm install && npm run dev
```

Open `http://localhost:5173`. Vite proxies `/api` and `/ws` requests to the Go backend.

## Build

```bash
make build-ui
```

This runs: `npm build` -> copy `dist/` to `cli/pkg/ui/dist/` -> `go build`. The copy step is required because `go:embed` only embeds files relative to the package directory.

## API Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/auth` | Cloud authentication status |
| GET | `/api/status` | Infrastructure/cluster/tunnel status |
| POST | `/api/{operation}` | Trigger operation (deploy-infra, destroy, backup, etc.) |
| WS | `/ws/logs` | Real-time operation log stream |
| GET | `/api/nodes` | Kubernetes nodes |
| GET | `/api/pods?ns=` | Pods in namespace |
| GET | `/api/pods/{name}?ns=` | Single pod detail |
| GET | `/api/pods/{name}/logs?ns=` | Pod logs (last 100 lines) |
| GET | `/api/pvcs?ns=` | Persistent volume claims |
| GET | `/api/backups` | Velero backups |
| DELETE | `/api/backups/{name}` | Delete a backup |
| GET | `/api/namespaces` | Kubernetes namespaces |
| GET | `/api/terraform/resources` | Terraform state |
| GET | `/api/redis/keys?pattern=` | Redis KEYS |
| GET | `/api/redis/get/{key}` | Redis GET |
| POST | `/api/redis/set` | Redis SET (JSON body: `{key, value}`) |
| DELETE | `/api/redis/del/{key}` | Redis DEL |
| POST | `/api/redis/flush` | Redis FLUSHDB |
