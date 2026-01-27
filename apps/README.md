# Applications

Kubernetes manifests for lab applications.

## What Gets Deployed

| Application | Type | Replicas | Storage | Purpose |
|-------------|------|----------|---------|---------|
| NGINX | Stateless | 2 | None | Web server, spread across zones |
| Redis | Stateful | 1 | 1Gi PVC | In-memory data store with persistence |

## Architecture

```
Namespace: application
├── NGINX Deployment (2 replicas)
│   ├── Pod in Zone A
│   ├── Pod in Zone B
│   └── ClusterIP Service (:80)
└── Redis Deployment (1 replica)
    ├── Pod with PVC mount
    ├── PersistentVolumeClaim (1Gi)
    └── ClusterIP Service (:6379)
            │
            ▼
    Cloud Persistent Disk
    (dynamically provisioned)
```

## Structure

```
apps/
├── gcp/
│   └── storageclass.yaml   # GCP Persistent Disk StorageClass
├── namespace.yaml          # Cloud-agnostic
├── nginx.yaml              # Cloud-agnostic
├── redis.yaml              # Cloud-agnostic
└── README.md
```

## Cloud-Specific vs Cloud-Agnostic

| Manifest | Type | Description |
|----------|------|-------------|
| `gcp/storageclass.yaml` | Cloud-specific | GCE PD CSI driver StorageClass |
| `namespace.yaml` | Cloud-agnostic | Application namespace |
| `nginx.yaml` | Cloud-agnostic | NGINX deployment and service |
| `redis.yaml` | Cloud-agnostic | Redis deployment, PVC, and service |

The PersistentVolumeClaim in `redis.yaml` uses the default StorageClass, which is set by the cloud-specific `storageclass.yaml`.

## Adding New Clouds

To add support for a new cloud provider:

1. Create a directory: `apps/<cloud>/`
2. Add `storageclass.yaml` with the cloud's CSI driver
3. Add CSI driver installation to `scripts/lib/<cloud>/csi.sh`
4. Update `scripts/apply.sh` and `scripts/destroy.sh` with cloud case

## Accessing Applications

See [scripts/README.md](../scripts/README.md#accessing-the-cluster) for cluster access instructions.
