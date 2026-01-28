# Infrastructure

Cloud infrastructure definitions using Terraform.

## Prerequisites

### Required Tools

| Tool | Install | Purpose |
|------|---------|---------|
| terraform | `brew install terraform` | Infrastructure provisioning |
| talosctl | `brew install siderolabs/tap/talosctl` | Talos cluster management |
| kubectl | `brew install kubectl` | Kubernetes CLI |
| jq | `brew install jq` | JSON parsing |
| velero | `brew install velero` | Backup/restore CLI |

Platform-specific CLI tools are listed in each platform section below.

## Supported Platforms

| Platform | Status | Path |
|----------|--------|------|
| GCP (GCE) | Supported | [gcp/terraform/](#gcp-setup) |

---

## GCP Setup

### Additional Tools

| Tool | Install | Purpose |
|------|---------|---------|
| gcloud | [Google Cloud SDK](https://cloud.google.com/sdk/docs/install) | GCP CLI, IAP tunneling |

### Authentication

This project uses Application Default Credentials with your user account:

```bash
gcloud auth application-default login
```

Your user account must have the following IAM roles:

| Role | Purpose |
|------|---------|
| `roles/compute.instanceAdmin.v1` | Create/delete VMs |
| `roles/compute.networkAdmin` | VPC, subnets, firewall rules |
| `roles/compute.storageAdmin` | Create Talos boot image |
| `roles/storage.admin` | Terraform state bucket |
| `roles/iap.tunnelResourceAccessor` | IAP TCP tunnels to VMs |
| `roles/iam.serviceAccountAdmin` | Create service account for Talos nodes |
| `roles/resourcemanager.projectIamAdmin` | Grant roles to the Talos node service account |

### Talos Linux Image

One-time setup per GCP project:

```bash
export PROJECT_ID="your-project-id"
export BUCKET_NAME="your-bucket-name"

# Download Talos v1.12.1 for GCP
curl -L -o /tmp/talos-gcp.raw.tar.gz \
  "https://factory.talos.dev/image/376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba/v1.12.1/gcp-amd64.raw.tar.gz"

# Upload to GCS
gsutil mb -p ${PROJECT_ID} gs://${BUCKET_NAME} 2>/dev/null || true
gsutil cp /tmp/talos-gcp.raw.tar.gz gs://${BUCKET_NAME}/images/talos-v1.12.1-gcp-amd64.raw.tar.gz

# Create compute image
gcloud compute images create talos-v1-12-1 \
  --project=${PROJECT_ID} \
  --source-uri=gs://${BUCKET_NAME}/images/talos-v1.12.1-gcp-amd64.raw.tar.gz \
  --guest-os-features=VIRTIO_SCSI_MULTIQUEUE
```

Schematic ID is vanilla Talos from [Image Factory](https://factory.talos.dev/).

### Configuration

Copy example files and edit:

```bash
cd infra/gcp/terraform
cp terraform.tfvars.example terraform.tfvars
cp backend.tf.example backend.tf
```

Key variables in `terraform.tfvars`: `project_id`, `name_prefix`, `boot_image`, `network_name`, `worker_count`.
Set `bucket` in `backend.tf` (created automatically on first run).

### Resources Created

Service account, VPC, Cloud NAT, firewall rules (IAP + internal), 1 control plane + 2 workers (multi-AZ).

### Shared Projects

Use unique `name_prefix`, `network_name`, and `bucket` values per user.
