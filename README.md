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
export KUBECONFIG=configs/kubeconfig

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
make backup gcp
make destroy gcp

# Day 2+: Restore from backup
make deploy-infra gcp
make restore gcp              # Installs tools + restores apps

# Verify backup and restore
export KUBECONFIG=configs/kubeconfig

# List backups
kubectl get backup -n velero

# Check backup details (use kubectl to avoid rate limiter errors)
kubectl get backup -n velero <backup-name> -o json | jq .status

# Verify restored data
kubectl exec -it deploy/redis -n application -- redis-cli GET user:1
# Should return seeded data
```

## Setup

### 1. Install Prerequisites

See [infra/README.md](infra/README.md#prerequisites) for required tools.

### 2. Platform Setup

Follow the setup guide for your cloud provider:

| Platform | Setup Guide |
|----------|-------------|
| GCP | [infra/README.md#gcp-setup](infra/README.md#gcp-setup) |

This includes: authentication, permissions, Talos image upload, and configuration.

### 3. Create Cluster and Deploy Applications

Run `make deploy gcp` followed by `make apply gcp`. See [scripts/README.md](scripts/README.md) for details.

## Repository Structure

```
k8s-lab/
├── apps/                     # Kubernetes manifests
│   ├── gcp/                  # GCP-specific (StorageClass)
│   ├── nginx.yaml            # NGINX deployment
│   └── redis.yaml            # Redis deployment with PVC
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
│       ├── common.sh         # Logging, prerequisites
│       ├── workloads.sh      # Workload helpers (cloud-agnostic)
│       ├── velero.sh         # Velero backup/restore (cloud-agnostic)
│       ├── talos.sh          # Talos operations (cloud-agnostic)
│       └── gcp/              # GCP-specific implementations
├── configs/                  # Generated configs (gitignored)
├── Makefile                  # Entry points
└── CLAUDE.md                 # Project policies and roadmap
```

## Tech Stack

| Component | Technology | Purpose |
|-----------|------------|---------|
| Infrastructure | Terraform | VMs, networking, firewall |
| Kubernetes OS | Talos Linux | Immutable, API-driven nodes |
| Backup | Velero | Cluster backup and restore to GCS |
| Cluster | 1 CP + 2 Workers | Spread across availability zones |

## Why Talos Linux?

Traditional Kubernetes nodes run a full Linux distribution with SSH, package managers, and manual configuration. Talos takes a different approach:

- **No SSH**: All management through a secure API (port 50000)
- **Immutable**: Read-only filesystem, no runtime modifications
- **Declarative**: Entire node configuration in a single YAML file
- **Minimal**: Purpose-built for Kubernetes, nothing else

This makes clusters reproducible - the same configuration always produces the same result.

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

## Documentation

| Document | Description |
|----------|-------------|
| [CLAUDE.md](CLAUDE.md) | Project policies, roadmap, architecture decisions |
| [apps/README.md](apps/README.md) | Application manifests and deployment |
| [infra/README.md](infra/README.md) | Infrastructure overview and platform links |
| [scripts/README.md](scripts/README.md) | Script details and Talos debugging guide |
