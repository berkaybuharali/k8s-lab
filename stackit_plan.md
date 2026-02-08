# STACKIT Cloud Extension Plan

## Current Status
- [ ] Phase 0: Planning & Discussion (IN PROGRESS)

---

## Context

STACKIT is a German cloud provider (owned by Schwarz Group / Lidl), EU-based, GDPR-compliant.
The k8s-lab project already supports GCP with a clean cloud-agnostic architecture.
Adding STACKIT means implementing the cloud-specific layer while reusing all agnostic code.

---

## Research Summary

### STACKIT Capabilities vs Requirements

| Requirement | STACKIT | GCP Equivalent | Notes |
|-------------|---------|----------------|-------|
| Terraform | `stackitcloud/stackit` provider (v0.79, pre-1.0) | `google` provider | All needed resources exist |
| CLI | `stackit` (brew install) | `gcloud` | Auth, resource management |
| VMs | `stackit_server` | `google_compute_instance` | Multiple machine types |
| Custom Images | `stackit_image` + API upload | `google_compute_image` | Upload Talos raw/qcow2 via API |
| Networking | `stackit_network` + security groups | VPC + firewall rules | Similar concepts |
| Public IPs | Floating IPs (`stackit_public_ip`) | External IP on instance | VMs get direct public IPs |
| Block Storage | `stackit_volume` (OpenStack Cinder) | Persistent Disk | 8 performance classes |
| CSI Driver | OpenStack Cinder CSI | GCE PD CSI | `cinder.csi.openstack.org` |
| Object Storage | S3-compatible | GCS | Works with Velero AWS plugin |
| TF State | S3 backend (with STACKIT endpoint) | GCS backend | Different backend config |
| Auth | Service Account + RSA key pair | `gcloud auth` / SA JSON | Key-based JWT signing |

### Key Differences from GCP

1. **No tunneling needed** -- STACKIT VMs get floating public IPs directly (GCP uses IAP tunnels)
2. **S3 instead of GCS** -- Object storage is S3-compatible, so Velero uses AWS plugin (not GCP plugin)
3. **Terraform state** -- Uses S3 backend instead of GCS backend
4. **CSI driver** -- OpenStack Cinder CSI instead of GCE PD CSI
5. **Image upload** -- Talos image must be uploaded via STACKIT API (not a public image reference)
6. **Auth model** -- RSA key pair for service accounts (no `gcloud auth application-default login` equivalent)

### What We Cannot Build (or Differs Significantly)

| Area | Status | Detail |
|------|--------|--------|
| Multi-AZ workers | Investigate | STACKIT has 2 regions (Germany-South, Austria-West) but AZ structure within regions unclear |
| IAP-style tunneling | Not needed | Public IPs are standard, no tunnel required |
| Terraform maturity | Risk | Pre-1.0 provider, possible breaking changes |
| Image management | Extra step | Talos image upload is API-based, needs scripting |

---

## Architecture: What Changes per Layer

```
Layer 1 (Infrastructure) -- CLOUD-SPECIFIC, must implement
  infra/stackit/terraform/     -- New Terraform configs

Layer 2 (Cluster) -- MOSTLY AGNOSTIC, minor changes
  Talos config generation      -- Reuse existing (different endpoint format)
  Talos patches                -- New CSI patch for Cinder

Layer 3 (Platform) -- CLOUD-SPECIFIC hooks, agnostic logic
  CSI driver                   -- Cinder CSI instead of GCE PD
  StorageClass                 -- Cinder provisioner
  Velero                       -- AWS plugin + S3 bucket

Layer 4 (Applications) -- FULLY AGNOSTIC, no changes
  nginx.yaml, redis.yaml       -- Reused as-is
```

---

## Implementation Phases

### Phase 1: Foundation (Bash Scripts)

**1a. Directory structure**
```
infra/stackit/
  terraform/
    providers.tf              # stackitcloud/stackit provider
    backend.tf                # S3 backend for TF state
    backend.tf.example
    variables.tf              # project_id, region, machine_type, image_id, etc.
    terraform.tfvars.example
    network.tf                # stackit_network, stackit_security_group
    compute.tf                # stackit_server (1 CP + 2 workers)
    outputs.tf                # IPs, instance names
  talos-patches/
    csi.yaml                  # Cinder CSI mount paths

scripts/lib/stackit/
    infra.sh                  # tf_create, tf_destroy, tf_get_outputs
    csi.sh                    # stackit_csi_install (Cinder CSI)
    velero.sh                 # stackit_velero_install (AWS plugin + S3)

apps/stackit/
    storageclass.yaml         # cinder.csi.openstack.org provisioner
```

