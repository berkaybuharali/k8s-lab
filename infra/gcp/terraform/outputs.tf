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
