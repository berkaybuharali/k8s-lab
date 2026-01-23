# -----------------------------------------------------------------------------
# Provider Configuration
# -----------------------------------------------------------------------------
# Uses Application Default Credentials (ADC) for authentication.
# Run: gcloud auth application-default login
# -----------------------------------------------------------------------------

terraform {
  required_version = ">= 1.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}