**1b. Update dispatch points (Bash)**
- `scripts/lib/common.sh` -- Add "stackit" to SUPPORTED_CLOUDS, add case in `source_cloud_modules`
- `scripts/platform/deploy.sh` -- Add stackit cases in `install_csi_driver`, `install_velero`

**1c. Key difference: No tunnel needed**
- GCP: `tunnel_start` / `tunnel_stop` / `k8s_connect` (IAP)
- STACKIT: Direct connection to public floating IPs
- `scripts/lib/stackit/infra.sh` exports endpoints directly (no tunnel functions needed)
- The talos.sh and other agnostic scripts use `$TALOS_ENDPOINT` and `$K8S_ENDPOINT` -- these just point to the public IP

### Phase 2: Terraform (Infrastructure as Code)

**2a. Networking**
```hcl
resource "stackit_network" "main" { ... }
resource "stackit_security_group" "k8s" { ... }
resource "stackit_security_group_rule" "talos_api" {
  # Port 50000 from user IP
}
resource "stackit_security_group_rule" "k8s_api" {
  # Port 6443 from user IP
}
resource "stackit_security_group_rule" "internal" {
  # All traffic between nodes
}
resource "stackit_security_group_rule" "ssh" {
  # Port 22 (optional, for debugging)
}
```

**2b. Compute**
```hcl
resource "stackit_server" "control_plane" {
  name         = "${var.name_prefix}-cp"
  machine_type = var.machine_type        # e.g. "g2i.2" (2 vCPU, 8 GB)
  image_id     = var.boot_image_id       # Pre-uploaded Talos image
  boot_volume { size = var.boot_disk_size }
  network_interface { network_id = stackit_network.main.id }
}

resource "stackit_public_ip" "cp" { ... }
resource "stackit_public_ip_associate" "cp" {
  server_id = stackit_server.control_plane.id
  ip_id     = stackit_public_ip.cp.id
}

# Similar for workers (count or for_each)
```

**2c. Backend (S3)**
```hcl
terraform {
  backend "s3" {
    bucket   = "k8s-lab-tf-state"
    key      = "stackit/terraform.tfstate"
    endpoint = "https://object.storage.eu01.onstackit.cloud"
    region   = "eu01"
    # S3-compatible settings
    skip_credentials_validation = true
    skip_metadata_api_check     = true
    skip_region_validation      = true
    force_path_style            = true
  }
}
```

**2d. Outputs**
```hcl
output "control_plane_public_ip" { ... }
output "worker_public_ips" { ... }
output "control_plane_private_ip" { ... }
output "worker_private_ips" { ... }
```

### Phase 3: Talos + Platform

**3a. Talos configuration**
- Endpoint = public floating IP of control plane (no tunnel)
- Generate configs with `talosctl gen config` using public IP as endpoint
- Apply configs using public IP directly

**3b. CSI Driver (Cinder)**
- Deploy OpenStack Cinder CSI driver
- Needs cloud-config secret with STACKIT credentials (Keystone endpoint, project ID, auth)
- Talos patch: mount paths for Cinder CSI

**3c. StorageClass**
```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: csi-cinder
provisioner: cinder.csi.openstack.org
parameters:
  type: storage_premium_perf2   # or appropriate STACKIT storage class
volumeBindingMode: WaitForFirstConsumer
allowVolumeExpansion: true
```

**3d. Velero (S3-compatible)**
- Use `velero-plugin-for-aws` (S3-compatible)
- Configure with STACKIT Object Storage endpoint
- Credential: STACKIT access key + secret key for Object Storage

### Phase 4: Go CLI Extension

**4a. New package: `cli/pkg/cloud/stackit/`**
```
cli/pkg/cloud/stackit/
    provider.go      # Name(), Validate(), GetProjectID()
    storage.go       # EnsureStateBucket() -- S3 bucket creation
    platform.go      # InstallCSIDriver(), GetVeleroInstallConfig()
```

Note: No tunnel.go needed -- CreateTalosEndpoint and CreateK8sEndpoint return the public IP directly (no tunnel cleanup function).

**4b. Update registrations**
- `cli/pkg/config/config.go` -- Add "stackit" to SupportedClouds
- `cli/pkg/prerequisites/prerequisite.go` -- Add Stackit CLI prerequisite, cloud prereqs
- Provider auto-registers via `init()` in stackit package

### Phase 5: Talos Image Management

**5a. Image upload script/command**
- Download Talos OpenStack image from factory.talos.dev (raw format)
- Upload to STACKIT via API:
  1. Create image metadata (`POST /v1/projects/{id}/images`)
  2. Upload binary (`PUT` to upload URL)
  3. Wait for status = AVAILABLE
