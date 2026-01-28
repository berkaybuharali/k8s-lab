# Scripts

Automation scripts for cluster lifecycle management. See [root README.md](../README.md#repository-structure) for complete structure.

**All commands run from repository root** using Makefile targets (e.g., `make deploy-infra gcp`).

## Script Details

### infra/deploy.sh

`make deploy-infra gcp` - Creates VMs, bootstraps Kubernetes, fetches kubeconfig.

### platform/deploy.sh

`make deploy-tools gcp` - Installs CSI driver, StorageClass, Velero.

### workloads/deploy.sh

`make deploy-applications gcp` - Deploys NGINX (2 replicas) and Redis (1 replica + PVC).

### infra/destroy.sh

`make destroy gcp` - Removes workloads, destroys infrastructure, cleans configs.

### infra/connect.sh

`make connect gcp` - Opens tunnel for kubectl access. Keep running in separate terminal.

### workloads/seed-redis.sh

`make seed-redis gcp` - Inserts test data (users, counter, config, queue).

### backup/create.sh

`make backup gcp` - Creates Velero backup (manifests + volume snapshots). Redis BGSAVE hooks auto-enabled via `configs/velero/backup-hooks.yaml`.

### backup/restore.sh

`make restore gcp` - Restores from latest backup, verifies data. Requires platform tools already installed. Redis PING validation auto-enabled.

Flow: `make deploy-infra gcp && make deploy-tools gcp && make restore gcp`

### backup/list.sh

`make list-backups gcp` - Lists all backups.

### backup/delete.sh

`make delete-backup gcp NAME=<name>` - Deletes backup and volume snapshots.

### backup/delete-all.sh

`make delete-all-backups gcp` - Deletes all backups.

## Cluster Access

Terminal 1: `make connect gcp` (keep running)
Terminal 2: `export KUBECONFIG=./configs/talos/kubeconfig && kubectl get nodes`

## Talos Debugging

Requires IAP tunnel on port 50000. All commands use:
```bash
talosctl --talosconfig="${PWD}/configs/talos/talosconfig" \
  --nodes localhost --endpoints localhost:50000 <command>
```

Commands: `service` (check status), `logs controller-runtime`, `dmesg`, `reboot`.

## Troubleshooting

**IAP tunnel fails:** Check `gcloud auth list`, verify IAP API enabled, firewall allows 35.235.240.0/20.

**Talos API not ready:** Check VM running in console, view serial console for errors.

**etcd stuck:** No internet (check Cloud NAT), bootstrap not run (`make destroy && make deploy`), firewall issues.

**Bootstrap fails:** One-time operation. `make destroy gcp && make deploy gcp` to retry.

**Port in use:** Kill orphaned tunnels: `pkill -f "gcloud.*start-iap-tunnel"` or `sudo lsof -i :6443` / `kill <pid>`.
