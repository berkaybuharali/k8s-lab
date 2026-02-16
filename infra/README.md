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
| `roles/artifactregistry.admin` | Create Artifact Registry repository for agent containers |

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

### One-Time Setup: Artifact Registry

Before first `deploy-infra` run, create the Artifact Registry repository:

```bash
gcloud artifacts repositories create k8s-lab \
  --repository-format=docker \
  --location=europe-west4 \
  --description="Container images for Magic Cake agents" \
  --project=<your-project-id>
```

This persists across daily infrastructure destroy/create cycles (Terraform references it but doesn't manage it).

### Resources Created

**Terraform manages (created/destroyed daily):**
- Service account for Talos nodes (with GCS and Artifact Registry permissions)
- VPC network with Cloud NAT
- Firewall rules (IAP tunneling + internal cluster communication)
- Compute instances: 1 control plane + 2 workers (multi-AZ)
- IAM bindings for Artifact Registry access

**Persistent (not managed by Terraform):**
- Artifact Registry repository `k8s-lab` (created once, referenced by Terraform)

**Service Account Permissions:**
- `roles/storage.objectAdmin` - Read/write GCS buckets (Velero backups)
- `roles/compute.storageAdmin` - Create persistent disks (CSI driver)
- `roles/artifactregistry.reader` - Pull agent container images

### Shared Projects

Use unique `name_prefix`, `network_name`, and `bucket` values per user.
