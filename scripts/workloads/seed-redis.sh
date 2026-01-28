#!/usr/bin/env bash
# -----------------------------------------------------------------------------
# Seed Redis with Test Data
# -----------------------------------------------------------------------------
# Populates Redis with sample data for testing backup/restore with Velero.
#
# Usage: ./seed-redis.sh <cloud>
# Run via: make seed <cloud>
# -----------------------------------------------------------------------------

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../lib/common.sh"

seed_data() {
    log_info "Checking Redis is ready"
    kubectl wait --for=condition=ready pod -l app=redis -n application --timeout=60s

    log_info "Inserting test data"
    kubectl exec deploy/redis -n application -- redis-cli SET user:1 '{"name":"Alice","email":"alice@example.com"}'
    kubectl exec deploy/redis -n application -- redis-cli SET user:2 '{"name":"Bob","email":"bob@example.com"}'
    kubectl exec deploy/redis -n application -- redis-cli SET user:3 '{"name":"Charlie","email":"charlie@example.com"}'
    kubectl exec deploy/redis -n application -- redis-cli SET counter:visits 1000
    kubectl exec deploy/redis -n application -- redis-cli SET config:app:version "1.0.0"
    kubectl exec deploy/redis -n application -- redis-cli LPUSH queue:tasks "task1" "task2" "task3"

    log_info "Verifying seeded data"
    kubectl exec deploy/redis -n application -- redis-cli KEYS '*'
}

main() {
    local cloud=$1

    validate_cloud "$cloud"
    source_cloud_modules "$cloud"

    log_step "Seeding Redis with test data for Velero testing"
    setup_error_handling
    k8s_connect
    seed_data
    log_info "Redis seeded successfully"
}

main "$@"
