# Applications

Kubernetes manifests for lab applications.

## What Gets Deployed

| Application | Type | Replicas | Storage | Purpose |
|-------------|------|----------|---------|---------|
| Redis | Stateful | 1 | 1Gi PVC | In-memory data store with persistence |

## Structure

- Cloud-specific: `gcp/storageclass.yaml` (GCE PD CSI)
- Cloud-agnostic: `namespace.yaml`, `redis.yaml`

Redis PVC uses default StorageClass set by cloud-specific config.

## Adding New Clouds

Create `apps/<cloud>/storageclass.yaml` and implement cloud modules in `scripts/lib/<cloud>/` (infra.sh, tunnel.sh, csi.sh).
