#!/usr/bin/env bash
# Stops Redis and removes the container and its data volume — a genuine
# reset, not just a stop. Whatever module-13.5's own lab stored is gone
# after this; that's the point.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

docker compose -f "$SCRIPT_DIR/../docker-compose.yml" down -v
