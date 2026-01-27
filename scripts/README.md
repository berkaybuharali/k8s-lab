# Scripts

Automation scripts for cluster lifecycle management.

For prerequisites, see [infra/README.md](../infra/README.md#prerequisites).

## Structure

Scripts are organized by architecture layer for clarity:

```
scripts/
├── infra/                    # Layer 1-2: Infrastructure & Kubernetes cluster
│   ├── deploy.sh             # Create infrastructure and bootstrap cluster
│   ├── destroy.sh            # Destroy all resources
│   └── connect.sh            # Start interactive tunnel for manual kubectl access
├── platform/                 # Layer 3: Platform services (CSI, Velero)
│   └── deploy.sh             # Deploy CSI driver, StorageClass, Velero
├── workloads/                # Layer 4: Application workloads
│   ├── deploy.sh             # Deploy NGINX and Redis
│   └── seed-redis.sh         # Seed Redis with test data
├── backup/                   # Backup operations (uses platform layer)
│   ├── create.sh             # Create backup
│   ├── restore.sh            # Restore from backup (includes platform setup)
│   ├── list.sh               # List all backups
│   ├── delete.sh             # Delete a backup by name
│   └── delete-all.sh         # Delete all backups
├── lib/                      # Shared libraries (cloud-agnostic + cloud-specific)
│   ├── common.sh             # Logging, prerequisites, utilities
│   ├── workloads.sh          # Workload helper functions (cloud-agnostic)
│   ├── velero.sh             # Velero backup/restore (cloud-agnostic)
│   ├── talos.sh              # Talos config and bootstrap (cloud-agnostic)
│   └── gcp/                  # GCP-specific implementations
│       ├── infra.sh          # Terraform operations
│       ├── tunnel.sh         # IAP tunnel management, k8s_connect
│       ├── csi.sh            # GCE PD CSI driver
│       ├── velero.sh         # GCP Velero plugin installation
│       └── verify.sh         # Resource verification
└── README.md
```

## Script Details

### infra/deploy.sh

Creates infrastructure and bootstraps a Kubernetes cluster.

Usage: `./infra/deploy.sh <cloud>` (via `make deploy-infra gcp`)

1. **Validate cloud provider** - Ensures supported cloud
2. **Check prerequisites** - Verifies all tools are installed
3. **Ensure state bucket** - Creates cloud storage bucket if needed
4. **Terraform apply** - Creates VPC, firewall rules, VMs
5. **Generate Talos configs** - Creates controlplane.yaml, worker.yaml
6. **Apply configs** - Sends configs to each node via tunnel
7. **Bootstrap cluster** - Initializes etcd and Kubernetes
8. **Fetch kubeconfig** - Retrieves kubectl credentials

### platform/deploy.sh

Deploys platform services required for applications and backups.

Usage: `./platform/deploy.sh <cloud>` (via `make deploy-tools gcp`)

1. **Start tunnel** - Connects to Kubernetes API
2. **Install CSI driver** - Cloud-specific storage driver
3. **Apply StorageClass** - Cloud-specific persistent disk class
4. **Install Velero** - Cloud-specific backup plugin
5. **Stop tunnel** - Cleans up connection

### workloads/deploy.sh

Deploys application workloads to the cluster.

Usage: `./workloads/deploy.sh <cloud>` (via `make deploy-applications gcp`)

1. **Start tunnel** - Connects to Kubernetes API
2. **Create namespace** - application namespace
3. **Deploy NGINX** - Stateless web server (2 replicas)
4. **Deploy Redis** - Stateful cache with persistent storage
5. **Wait for readiness** - Ensures deployments are running
6. **Stop tunnel** - Cleans up connection

### infra/destroy.sh

Tears down the cluster and all resources.

Usage: `./infra/destroy.sh <cloud>` (via `make destroy gcp`)

1. **Remove workloads (best effort)** - Deletes deployments, PVCs (triggers disk deletion)
2. **Terraform destroy** - Removes all cloud resources
3. **Cleanup configs** - Deletes generated configs from `configs/`
4. **Verify destruction** - Confirms all resources are removed

### infra/connect.sh

Starts an interactive tunnel for manual cluster access.

Usage: `./infra/connect.sh <cloud>` (via `make connect gcp`)

Keeps the tunnel open until Ctrl+C. Use in one terminal while running kubectl in another.

### workloads/seed-redis.sh

Seeds Redis with test data for backup/restore testing.

Usage: `./workloads/seed-redis.sh <cloud>` (via `make seed-redis gcp`)

1. **Start tunnel** - Connects to Kubernetes API
2. **Wait for Redis** - Ensures Redis pod is ready
3. **Insert test data** - Creates sample keys (users, counter, config, queue)
4. **Verify data** - Lists all keys to confirm insertion

Test data includes:
- `user:1`, `user:2`, `user:3` - Sample user JSON objects
- `counter:visits` - Numeric counter
- `config:app:version` - Configuration value
- `queue:tasks` - List with sample tasks

### backup/create.sh

Creates a backup of applications to cloud storage.

Usage: `./backup/create.sh <cloud>` (via `make backup gcp`)

1. **Start tunnel** - Connects to Kubernetes API
2. **Create backup** - Velero backs up application namespace (manifests + volume snapshots)
3. **Verify backup** - Confirms backup status is "Completed"
4. **Stop tunnel** - Cleans up connection

### backup/restore.sh

Restores applications from a backup. Use after `make deploy-infra` on a fresh cluster.

Usage: `./backup/restore.sh <cloud>` (via `make restore gcp`)

1. **Start tunnel** - Connects to Kubernetes API
2. **Install platform** - CSI driver, StorageClass, Velero (required for PVC restore)
3. **Restore** - Velero recreates resources and rebinds PVCs from snapshots
4. **Verify Velero** - Checks Velero restore CR status
5. **Verify workloads** - Waits for pods and verifies data integrity
6. **Stop tunnel** - Cleans up connection

### backup/list.sh

Lists all backups with their status.

Usage: `./backup/list.sh <cloud>` (via `make list-backups gcp`)

### backup/delete.sh

Deletes a backup by name and its associated volume snapshots.

Usage: `./backup/delete.sh <cloud> <name>` (via `make delete-backup gcp NAME=k8s-lab-backup`)

### backup/delete-all.sh

Deletes all backups and their associated volume snapshots.

Usage: `./backup/delete-all.sh <cloud>` (via `make delete-all-backups gcp`)

## Library Modules

### lib/common.sh

Shared utilities sourced by all scripts:

- **Logging**: `log_info`, `log_warn`, `log_error`, `log_step`
- **Error handling**: Trap for exit with line number
- **Prerequisites**: Checks for required tools
- **Path constants**: REPO_ROOT, TF_DIR, CONFIGS_DIR

### lib/workloads.sh

Cloud-agnostic workload helper functions:

- `apps_remove` - Delete applications and PVCs
- `apps_verify` - Verify deployments are running and check data integrity
- `apps_status` - Show deployment status

Note: Functions retain `apps_*` names for compatibility as they operate on application resources.

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

After `make deploy-infra gcp`, access requires an IAP tunnel:

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
