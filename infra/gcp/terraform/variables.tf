# -----------------------------------------------------------------------------
# Input Variables
# -----------------------------------------------------------------------------
# All values should be set in terraform.tfvars
# -----------------------------------------------------------------------------

variable "project_id" {
  description = "GCP Project ID"
  type        = string
}

variable "name_prefix" {
  description = "Prefix for all resource names (use your initials to avoid conflicts)"
  type        = string
}

variable "worker_count" {
  description = "Number of worker nodes"
  type        = number
  default     = 2
}

variable "region" {
  description = "GCP Region for resources"
  type        = string
}

variable "zones" {
  description = "Availability zones for distributing nodes"
  type        = list(string)
}

variable "machine_type" {
  description = "GCE instance machine type"
  type        = string
  default     = "e2-small"
}

variable "boot_disk_size" {
  description = "Boot disk size in GB"
  type        = number
  default     = 20
}

variable "network_name" {
  description = "VPC network name"
  type        = string
  default     = "talos-vpc"
}

variable "subnet_cidr" {
  description = "Subnet CIDR range for cluster nodes"
  type        = string
  default     = "10.0.0.0/24"
}

variable "boot_image" {
  description = "Talos Linux boot disk image"
  type        = string
}

variable "state_bucket" {
  description = "GCS bucket name for Terraform state and Velero backups (must match backend.tf)"
  type        = string
}
