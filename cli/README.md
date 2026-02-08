# Go CLI

A standalone Go binary for managing Kubernetes lab environments. Provides the same functionality as the Makefile interface with a modern CLI experience.

## Building

From the repository root:

```bash
cd cli
go build -o ../bin/k8s-lab .
```

Binary location: `bin/k8s-lab`

For installation instructions to run without the `./bin/` prefix, see the [root README Quick Start](../README.md#quick-start).

## Architecture

### Framework

Built with [Cobra](https://github.com/spf13/cobra) for CLI structure:
- `cmd/`: Command implementations (deploy-infra, deploy-tools, backup, restore, etc.)
- `pkg/`: Reusable packages

### Package Structure

```
cli/
├── cmd/                     # Cobra commands
│   ├── root.go             # Root command + global flags (--cloud, --verbose)
│   ├── deploy_infra.go     # Infrastructure deployment
│   ├── deploy_tools.go     # Platform tools (CSI, Velero)
│   ├── deploy_applications.go # Application deployment
│   ├── seed_redis.go       # Redis data seeding
│   ├── backup.go           # Velero backup
│   ├── restore.go          # Velero restore
│   └── ui.go               # Web dashboard server
├── pkg/
│   ├── cloud/              # Cloud provider abstraction
│   │   ├── interface.go    # Provider interface
│   │   └── gcp/           # GCP implementation (IAP tunnels, CSI, Velero config)
│   ├── config/            # Configuration (paths, repo detection)
│   ├── k8s/               # Kubernetes operations (client-go wrappers)
│   ├── logger/            # Colored output
│   ├── prerequisites/     # Tool checking (kubectl, terraform, etc.)
│   ├── terraform/         # Terraform operations (init, apply, output)
│   ├── ui/                # Web dashboard backend
│   │   ├── server.go      # HTTP server, embedded SPA, route registration
│   │   ├── handlers.go    # Auth, status, operation execution
│   │   ├── handlers_data.go # kubectl/velero/redis data endpoints
│   │   ├── tunnel.go      # IAP tunnel lifecycle + health checks
│   │   ├── websocket.go   # WebSocket hub for log broadcasting
│   │   └── dist/          # Embedded React build (copied from ui/frontend/dist)
│   └── velero/            # Velero operations (install, backup, restore)
└── main.go                # Entry point
```

### Design Patterns

**When we shell out to external CLIs:**
- **Terraform:** Uses Go SDK when available, shells out for operations without SDK support
- **Velero install:** Velero Go SDK doesn't support installation, so we use `exec.Command("velero", "install", ...)`
- **kubectl apply -k:** Kustomize integration via kubectl

This follows the same pattern as terraform-exec and other infrastructure tools.

**When we use native Go:**
- **Backup/Restore CRs:** Uses client-go dynamic client with Velero API types (pure Go, testable)
- **Namespace management:** client-go Kubernetes clientset
- **Deployment readiness checks:** client-go watch API

**Why hybrid approach?**
- Tool installation (velero install, kubectl apply -k) lacks Go SDK support
- Runtime operations (backup, restore) benefit from pure Go (testability, error handling)
- Infrastructure operations (terraform) use official SDKs where available

## Commands

| Command | Description | Equivalent Makefile |
|---------|-------------|-------------------|
| `deploy-infra` | Deploy infrastructure and bootstrap cluster | `make deploy-infra gcp` |
| `deploy-tools` | Install CSI driver, StorageClass, Velero | `make deploy-tools gcp` |
| `deploy-applications` | Deploy NGINX and Redis | `make deploy-applications gcp` |
| `seed-redis` | Populate Redis with test data | `make seed-redis gcp` |
| `backup` | Create Velero backup | `make backup gcp` |
| `restore` | Restore from Velero backup | `make restore gcp` |
| `ui` | Start web dashboard | N/A (CLI only) |

All commands require `--cloud <provider>` flag (currently only `gcp` supported). The `ui` command also accepts `--port` (default: 3000). See [root README Quick Start](../README.md#quick-start-go-cli) for usage examples.

## Cloud Provider Interface

The `cloud.Provider` interface abstracts cloud-specific operations:

```go
type Provider interface {
    Name() string
    Validate(ctx context.Context) error
    GetProjectID(terraformDir string) (string, error)
    EnsureStateBucket(ctx context.Context, terraformDir string) error
    CreateTalosEndpoint(ctx context.Context, instanceName, zone, projectID string) (string, func(), error)
    CreateK8sEndpoint(ctx context.Context, instanceName, zone, projectID string) (string, func(), error)
    InstallCSIDriver(ctx context.Context, kubeconfigPath string) error
    GetVeleroInstallConfig(terraformDir string) (interface{}, error)
}
```

Current implementations:
- **GCP:** IAP tunneling, GCE PD CSI driver, Velero with GCS backend

Future providers (STACKIT, AWS, Azure) will implement this interface.

## Error Handling

- **Fail fast:** Prerequisites checked before operations
- **Context propagation:** All operations support context cancellation
- **Cleanup:** Deferred cleanup functions for tunnels and resources
- **Descriptive errors:** Wrapped errors with context (`fmt.Errorf("failed to X: %w", err)`)

## Configuration

Configuration is auto-detected from repository structure:
- **Repo root:** Detected via .git directory
- **Kubeconfig:** `configs/talos/kubeconfig`
- **Terraform dir:** `infra/<cloud>/terraform`

Override via flags where needed (`--cloud` is required for most commands).
