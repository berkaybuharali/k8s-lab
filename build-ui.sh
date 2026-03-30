#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
cd ui/frontend && npm run build
rm -rf ../../cli/pkg/ui/dist && mkdir -p ../../cli/pkg/ui/dist
cp -R dist/* ../../cli/pkg/ui/dist/
cd ../../cli && go build -o ../bin/k8s-lab .
echo "Build complete: bin/k8s-lab"
