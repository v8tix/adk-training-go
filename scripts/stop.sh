#!/usr/bin/env bash
# Stops Redis without removing the container or its data volume — start.sh
# resumes it later exactly where it left off.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

docker compose -f "$SCRIPT_DIR/../docker-compose.yml" stop
