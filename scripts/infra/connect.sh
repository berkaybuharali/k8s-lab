#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# Connect Instructions
# -----------------------------------------------------------------------------
# Prints the commands needed to connect to the cluster from your local machine.
#
# After 'make deploy' and 'make apply', use this to get the tunnel command
# with all values pre-filled from your Terraform state.
#
# Usage: make connect gcp
# -----------------------------------------------------------------------------

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"
TF_DIR="${REPO_ROOT}/infra/gcp/terraform"
CONFIGS_DIR="${REPO_ROOT}/configs"

CLOUD=${1:-}
if [[ -z "$CLOUD" || "$CLOUD" != "gcp" ]]; then
    echo "Usage: $0 gcp"
    exit 1
fi

cd "$TF_DIR"
if ! terraform state list &>/dev/null; then
    echo "No cluster found. Run 'make deploy gcp' first."
    exit 1
fi

PROJECT_ID=$(terraform output -raw project_id)
CP_NAME=$(terraform output -raw control_plane_name)
CP_ZONE=$(terraform output -raw control_plane_zone)

cat <<EOF

CONNECT TO CLUSTER
==================

1. Start IAP tunnel (keep this terminal open):

   gcloud compute start-iap-tunnel ${CP_NAME} 6443 \\
     --local-host-port=localhost:6443 \\
     --zone=${CP_ZONE} \\
     --project=${PROJECT_ID}

2. In another terminal, use kubectl:

   export KUBECONFIG=${CONFIGS_DIR}/kubeconfig
   kubectl get nodes
   kubectl get pods -n application

3. Test applications:

   kubectl port-forward svc/nginx -n application 8080:80 &
   curl http://localhost:8080

   kubectl exec -it deploy/redis -n application -- redis-cli ping

EOF
