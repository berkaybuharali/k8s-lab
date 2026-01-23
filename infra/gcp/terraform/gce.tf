# -----------------------------------------------------------------------------
# GCE Compute Instances
# -----------------------------------------------------------------------------
# Control plane and worker nodes for the Kubernetes cluster.
# - No external IPs: access via IAP TCP tunnel (talosctl, not SSH)
# - Workers distributed across zones for availability
# - Boot image is Talos Linux (immutable OS for Kubernetes)
# -----------------------------------------------------------------------------

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
}
