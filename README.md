# Kubernetes Lab

A hands-on lab for running self-managed Kubernetes clusters on cloud VMs. Not managed Kubernetes (GKE, EKS, AKS) - you control the full stack from VMs to workloads.

## Goals

- **Learn Talos Linux**: Immutable, API-driven Kubernetes OS
- **Learn Velero**: Backup and restore Kubernetes workloads
- **Reproducibility**: Destroy and recreate clusters from scratch in minutes
- **Multi-cloud**: Same platform layer across different cloud providers

## Current Status

| Phase | Status |
|-------|--------|
| Foundation (Terraform, automation) | Complete |
| Cluster (Talos Linux, Kubernetes) | Complete |
| Backup (Velero) | Complete |
| Applications (Stateful workloads) | In Progress |
| Multi-cloud expansion | Planned |

### Supported Platforms

| Cloud | Compute | Status |
|-------|---------|--------|
| GCP | GCE | Supported |

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Layer 4: Applications                     │
│              (User workloads, PostgreSQL, etc.)              │
├─────────────────────────────────────────────────────────────┤
│                    Layer 3: Platform                         │
│                   (Velero backup/restore)                    │
├─────────────────────────────────────────────────────────────┤
│                    Layer 2: Cluster                          │
│                     (Talos Linux)                            │
├─────────────────────────────────────────────────────────────┤
│                  Layer 1: Infrastructure                     │
│                 (VMs, VPC, Firewall, NAT)                    │
└─────────────────────────────────────────────────────────────┘

Layers 1-2: Cloud-specific
Layers 3-4: Cloud-agnostic (portable across providers)
```

## Prerequisites

**Required tools:** See [infra/README.md](infra/README.md#prerequisites)

**Platform setup:** [GCP setup guide](infra/README.md#gcp-setup) (authentication, permissions, Talos image)

## Quick Start

### All-in-One Deployment

```bash
make deploy gcp   # Infrastructure + tools + applications
make destroy gcp  # Destroy all resources
```

### Step-by-Step Deployment

```bash
# 1. Create infrastructure and bootstrap Kubernetes cluster
make deploy-infra gcp

# 2. Deploy cluster tools (CSI driver, StorageClass, Velero)
make deploy-tools gcp

# 3. Deploy applications (NGINX, Redis)
make deploy-applications gcp

# 4. Connect to cluster for testing
make connect gcp
# Keep this terminal open, then in another terminal:

# 5. Test the deployment
export KUBECONFIG=configs/talos/kubeconfig

# Check pods are running
kubectl get pods -n application

# Test Redis
kubectl exec -it deploy/redis -n application -- redis-cli ping
# Should return: PONG

# Test NGINX
kubectl port-forward svc/nginx -n application 8080:80 &
curl http://localhost:8080
# Should return: NGINX welcome page

# 6. Clean up
make destroy gcp
```

### Backup and Restore Workflow

```bash
# Day 1: Deploy, seed data, backup
make deploy-infra gcp
make deploy-tools gcp
make deploy-applications gcp
make seed-redis gcp
make backup gcp               # Redis BGSAVE hooks via pod annotations
make destroy gcp

# Day 2+: Restore from backup
make deploy-infra gcp
make deploy-tools gcp
make restore gcp              # Restores apps from latest backup

# Verify restored data
export KUBECONFIG=configs/talos/kubeconfig
kubectl exec -it deploy/redis -n application -- redis-cli GET user:1
# Should return seeded data

# List backups
kubectl get backup -n velero

# Delete all backups (cleanup)
make delete-all-backups gcp
```

**Backup Hooks:** Redis uses pod annotations for backup hooks (apps/redis.yaml:40-45). For centralized hooks with label selectors, see configs/velero/backup-hooks.yaml.example (Redis + PostgreSQL examples).

### Using Centralized Hooks (Optional)

To use centralized hooks instead of pod annotations:

```bash
# 1. Create hooks configuration
cp configs/velero/backup-hooks.yaml.example configs/velero/backup-hooks.yaml
cp configs/velero/restore-hooks.yaml.example configs/velero/restore-hooks.yaml

# 2. Edit configs/velero/backup-hooks.yaml and uncomment desired hooks

