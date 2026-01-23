# -----------------------------------------------------------------------------
# VPC Network
# -----------------------------------------------------------------------------
# Custom VPC with manual subnet creation for full control over IP ranges.
# auto_create_subnetworks=false prevents default regional subnets.
# -----------------------------------------------------------------------------

resource "google_compute_network" "talos" {
  name                    = var.network_name
  auto_create_subnetworks = false
}

resource "google_compute_subnetwork" "talos" {
  name          = "${var.network_name}-subnet"
  ip_cidr_range = var.subnet_cidr
  region        = var.region
  network       = google_compute_network.talos.id

  # VPC Flow Logs required by org policy
  log_config {
    aggregation_interval = "INTERVAL_5_SEC"
    flow_sampling        = 0.5
    metadata             = "INCLUDE_ALL_METADATA"
  }
}

# -----------------------------------------------------------------------------
# Cloud NAT
# -----------------------------------------------------------------------------
# Allows VMs without external IPs to access the internet.
# Required for:
# - Talos discovery service (discovery.talos.dev)
# - Pulling container images (gcr.io, registry.k8s.io)
# -----------------------------------------------------------------------------

resource "google_compute_router" "talos" {
  name    = "${var.network_name}-router"
  region  = var.region
  network = google_compute_network.talos.id
}

resource "google_compute_router_nat" "talos" {
  name                               = "${var.network_name}-nat"
  router                             = google_compute_router.talos.name
  region                             = var.region
  nat_ip_allocate_option             = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "ALL_SUBNETWORKS_ALL_IP_RANGES"

  log_config {
    enable = true
    filter = "ERRORS_ONLY"
  }
}
