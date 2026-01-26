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
| Backup (Velero) | Planned |
| Applications (Stateful workloads) | Planned |
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

```bash
make deploy  # Create infrastructure and bootstrap cluster
make down    # Destroy all resources
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

### 3. Create Cluster

```bash
make deploy
```

## Repository Structure

```
k8s-lab/
├── infra/                    # Cloud infrastructure (Terraform)
│   └── gcp/                  # GCP-specific
│       └── terraform/
├── scripts/                  # Automation
│   ├── setup.sh              # Cluster creation
│   ├── destroy.sh            # Cluster teardown
│   └── lib/                  # Shared functions
├── k8s/                      # Kubernetes manifests (future)
├── configs/                  # Generated configs (gitignored)
├── Makefile                  # Entry points
└── CLAUDE.md                 # Project policies and roadmap
```

## Tech Stack

| Component | Technology | Purpose |
|-----------|------------|---------|
| Infrastructure | Terraform | VMs, networking, firewall |
| Kubernetes OS | Talos Linux | Immutable, API-driven nodes |
| Backup | Velero (planned) | Cluster backup and restore |
| Cluster | 1 CP + 2 Workers | Spread across availability zones |

## Why Talos Linux?

Traditional Kubernetes nodes run a full Linux distribution with SSH, package managers, and manual configuration. Talos takes a different approach:

- **No SSH**: All management through a secure API (port 50000)
- **Immutable**: Read-only filesystem, no runtime modifications
- **Declarative**: Entire node configuration in a single YAML file
- **Minimal**: Purpose-built for Kubernetes, nothing else

This makes clusters reproducible - the same configuration always produces the same result.

## Documentation

| Document | Description |
|----------|-------------|
| [CLAUDE.md](CLAUDE.md) | Project policies, roadmap, architecture decisions |
| [infra/README.md](infra/README.md) | Infrastructure overview and platform links |
| [scripts/README.md](scripts/README.md) | Script details and Talos debugging guide |
