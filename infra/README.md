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

### Talos Linux Image

VMs boot from a Talos Linux image. This is a one-time setup per GCP project.

```bash
# Set your project
export PROJECT_ID="your-project-id"
export BUCKET_NAME="your-bucket-name"

# Download Talos v1.12.1 for GCP
curl -L -o /tmp/talos-gcp.raw.tar.gz \
  "https://factory.talos.dev/image/376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba/v1.12.1/gcp-amd64.raw.tar.gz"

# Upload to GCS (create bucket if needed)
gsutil mb -p ${PROJECT_ID} gs://${BUCKET_NAME} 2>/dev/null || true
gsutil cp /tmp/talos-gcp.raw.tar.gz gs://${BUCKET_NAME}/images/talos-v1.12.1-gcp-amd64.raw.tar.gz

# Create compute image
gcloud compute images create talos-v1-12-1 \
  --project=${PROJECT_ID} \
  --source-uri=gs://${BUCKET_NAME}/images/talos-v1.12.1-gcp-amd64.raw.tar.gz \
  --guest-os-features=VIRTIO_SCSI_MULTIQUEUE
```

The schematic ID `376567988ad370138ad8b2698212367b8edcb69b5fd68c80be1f2ec7d603b4ba` is the default vanilla Talos image from [Talos Image Factory](https://factory.talos.dev/).

### Configuration

#### 1. Copy Example Files

```bash
cd infra/gcp/terraform
cp terraform.tfvars.example terraform.tfvars
cp backend.tf.example backend.tf
```

#### 2. Edit terraform.tfvars

| Variable | Description | Example |
|----------|-------------|---------|
| `project_id` | Your GCP project ID | `my-project-123` |
| `name_prefix` | Unique prefix for resources (use initials) | `bb-talos` |
| `boot_image` | Talos image path | `projects/my-project/global/images/talos-v1-12-1` |
| `network_name` | VPC name (unique per user in shared projects) | `bb-talos-vpc` |
| `worker_count` | Number of worker nodes | `2` |

#### 3. Edit backend.tf

Set `bucket` to a globally unique name for Terraform state storage. The bucket will be created automatically on first `make up`.

### Terraform Structure

```
gcp/terraform/
├── backend.tf          # GCS state backend (gitignored)
├── backend.tf.example  # Template for backend.tf
├── providers.tf        # Provider configuration
├── variables.tf        # Variable definitions
├── terraform.tfvars    # Your values (gitignored)
├── terraform.tfvars.example  # Template for tfvars
├── network.tf          # VPC, subnet, Cloud NAT
├── firewall.tf         # IAP and internal firewall rules
├── gce.tf              # VM instances
└── outputs.tf          # Output values for scripts
```

### What Gets Created

| Resource | Description |
|----------|-------------|
| VPC Network | Custom VPC with one subnet |
| Cloud NAT | Outbound internet access for VMs (no public IPs) |
| Firewall Rules | IAP access (22, 50000, 6443), internal cluster traffic |
| Control Plane | 1x VM running Talos Linux |
| Workers | 2x VMs (configurable) across availability zones |

### Running Multiple Labs in Same Project

If multiple people use the same GCP project, each person must use unique values for:

- `name_prefix` in terraform.tfvars (e.g., your initials)
- `network_name` in terraform.tfvars
- `bucket` in backend.tf
