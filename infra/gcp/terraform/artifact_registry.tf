# Artifact Registry for agent container images
# The repository is created outside Terraform (one-time setup) and referenced here
# This allows it to persist across daily infrastructure destroy/create cycles

data "google_artifact_registry_repository" "k8s_lab" {
  project       = var.project_id
  location      = var.region
  repository_id = "k8s-lab"
}

# Grant VM service account permission to pull images
resource "google_artifact_registry_repository_iam_member" "nodes_reader" {
  project    = var.project_id
  location   = var.region
  repository = data.google_artifact_registry_repository.k8s_lab.name
  role       = "roles/artifactregistry.reader"
  member     = "serviceAccount:${google_service_account.talos_nodes.email}"
}
