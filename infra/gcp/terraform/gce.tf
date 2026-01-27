# -----------------------------------------------------------------------------
# GCE Compute Instances
# -----------------------------------------------------------------------------
# Control plane and worker nodes for the Kubernetes cluster.
# - No external IPs: access via IAP TCP tunnel (talosctl, not SSH)
# - Workers distributed across zones for availability
# - Boot image is Talos Linux (immutable OS for Kubernetes)
# -----------------------------------------------------------------------------

# -----------------------------------------------------------------------------
# Service Account
# -----------------------------------------------------------------------------
# Service account for Talos nodes. Required for:
# - GCE PD CSI driver to create/attach persistent disks
# - Cloud provider integration
# -----------------------------------------------------------------------------
resource "google_service_account" "talos_nodes" {
  account_id   = "${var.name_prefix}-talos-nodes"
  display_name = "Talos Kubernetes Nodes"
  description  = "Service account for Talos nodes - allows CSI driver disk operations"
}

# Grant compute storage admin for disk operations (create, delete, resize, snapshot)
resource "google_project_iam_member" "talos_nodes_compute_storage" {
  project = var.project_id
  role    = "roles/compute.storageAdmin"
  member  = "serviceAccount:${google_service_account.talos_nodes.email}"
}

# Grant compute instance admin for disk attach/detach on instances
# compute.storageAdmin covers disk-level ops but NOT compute.instances.attachDisk
resource "google_project_iam_member" "talos_nodes_compute_instance_admin" {
  project = var.project_id
  role    = "roles/compute.instanceAdmin.v1"
  member  = "serviceAccount:${google_service_account.talos_nodes.email}"
}

# Grant service account user - required for instanceAdmin to act as the SA
# when attaching disks to instances that use this service account
resource "google_project_iam_member" "talos_nodes_sa_user" {
  project = var.project_id
  role    = "roles/iam.serviceAccountUser"
  member  = "serviceAccount:${google_service_account.talos_nodes.email}"
}

# Grant compute viewer for metadata access
resource "google_project_iam_member" "talos_nodes_compute_viewer" {
  project = var.project_id
  role    = "roles/compute.viewer"
  member  = "serviceAccount:${google_service_account.talos_nodes.email}"
}

# Control Plane Node
resource "google_compute_instance" "control_plane" {
  name         = "${var.name_prefix}-cp-0"
  machine_type = var.machine_type
  zone         = var.zones[0]

  tags = ["talos-node", "talos-control-plane"]

  boot_disk {
    auto_delete = true
    initialize_params {
      image = var.boot_image
      size  = var.boot_disk_size
    }
  }

  network_interface {
    subnetwork = google_compute_subnetwork.talos.id
    # No access_config block = no external IP
  }

  # OS Login disabled - Talos has no SSH daemon, access via talosctl
  metadata = {
    enable-oslogin = "FALSE"
  }

  labels = {
    role = "control-plane"
  }

  # Service account for GCE PD CSI driver and cloud provider integration
  service_account {
    email  = google_service_account.talos_nodes.email
    scopes = ["cloud-platform"]
  }
}

# Worker Nodes
# count.index % length(zones) distributes workers across available zones
resource "google_compute_instance" "workers" {
  count        = var.worker_count
  name         = "${var.name_prefix}-worker-${count.index}"
  machine_type = var.machine_type
  zone         = var.zones[count.index % length(var.zones)]

  tags = ["talos-node", "talos-worker"]

  boot_disk {
    auto_delete = true
    initialize_params {
      image = var.boot_image
      size  = var.boot_disk_size
    }
  }

  network_interface {
    subnetwork = google_compute_subnetwork.talos.id
  }

  metadata = {
    enable-oslogin = "FALSE"
  }

  labels = {
    role = "worker"
  }

  # Service account for GCE PD CSI driver and cloud provider integration
  service_account {
    email  = google_service_account.talos_nodes.email
    scopes = ["cloud-platform"]
  }
}
