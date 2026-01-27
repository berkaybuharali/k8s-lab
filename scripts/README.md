# Scripts

Automation scripts for cluster lifecycle management.

For prerequisites, see [infra/README.md](../infra/README.md#prerequisites).

## Structure

```
scripts/
├── cluster/
│   ├── deploy.sh             # Create infrastructure and bootstrap cluster
│   ├── destroy.sh            # Destroy cluster and applications
│   └── connect.sh            # Start interactive tunnel for manual kubectl access
├── apps/
│   ├── apply.sh              # Deploy applications + Velero
│   └── seed-redis.sh         # Seed Redis with test data for Velero testing
├── velero/
│   ├── backup.sh             # Backup applications to cloud storage
│   ├── restore.sh            # Restore applications from backup
│   ├── list-backups.sh       # List all backups
│   ├── delete-backup.sh      # Delete a backup by name
│   └── delete-all-backups.sh # Delete all backups
├── lib/
│   ├── common.sh             # Logging, prerequisites, utilities
│   ├── apps.sh               # Application deployment (cloud-agnostic)
│   ├── velero.sh             # Velero backup/restore (cloud-agnostic)
│   ├── talos.sh              # Talos config and bootstrap (cloud-agnostic)
│   └── gcp/
│       ├── infra.sh          # Terraform operations
│       ├── tunnel.sh         # IAP tunnel management, k8s_connect
│       ├── csi.sh            # GCE PD CSI driver
│       ├── velero.sh         # GCP Velero plugin installation
│       └── verify.sh         # Resource verification
└── README.md
```

## Script Details

### cluster/deploy.sh

Creates a complete Kubernetes cluster.

Usage: `./cluster/deploy.sh <cloud>` (e.g., `./cluster/deploy.sh gcp`)

1. **Validate cloud provider** - Ensures supported cloud
2. **Check prerequisites** - Verifies all tools are installed
3. **Ensure state bucket** - Creates cloud storage bucket if needed
4. **Terraform apply** - Creates VPC, firewall rules, VMs
5. **Generate Talos configs** - Creates controlplane.yaml, worker.yaml
6. **Apply configs** - Sends configs to each node via tunnel
7. **Bootstrap cluster** - Initializes etcd and Kubernetes
8. **Fetch kubeconfig** - Retrieves kubectl credentials

### apps/apply.sh

Deploys applications to the cluster.

Usage: `./apps/apply.sh <cloud>` (e.g., `./apps/apply.sh gcp`)

1. **Start tunnel** - Connects to Kubernetes API
2. **Install CSI driver** - Cloud-specific storage driver
3. **Install Velero** - Cloud-specific backup plugin
4. **Apply StorageClass** - Cloud-specific persistent disk class
5. **Deploy apps** - NGINX and Redis from `apps/` manifests
6. **Stop tunnel** - Cleans up connection

### cluster/destroy.sh

Tears down the cluster and all resources.

Usage: `./cluster/destroy.sh <cloud>` (e.g., `./cluster/destroy.sh gcp`)

1. **Remove apps (best effort)** - Deletes deployments, PVCs (triggers disk deletion)
2. **Terraform destroy** - Removes all cloud resources
3. **Cleanup configs** - Deletes generated configs from `configs/`
4. **Verify destruction** - Confirms all resources are removed

### cluster/connect.sh

Starts an interactive tunnel for manual cluster access.

Usage: `./cluster/connect.sh <cloud>` (e.g., `./cluster/connect.sh gcp`)

Keeps the tunnel open until Ctrl+C. Use in one terminal while running kubectl in another.

### apps/seed-redis.sh

Seeds Redis with test data for Velero backup/restore testing.

Usage: `./apps/seed-redis.sh <cloud>` (e.g., `./apps/seed-redis.sh gcp`)

1. **Start tunnel** - Connects to Kubernetes API
2. **Wait for Redis** - Ensures Redis pod is ready
3. **Insert test data** - Creates sample keys (users, counter, config, queue)
4. **Verify data** - Lists all keys to confirm insertion

Test data includes:
- `user:1`, `user:2`, `user:3` - Sample user JSON objects
- `counter:visits` - Numeric counter
- `config:app:version` - Configuration value
- `queue:tasks` - List with sample tasks

### velero/backup.sh

Backs up applications to cloud storage using Velero.

Usage: `./velero/backup.sh <cloud>` (e.g., `./velero/backup.sh gcp`)

1. **Start tunnel** - Connects to Kubernetes API
2. **Create backup** - Velero backs up application namespace (manifests + volume snapshots)
3. **Verify backup** - Confirms backup status is "Completed"
4. **Stop tunnel** - Cleans up connection

### velero/restore.sh

Restores applications from a Velero backup. Use after `make deploy` on a fresh cluster.

Usage: `./velero/restore.sh <cloud>` (e.g., `./velero/restore.sh gcp`)

1. **Start tunnel** - Connects to Kubernetes API
2. **Install CSI driver** - Required for PVC restore (volumes need a storage provisioner)
3. **Apply StorageClass** - Required for PVC binding
4. **Install Velero** - Cloud-specific plugin to access backup storage
5. **Restore** - Velero recreates resources and rebinds PVCs from snapshots
6. **Verify Velero** - Checks Velero restore CR status
7. **Verify apps** - Waits for pods and verifies data integrity
8. **Stop tunnel** - Cleans up connection

### velero/list-backups.sh

Lists all Velero backups with their status.

Usage: `./velero/list-backups.sh <cloud>` (e.g., `make list-backups gcp`)

### velero/delete-backup.sh

Deletes a Velero backup by name and its associated volume snapshots.

Usage: `./velero/delete-backup.sh <cloud> <name>` (e.g., `make delete-backup gcp NAME=k8s-lab-backup`)

### velero/delete-all-backups.sh

Deletes all Velero backups and their associated volume snapshots.

Usage: `./velero/delete-all-backups.sh <cloud>` (e.g., `make delete-all-backups gcp`)

## Library Modules

### lib/common.sh

Shared utilities sourced by all scripts:

- **Logging**: `log_info`, `log_warn`, `log_error`, `log_step`
- **Error handling**: Trap for exit with line number
- **Prerequisites**: Checks for required tools
- **Path constants**: REPO_ROOT, TF_DIR, CONFIGS_DIR

### lib/apps.sh

Cloud-agnostic application lifecycle:

- `apps_deploy` - Apply all application manifests and wait for readiness
- `apps_remove` - Delete applications and PVCs
- `apps_verify` - Verify deployments are running and check data integrity
- `apps_status` - Show deployment status

### lib/talos.sh

Talos Linux operations (cloud-agnostic):

- `talos_generate_configs` - Create machine configs
- `talos_apply_config` - Send config to a node
- `talos_apply_all_configs` - Configure all nodes
- `talos_bootstrap` - Initialize Kubernetes
- `talos_fetch_kubeconfig` - Get kubectl credentials
- `talos_cleanup_configs` - Remove generated files

### lib/gcp/infra.sh

GCP Terraform operations:

- `tf_create` - Initialize and apply
- `tf_destroy` - Destroy all resources
- `tf_get_outputs` - Read VM IPs, names, zones
- `tf_get_project_id` - Read project ID from tfvars

### lib/gcp/tunnel.sh

GCP IAP tunnel management:

- `tunnel_start` - Open tunnel to VM (returns PID)
- `tunnel_stop` - Close tunnel
- `tunnel_cleanup_all` - Close all tunnels (automatic on exit)
- `k8s_connect` - Cloud-agnostic interface for Kubernetes API access

Why IAP? VMs have no external IPs. IAP provides secure access using gcloud credentials.

The `k8s_connect` function is the multi-cloud interface. Each cloud implements it according to its access method (GCP uses IAP tunnel).

### lib/velero.sh

Cloud-agnostic Velero operations (backup/restore CRs only):

- `velero_wait_ready` - Wait for Velero deployment to be running
- `velero_backup` - Create backup of application namespace
- `velero_verify_backup` - Check backup CR completed successfully
- `velero_list_backups` - List all backups with status
- `velero_delete_backup` - Delete a backup by name
- `velero_delete_all_backups` - Delete all backups
- `velero_restore` - Restore from most recent backup
- `velero_verify_restore` - Check restore CR status only (use `apps_verify` for application checks)

### lib/gcp/csi.sh

GCE Persistent Disk CSI driver:

- `gcp_csi_install` - Install CSI driver from official release
- `gcp_csi_uninstall` - Remove CSI driver

### lib/gcp/velero.sh

GCP Velero plugin installation:

- `gcp_velero_install` - Install Velero with GCS storage and GCE PD snapshots

Uses the same GCS bucket as Terraform state (with `velero/` prefix). Authentication via VM service account (`--no-secret`).

### lib/gcp/verify.sh

Resource verification:

- `gcp_verify_all_destroyed` - Confirm all resources are removed

## Accessing the Cluster

After `make deploy gcp`, access requires an IAP tunnel:

```bash
# Terminal 1 - Start tunnel (keep running)
make connect gcp

# Terminal 2 - Use kubectl
export KUBECONFIG=./configs/kubeconfig
kubectl get nodes
```

## Talos Debugging

Talos has no SSH access. All debugging is done via `talosctl` through an IAP tunnel.

Prerequisites: An IAP tunnel must be running on port 50000.

### Check Service Status

Shows all Talos services and their health.

```bash
talosctl --talosconfig="${PWD}/configs/talosconfig" \
  --nodes localhost --endpoints localhost:50000 \
  service
```

Expected output when healthy:
```
SERVICE      STATE     HEALTH
etcd         Running   OK
kubelet      Running   OK
apid         Running   OK
```

### View Controller Logs

Shows Talos controller-runtime logs.

```bash
talosctl --talosconfig="${PWD}/configs/talosconfig" \
  --nodes localhost --endpoints localhost:50000 \
  logs controller-runtime | tail -100
```

### View System Messages

Shows kernel and system messages.

```bash
talosctl --talosconfig="${PWD}/configs/talosconfig" \
  --nodes localhost --endpoints localhost:50000 \
  dmesg | tail -50
```

### Reboot a Node

Forces a node reboot.

```bash
talosctl --talosconfig="${PWD}/configs/talosconfig" \
  --nodes localhost --endpoints localhost:50000 \
  reboot
```

## Troubleshooting

### "Missing required tools"

Install the missing tool as indicated in the error message.

### IAP tunnel fails

1. Check gcloud authentication: `gcloud auth list`
2. Verify IAP API is enabled in GCP project
3. Check firewall allows IAP (35.235.240.0/20)

### Talos API not ready

VMs take time to boot. The script waits up to 10 minutes. If it still fails:

1. Check VM is running in GCP Console
2. Check VM serial console for boot errors

### etcd stuck in "Preparing"

etcd not starting usually means:

1. **No internet access**: VMs need Cloud NAT to pull images
   - Check: `logs controller-runtime` shows `discovery.talos.dev` errors
   - Fix: Ensure Cloud NAT is configured in Terraform

2. **Bootstrap not initiated**: etcd waits for bootstrap command
   - If interrupted, run `make destroy gcp && make deploy gcp` to start fresh

3. **Network issues**: Nodes can't communicate
   - Check firewall rules allow internal traffic

### Bootstrap fails

Bootstrap can only run once. If it fails mid-way:

1. Run `make destroy gcp`
2. Run `make deploy gcp` to start fresh

### Port already in use

Scripts use IAP tunnels on ports 6443 (Kubernetes API) and 50000 (Talos API). If a script fails or is interrupted, tunnel processes may remain running and block subsequent runs.

Symptoms:
- "Address already in use" error
- Tunnel fails to start
- Script hangs at tunnel creation

Find and kill orphaned tunnel processes:

```bash
# Check which process is using the port
sudo lsof -i :6443
sudo lsof -i :50000

# Kill the process (use PID from lsof output)
kill <pid>

# Force kill if needed
kill -9 <pid>
```

One-liner to kill all gcloud tunnels:

```bash
pkill -f "gcloud.*start-iap-tunnel"
```
