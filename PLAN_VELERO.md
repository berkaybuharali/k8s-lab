# Velero Backup/Restore Plan (Phase 3)

## Goal

After `make deploy gcp` + `make apply gcp` + `make seed-redis`, run `make backup gcp` to save state to GCS. Next day, after `make deploy gcp`, run `make restore gcp` to get Redis data and NGINX back without re-applying or re-seeding.

## Architecture

```
Existing GCS Bucket (terraform state bucket)
  ├── terraform/          (existing state)
  └── velero/             (backups - new)
       └── backups/
            └── <backup-name>/  (namespace manifests + PV snapshots)
```

Velero uses two mechanisms:
1. **Resource backup** - K8s manifests (namespaces, deployments, services, PVCs) stored as JSON in GCS
2. **Volume snapshots** - GCE PD snapshots for Redis PVC (GCP disk snapshots, referenced from GCS)

## Script Design: Vendor-Agnostic Interfaces

Key principle: Velero logic is cloud-agnostic. Only the **install configuration** (plugin, bucket, credentials) is cloud-specific. Scripts are split accordingly:

```
scripts/lib/
  velero.sh              # Cloud-agnostic: velero_backup(), velero_restore(), velero_wait_ready()
  gcp/
    velero.sh            # GCP-specific: gcp_velero_install() (plugin, bucket, SA config)
```

`backup.sh` and `restore.sh` call cloud-agnostic functions. Each cloud provider implements its own `<cloud>_velero_install()` function that handles provider-specific Velero configuration.

## What restore.sh Must Do (CSI + StorageClass)

Restore needs CSI driver and StorageClass **before** Velero can bind restored PVCs. This means `restore.sh` must install CSI + StorageClass as a prerequisite, then install Velero, then restore. This is unavoidable - Velero cannot restore PVCs without a working storage provisioner.

This means `make apply` and `make restore` both install CSI+StorageClass, but for different purposes:
- `make apply` = CSI + StorageClass + Velero + deploy apps fresh
- `make restore` = CSI + StorageClass + Velero + restore apps from backup

## Changes

### 1. Terraform: IAM Only (No New Bucket)

**File: `infra/gcp/terraform/gce.tf`**
- Add `roles/storage.objectAdmin` to node service account (Velero writes to GCS)
- `roles/compute.storageAdmin` already exists (disk snapshots)

No new bucket or variables needed - reuse existing terraform state bucket with `velero/` prefix.

### 2. Cloud-Agnostic Velero Library

**File: `scripts/lib/velero.sh`** (new)
- `velero_wait_ready()` - Wait for Velero deployment to be running
- `velero_backup()` - `velero backup create <name> --include-namespaces application --wait`
- `velero_get_latest_backup()` - Find most recent successful backup
- `velero_restore()` - `velero restore create --from-backup <name> --wait`
- `velero_verify_backup()` - Check backup status is "Completed"
- `velero_verify_restore()` - Check restore status, verify pods running, verify Redis data

### 3. GCP-Specific Velero Install

**File: `scripts/lib/gcp/velero.sh`** (new)
- `gcp_velero_install()` - Install Velero with GCP plugin
  - `velero install` with:
    - `--provider gcp`
    - `--plugins velero/velero-plugin-for-gcp:v1.11.0`
    - `--bucket <state-bucket>` (same as terraform)
    - `--prefix velero` (separate path within bucket)
    - `--no-secret` (use VM service account, no JSON key)
    - `--backup-location-config serviceAccount=<sa-email>`
    - `--snapshot-location-config project=<project-id>`
  - Calls `velero_wait_ready()`

### 4. Backup and Restore Scripts

**File: `scripts/backup.sh`** (new)
- Sources: common.sh, gcp/infra.sh, gcp/tunnel.sh, velero.sh
- Opens IAP tunnel
- Calls `velero_backup()`
- Verifies backup completed
- Closes tunnel

**File: `scripts/restore.sh`** (new)
- Sources: common.sh, gcp/infra.sh, gcp/tunnel.sh, gcp/csi.sh, velero.sh, gcp/velero.sh
- Opens IAP tunnel
- Installs CSI driver + StorageClass (prerequisite for PVC restore)
- Calls `gcp_velero_install()` (cloud-specific)
- Calls `velero_restore()` (cloud-agnostic)
- Verifies: namespace, NGINX, Redis, Redis data
- Closes tunnel

### 5. Makefile Targets

```makefile
backup:    scripts/backup.sh $(cloud)
restore:   scripts/restore.sh $(cloud)
```

### 6. Integration with apply.sh

**File: `scripts/apply.sh`**
- Add Velero installation after CSI driver install (so `make backup` works without extra setup)
- Sources gcp/velero.sh, calls `gcp_velero_install()`

### 7. Prerequisites

**File: `scripts/lib/common.sh`**
- Add `velero` to brew prerequisites check

## Daily Workflow

**Day 1 (first time):**
```
make deploy gcp     # infra + k8s
make apply gcp      # CSI + StorageClass + Velero + apps
make seed-redis gcp # test data
make backup gcp     # save to GCS
make destroy gcp    # tear down
```

**Day 2+ (restore flow):**
```
make deploy gcp     # infra + k8s
make restore gcp    # CSI + StorageClass + Velero + restore from backup
# Apps + Redis data restored automatically
make destroy gcp
```

## Files to Create/Modify

| File | Action |
|------|--------|
| `infra/gcp/terraform/gce.tf` | Modify - add storage.objectAdmin IAM role |
| `scripts/lib/velero.sh` | Create - cloud-agnostic Velero functions |
| `scripts/lib/gcp/velero.sh` | Create - GCP-specific Velero install |
| `scripts/backup.sh` | Create - backup orchestration |
| `scripts/restore.sh` | Create - restore orchestration |
| `scripts/apply.sh` | Modify - add Velero install step |
| `scripts/lib/common.sh` | Modify - add velero to prerequisites |
| `Makefile` | Modify - add backup/restore targets |
| `README.md` | Update - Phase 3 status, new commands |
| `CLAUDE.md` | Update - Phase 3 status |

## Verification

1. `make deploy gcp && make apply gcp && make seed-redis gcp`
2. `make backup gcp` - verify backup shows "Completed"
3. `make destroy gcp`
4. `make deploy gcp && make restore gcp`
5. Verify: `kubectl get pods -n application` shows NGINX + Redis running
6. Verify: `kubectl exec` into Redis, check `GET user:1` returns seeded data
