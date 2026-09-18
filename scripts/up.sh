#!/usr/bin/env bash
# Starts Redis (module-13.5's own persistence lab needs a real, reachable
# instance — see the top-level README's Tooling section) in the background,
# building/pulling the image if it isn't already present.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

docker compose -f "$SCRIPT_DIR/../docker-compose.yml" up -d

echo "Redis is starting — check status with: docker compose -f docker-compose.yml ps"
echo "Connect from a Go program via REDIS_ADDR (default localhost:6379)."