- Store image ID in terraform.tfvars

**5b. Decision: manual vs automated**
- Option A: Document manual steps (simpler, one-time per Talos version)
- Option B: Script/Make target `make upload-image stackit` (automation)
- Recommend: Start with Option A, automate later if needed

### Phase 6: Documentation & Testing

- Create `infra/stackit/README.md` with setup instructions
- Create `terraform.tfvars.example` and `backend.tf.example`
- Update root README.md with STACKIT quick start
- Update CLAUDE.md if needed
- Update LOCAL.md with STACKIT prerequisites
- End-to-end test: deploy-infra, deploy-tools, deploy-applications, backup, destroy

---

## Open Questions for Discussion

1. **Talos image upload** -- Manual or scripted? First time is always manual (need STACKIT account first).

2. **STACKIT authentication** -- Service account key flow requires:
   - Create SA in STACKIT portal
   - Generate RSA key pair
   - Set env vars or config file
   - How to document this cleanly?

3. **Terraform state bucket** -- S3 backend needs credentials configured before `terraform init`. Chicken-and-egg: create bucket via CLI first, then configure backend?

4. **Cinder CSI credentials** -- The CSI driver needs a cloud-config with Keystone auth. How does STACKIT expose this? (Likely via their OpenStack-compatible API endpoint)

5. **Machine types** -- Suggested:
   - Control plane: `g2i.2` (2 vCPU, 8 GB) or `c2i.2` (2 vCPU, 4 GB)
   - Workers: `c2i.2` (2 vCPU, 4 GB)
   - Cheapest viable: `c2i.1` (1 vCPU, 2 GB) -- may be too small for K8s

6. **Region** -- Default to `eu01` (Germany-South)? Only 2 regions available.

7. **Multi-AZ** -- GCP spreads workers across zones. Does STACKIT have availability zones within a region? If not, all nodes go in same location.

8. **Object Storage credentials** -- Separate from compute credentials? Need access key + secret key for S3 API.

---

## Estimated Effort per Phase

| Phase | Scope | Depends On |
|-------|-------|------------|
| 1. Foundation | Directory structure, bash dispatch | Nothing |
| 2. Terraform | All .tf files for STACKIT | Phase 1, STACKIT account for testing |
| 3. Talos + Platform | CSI, StorageClass, Velero config | Phase 2 |
| 4. Go CLI | Provider implementation | Phase 1-3 (can parallel with Phase 2-3) |
| 5. Image Management | Talos upload script/docs | STACKIT account |
| 6. Docs & Testing | READMEs, e2e test | All phases |

---

## File Change Summary

### New Files
```
infra/stackit/terraform/providers.tf
infra/stackit/terraform/backend.tf.example
infra/stackit/terraform/variables.tf
infra/stackit/terraform/terraform.tfvars.example
infra/stackit/terraform/network.tf
infra/stackit/terraform/compute.tf
infra/stackit/terraform/outputs.tf
infra/stackit/talos-patches/csi.yaml
infra/stackit/README.md

scripts/lib/stackit/infra.sh
scripts/lib/stackit/csi.sh
scripts/lib/stackit/velero.sh

apps/stackit/storageclass.yaml

cli/pkg/cloud/stackit/provider.go
cli/pkg/cloud/stackit/storage.go
cli/pkg/cloud/stackit/platform.go
```

### Modified Files
```
scripts/lib/common.sh              # SUPPORTED_CLOUDS, source_cloud_modules, check_prerequisites
scripts/platform/deploy.sh         # install_csi_driver, install_velero cases

cli/pkg/config/config.go           # SupportedClouds
cli/pkg/prerequisites/prerequisite.go  # Stackit prereq, CloudPrereqs

README.md                          # STACKIT quick start
LOCAL.md                           # STACKIT prerequisites
CLAUDE.md                          # If architecture table needs update
```

### Unchanged (Cloud-Agnostic, Reused As-Is)
```
scripts/lib/talos.sh
scripts/lib/velero.sh
scripts/lib/workloads.sh
scripts/infra/deploy.sh            # Already dispatches via source_cloud_modules
scripts/infra/destroy.sh
scripts/workloads/deploy.sh
scripts/workloads/seed-redis.sh
scripts/backup/*.sh
apps/namespace.yaml
apps/nginx.yaml
apps/redis.yaml
Makefile                           # Already passes $(CLOUD) generically
cli/cmd/*.go                       # Commands are cloud-agnostic
cli/pkg/terraform/terraform.go
cli/pkg/talos/*.go
cli/pkg/k8s/client.go
cli/pkg/velero/client.go
```
