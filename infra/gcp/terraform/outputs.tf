# -----------------------------------------------------------------------------
# Outputs
# -----------------------------------------------------------------------------
# Values used by bootstrap scripts. Access via: terraform output -json
# -----------------------------------------------------------------------------

output "project_id" {
  description = "GCP project ID"
  value       = var.project_id
}

output "control_plane_name" {
  description = "Control plane instance name"
  value       = google_compute_instance.control_plane.name
}

output "control_plane_zone" {
  description = "Control plane zone"
  value       = google_compute_instance.control_plane.zone
}

output "control_plane_ip" {
  description = "Internal IP of control plane node"
  value       = google_compute_instance.control_plane.network_interface[0].network_ip
}

output "worker_names" {
  description = "Worker instance names"
  value       = google_compute_instance.workers[*].name
}

output "worker_zones" {
  description = "Worker instance zones"
  value       = google_compute_instance.workers[*].zone
}

output "worker_ips" {
  description = "Internal IPs of worker nodes"
  value       = google_compute_instance.workers[*].network_interface[0].network_ip
}

output "node_service_account_email" {
  description = "Service account email for Talos nodes (used by Velero for GCS access)"
  value       = google_service_account.talos_nodes.email
}

output "state_bucket" {
  description = "GCS bucket name for Terraform state and Velero backups"
  value       = var.state_bucket
}

output "artifact_registry" {
  description = "Artifact Registry URL for agent container images"
  value       = "${var.region}-docker.pkg.dev/${var.project_id}/${data.google_artifact_registry_repository.k8s_lab.repository_id}"
}
