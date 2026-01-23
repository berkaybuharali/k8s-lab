# Scripts

Automation scripts for cluster lifecycle management.

## Quick Start

```bash
make up      # Create cluster (VPC, VMs, Kubernetes)
make down    # Destroy everything
```

## Prerequisites

Install these tools before running scripts:

| Tool | Install | Purpose |
|------|---------|---------|
| gcloud | [Google Cloud SDK](https://cloud.google.com/sdk/docs/install) | IAP tunneling to VMs |
| terraform | `brew install terraform` | Infrastructure provisioning |
| talosctl | `brew install talosctl` | Talos cluster management |
| jq | `brew install jq` | JSON parsing |

## Structure

```
scripts/
├── setup.sh          # Main: create cluster
├── destroy.sh        # Main: destroy cluster
├── lib/
│   ├── common.sh     # Logging, prerequisites, utilities
│   └── gcp/
│       ├── terraform.sh  # Terraform create/destroy/outputs
│       ├── tunnel.sh     # IAP tunnel management
│       └── verify.sh     # Resource verification
│   └── talos.sh      # Talos config and bootstrap
└── README.md
```

## Script Details

### setup.sh

Creates a complete Kubernetes cluster:

1. **Check prerequisites** - Verifies all tools are installed
2. **Ensure state bucket** - Creates GCS bucket if needed
3. **Terraform apply** - Creates VPC, firewall rules, VMs
4. **Generate Talos configs** - Creates controlplane.yaml, worker.yaml
5. **Apply configs** - Sends configs to each node via IAP tunnel
6. **Bootstrap cluster** - Initializes etcd and Kubernetes
7. **Fetch kubeconfig** - Retrieves kubectl credentials

### destroy.sh

Tears down the cluster:

1. **Terraform destroy** - Removes all cloud resources
2. **Cleanup configs** - Deletes generated configs from `configs/`
3. **Verify destruction** - Confirms all resources are removed

## Library Modules

### lib/common.sh

Shared utilities sourced by all scripts:

- **Logging**: `log_info`, `log_warn`, `log_error`, `log_step`
- **Error handling**: Trap for exit with line number
- **Prerequisites**: Checks for required tools
- **Path constants**: REPO_ROOT, TF_DIR, CONFIGS_DIR

### lib/terraform.sh

Terraform operations:

- `tf_create` - Initialize and apply
- `tf_destroy` - Destroy all resources
- `tf_get_outputs` - Read VM IPs, names, zones

### lib/tunnel.sh

IAP tunnel management:

- `tunnel_start` - Open tunnel to VM (returns PID)
- `tunnel_stop` - Close tunnel
- `tunnel_cleanup_all` - Close all tunnels (automatic on exit)

Why IAP? VMs have no external IPs. IAP provides secure access using gcloud credentials.

### lib/talos.sh

Talos Linux operations:

- `talos_generate_configs` - Create machine configs
- `talos_apply_config` - Send config to a node (low-level)
- `talos_apply_node_config` - Apply config to single node with tunnel management
- `talos_apply_cp_config` - Configure control plane node
- `talos_apply_worker_configs` - Configure all worker nodes
- `talos_apply_all_configs` - Configure all nodes (orchestrator)
- `talos_wait_for_api_ready` - Wait for Talos API (authenticated mode)
- `talos_bootstrap` - Initialize Kubernetes
- `talos_fetch_kubeconfig` - Get kubectl credentials
- `talos_cleanup_configs` - Remove generated files

## Accessing the Cluster

After `make up`, access requires an IAP tunnel:

```bash
# Terminal 1 - Start tunnel (keep running)
# Replace <name-prefix> with your name_prefix from terraform.tfvars (e.g., bb-talos)
gcloud compute start-iap-tunnel <name-prefix>-cp-0 6443 \
  --local-host-port=localhost:6443 \
  --zone=europe-west4-a \
  --project=<your-project>

# Terminal 2 - Use kubectl
export KUBECONFIG=./configs/kubeconfig
kubectl get nodes
```

## Talos Debugging

Talos has no SSH access. All debugging is done via `talosctl` through an IAP tunnel.

Prerequisites: An IAP tunnel must be running on port 50000 (the setup script starts one automatically during bootstrap).

### Check Service Status

Shows all Talos services and their health. Key services: etcd, kubelet, apid.

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

If etcd or kubelet are stuck in `Preparing`, see troubleshooting below.

### View Controller Logs

Shows Talos controller-runtime logs. Useful for diagnosing bootstrap and network issues.

```bash
talosctl --talosconfig="${PWD}/configs/talosconfig" \
  --nodes localhost --endpoints localhost:50000 \
  logs controller-runtime | tail -100
```

Common errors:
- `discovery.talos.dev:443` unreachable: VMs have no internet access (need Cloud NAT)
- `NodeApplyController` timeout: etcd not running yet

### View System Messages

Shows kernel and system messages (similar to dmesg).

```bash
talosctl --talosconfig="${PWD}/configs/talosconfig" \
  --nodes localhost --endpoints localhost:50000 \
  dmesg | tail -50
```

### Reboot a Node

Forces a node reboot. Use when services are stuck and not recovering.

```bash
talosctl --talosconfig="${PWD}/configs/talosconfig" \
  --nodes localhost --endpoints localhost:50000 \
  reboot
```

Wait 2-3 minutes after reboot, then check services again.

### Manual Health Check

Runs the same health check the bootstrap script uses.

```bash
talosctl --talosconfig="${PWD}/configs/talosconfig" \
  --nodes localhost --endpoints localhost:50000 \
  health --wait-timeout=60s
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

1. **No internet access**: VMs need Cloud NAT to pull images and reach discovery service
   - Check: `logs controller-runtime` shows `discovery.talos.dev` errors
   - Fix: Ensure Cloud NAT is configured in Terraform

2. **Bootstrap not initiated**: etcd waits for bootstrap command
   - The setup script runs `talosctl bootstrap` automatically
   - If interrupted, run `make down && make up` to start fresh

3. **Network issues**: Nodes can't communicate
   - Check firewall rules allow internal traffic

### Bootstrap fails

Bootstrap can only run once. If it fails mid-way:

1. Run `make down` to destroy
2. Run `make up` to start fresh
