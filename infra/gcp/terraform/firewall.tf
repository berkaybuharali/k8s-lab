# -----------------------------------------------------------------------------
# Firewall Rules
# -----------------------------------------------------------------------------
# VMs have no external IPs. Access is via IAP (Identity-Aware Proxy).
# IAP range 35.235.240.0/20 is Google's fixed range for IAP tunneling.
# -----------------------------------------------------------------------------

# Allow SSH, Talos API, and Kubernetes API via IAP tunnel (no external IP required)
resource "google_compute_firewall" "iap_ssh" {
  name    = "${var.network_name}-allow-iap-ssh"
  network = google_compute_network.talos.name

  allow {
    protocol = "tcp"
    ports    = ["22", "50000", "6443"]
  }

  # Google's IAP forwarding range - required for gcloud compute ssh --tunnel-through-iap and IAP tunnels
  source_ranges = ["35.235.240.0/20"]
  target_tags   = ["talos-node"]
}

# Allow all internal traffic between cluster nodes
resource "google_compute_firewall" "internal" {
  name    = "${var.network_name}-allow-internal"
  network = google_compute_network.talos.name

  allow {
    protocol = "tcp"
    ports    = ["0-65535"]
  }

  allow {
    protocol = "udp"
    ports    = ["0-65535"]
  }

  allow {
    protocol = "icmp"
  }

  source_ranges = [var.subnet_cidr]
  target_tags   = ["talos-node"]
}

# Talos API (50000) and Kubernetes API (6443) - internal only
resource "google_compute_firewall" "talos_api" {
  name    = "${var.network_name}-allow-talos-api"
  network = google_compute_network.talos.name

  allow {
    protocol = "tcp"
    ports    = ["50000", "6443"]
  }

  source_ranges = [var.subnet_cidr]
  target_tags   = ["talos-node"]
}
