# Velero Backup Hooks Implementation

## Overview

This project implements Velero backup hooks to ensure database consistency during backups. Without hooks, backing up stateful applications like Redis or PostgreSQL can result in data corruption due to in-flight writes or uncommitted transactions.

## Redis Implementation

Redis backup hooks are configured via pod annotations in `apps/redis.yaml`:

```yaml
annotations:
  # Pre-backup: Force Redis to persist in-memory data
  pre.hook.backup.velero.io/command: '["/usr/local/bin/redis-cli", "BGSAVE"]'
  pre.hook.backup.velero.io/container: redis
  pre.hook.backup.velero.io/timeout: 3m
  pre.hook.backup.velero.io/on-error: Fail

  # Post-backup: Verify save completed
  post.hook.backup.velero.io/command: '["/usr/local/bin/redis-cli", "LASTSAVE"]'
  post.hook.backup.velero.io/container: redis
```

### How It Works

1. **Pre-backup hook**: Runs `BGSAVE` command
   - Triggers background RDB snapshot to disk
   - Non-blocking (background save)
   - Ensures all in-memory keys are persisted
   - Fails backup if BGSAVE fails

2. **Volume snapshot**: Velero snapshots the PVC
   - Captures RDB file + AOF log
   - Both persistence mechanisms captured

3. **Post-backup hook**: Runs `LASTSAVE`
   - Returns timestamp of last successful save
   - Logged for verification

### Redis Persistence Configuration

Redis uses dual persistence (RDB + AOF) for maximum durability:

```yaml
args:
  # RDB: Point-in-time snapshots
  - --save "900 1"      # 15 min if 1 key changed
  - --save "300 10"     # 5 min if 10 keys changed
  - --save "60 10000"   # 1 min if 10k keys changed

  # AOF: Append-only log (synced every second)
  - --appendonly "yes"
  - --appendfsync everysec
```

## Backup Command Enhancements

### Timestamped Backups

All backups automatically include a timestamp suffix:

```bash
make backup gcp
# Creates: k8s-lab-backup-27012026-1430
```

Format: `<base-name>-ddmmyyyyhhmm` (UTC)

### Custom Backup Name

```bash
NAME=prod-backup make backup gcp
# Creates: prod-backup-27012026-1430
```

### Multiple Namespaces

```bash
NAMESPACES=app1,app2,app3 make backup gcp
# Backs up three namespaces with timestamp
```

### Combined

```bash
NAME=multi-app NAMESPACES=frontend,backend,cache make backup gcp
# Creates: multi-app-27012026-1430 (3 namespaces)
```

## Backup Flow

```
User: make backup gcp
  ↓
1. Velero discovers pods in namespace(s)
  ↓
2. Pre-backup hooks execute
   - Redis: BGSAVE (persist memory to disk)
  ↓
3. Velero uploads resource manifests to GCS
  ↓
4. CSI driver creates GCE PD snapshots
   - Redis data: dump.rdb + appendonly.aof
  ↓
5. Post-backup hooks execute
   - Redis: LASTSAVE (verify)
  ↓
6. Backup marked "Completed"
```

## Restore Flow

```
User: make restore gcp
  ↓
1. Velero fetches manifests from GCS
  ↓
2. Creates PVCs with snapshot references
  ↓
3. CSI driver provisions disks from snapshots
  ↓
4. Pods scheduled, PVCs bound
  ↓
5. Redis starts, loads dump.rdb
  ↓
6. Redis replays AOF log
  ↓
7. Data restored with zero loss
```

## Verification

### Check Backup Succeeded

```bash
make list-backups gcp
# Look for "Completed" status
```

### Check Hook Execution

```bash
velero backup logs k8s-lab-backup-27012026-1430
# Look for:
# time="..." level=info msg="running exec hook" hookCommand="[/usr/local/bin/redis-cli BGSAVE]"
```

### Verify Data After Restore

```bash
# After restore
kubectl exec -n application deploy/redis -- redis-cli GET user:1
# Should return data seeded before backup
```

## Future: PostgreSQL Hooks (Planned)

PostgreSQL will use one of two strategies:

### Option 1: CHECKPOINT (Simple)
```yaml
annotations:
  pre.hook.backup.velero.io/command: '["/usr/bin/psql", "-U", "postgres", "-c", "CHECKPOINT;"]'
```

Pros: Simple, fast
Cons: Relies on WAL replay (less safe)

### Option 2: pg_dump (Safest)
```yaml
annotations:
  pre.hook.backup.velero.io/command: '["/bin/bash", "-c", "pg_dumpall -U postgres > /backup/dump.sql"]'
```

Pros: Consistent SQL dump
Cons: Slower, requires backup volume

See `backup-hooks.yaml` for full examples.

## References

- [Velero Backup Hooks Documentation](https://velero.io/docs/main/backup-hooks/)
- [Redis Persistence Documentation](https://redis.io/docs/management/persistence/)
- Implementation: `apps/redis.yaml` (pod annotations)
- Backup logic: `scripts/lib/velero.sh` (timestamping, namespaces)