# 3. Set environment variables and run backup
export VELERO_BACKUP_HOOKS_FILE="${PWD}/configs/velero/backup-hooks.yaml"
export VELERO_RESTORE_HOOKS_FILE="${PWD}/configs/velero/restore-hooks.yaml"
make backup gcp
make restore gcp
```

Centralized hooks use label selectors to target multiple pods and can be changed without redeploying applications. Pod annotations take precedence when both are defined.

## Repository Structure

```
k8s-lab/
├── apps/                     # Kubernetes manifests
│   ├── gcp/                  # GCP-specific (StorageClass)
│   ├── nginx.yaml            # NGINX deployment
│   └── redis.yaml            # Redis deployment with PVC and backup hooks
├── infra/                    # Cloud infrastructure
│   └── gcp/
│       ├── talos-patches/    # Talos machine config patches
│       └── terraform/        # Terraform definitions
├── scripts/                  # Automation (organized by architecture layer)
│   ├── infra/                # Layer 1-2: Infrastructure & Kubernetes
│   │   ├── deploy.sh         # Create infrastructure and bootstrap cluster
│   │   ├── destroy.sh        # Tear down all resources
│   │   └── connect.sh        # Interactive tunnel for kubectl
│   ├── platform/             # Layer 3: Platform services
│   │   └── deploy.sh         # Deploy CSI driver, StorageClass, Velero
│   ├── workloads/            # Layer 4: Application workloads
│   │   ├── deploy.sh         # Deploy NGINX and Redis
│   │   └── seed-redis.sh     # Seed Redis with test data
│   ├── backup/               # Backup operations (uses platform layer)
│   │   ├── create.sh         # Create backup
│   │   ├── restore.sh        # Restore from backup
│   │   ├── list.sh           # List all backups
│   │   ├── delete.sh         # Delete a backup by name
│   │   └── delete-all.sh     # Delete all backups
│   └── lib/                  # Shared functions (cloud-agnostic + cloud-specific)
│       ├── common.sh         # Logging, prerequisites, utilities
│       ├── workloads.sh      # Workload helpers (cloud-agnostic)
│       ├── velero.sh         # Velero backup/restore (cloud-agnostic)
│       ├── talos.sh          # Talos config and bootstrap (cloud-agnostic)
│       └── gcp/              # GCP-specific implementations
│           ├── infra.sh      # Terraform operations
│           ├── tunnel.sh     # IAP tunnel management
│           ├── csi.sh        # GCE PD CSI driver
│           ├── velero.sh     # GCP Velero plugin installation
│           └── verify.sh     # Resource verification
├── configs/                  # Configuration files
│   ├── velero/               # Velero centralized hooks (opt-in)
│   │   ├── backup-hooks.yaml.example # Redis, PostgreSQL examples
│   │   └── restore-hooks.yaml.example# Redis, PostgreSQL examples
│   └── talos/                # Generated Talos/K8s configs (gitignored)
│       ├── controlplane.yaml
│       ├── worker.yaml
│       ├── talosconfig
│       └── kubeconfig
├── Makefile                  # Entry points
└── CLAUDE.md                 # Project policies and roadmap
```

## Tech Stack

| Component | Technology | Purpose |
|-----------|------------|---------|
| Infrastructure | Terraform | VMs, networking, firewall |
| Kubernetes OS | Talos Linux | Immutable, API-driven nodes |
| Backup | Velero | Cluster backup/restore to GCS |
| Cluster | 1 CP + 2 Workers | Multi-AZ |

## Lessons Learned

| Lesson | Tool | Platform | Workaround |
|--------|------|----------|------------|
| **No SSH access** - All management via Talos API (port 50000). Combined with IAP tunneling, scripts must manage tunnel lifecycle. Orphaned tunnels from failed scripts need manual cleanup (`pkill -f "gcloud.*start-iap-tunnel"`). | Talos | GCP | Sequential tunnel per operation |
| **Filesystem layout differs** - CSI driver expects `/lib/udev`, `/etc/udev`. Talos uses `/usr/lib/udev`, `/etc/udev` doesn't exist. | Talos | GCP | kubelet `extraMounts` patch ([Talos-recommended](https://github.com/siderolabs/talos/issues/4143)) |
| **CSI driver needs cloud credentials** - GCE PD CSI runs as pods, needs API access to create disks. Talos VMs don't inherit credentials like GKE. | Talos | GCP | Service account with `compute.storageAdmin` attached to VMs |
| **API port fixed at 50000** - `talosctl apply-config` only works on port 50000. Different local tunnel port fails silently. | Talos | Any | Always tunnel to localhost:50000, configure nodes sequentially |
| **Bootstrap is one-time** - `talosctl bootstrap` initializes etcd, can only run once. Running again corrupts cluster. | Talos | Any | Failed mid-way? Full destroy and recreate |
| **IAP tunnel needs time** - Starting tunnel and immediately using it fails. Connection not ready yet. | gcloud | GCP | Sleep 10s after start, retry logic (5 attempts) |
| **Velero CLI rate limiter errors** - `velero backup describe/logs` shows "rate limiter Wait returned an error" when querying backup details. Root cause: Velero CLI binary has hardcoded client-go rate limiter (default 5 QPS / 10 burst). High-latency connections like IAP tunnels cause operations to take longer, exhausting burst tokens and exceeding context deadline. Backup itself succeeds, only CLI display commands fail. | Velero | Any | Use kubectl instead: `kubectl get backup -n velero <name> -o yaml` or `kubectl get podvolumebackups -n velero -l velero.io/backup-name=<name>`. Server-side `--client-qps/burst` flags don't fix CLI issues (see [#7991](https://github.com/vmware-tanzu/velero/issues/7991)) |
| **Velero Go SDK doesn't support installation** - The `github.com/vmware-tanzu/velero/pkg/client` package only provides client factories for accessing Velero resources, not installation. For programmatic installation, either shell out to `velero install` CLI command (like terraform-exec pattern) or use Helm SDK. Generated clientsets for Velero CRDs don't exist in the public module. | Velero | Any | Hybrid approach: Use velero CLI for installation (`exec.Command`), client-go dynamic client with Velero API types for backup/restore operations. Follow terraform/gcloud precedent of shelling out where native SDK gaps exist. |

## Documentation

| Document | Description |
|----------|-------------|
| [CLAUDE.md](CLAUDE.md) | Project policies, roadmap, architecture decisions |
| [apps/README.md](apps/README.md) | Application manifests and deployment |
| [infra/README.md](infra/README.md) | Infrastructure overview and platform links |
| [scripts/README.md](scripts/README.md) | Script details and Talos debugging guide |
