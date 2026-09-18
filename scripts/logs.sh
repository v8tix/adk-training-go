#!/usr/bin/env bash
# Tails Redis's own container logs. Pass -f-style flags through, e.g.
# `./logs.sh --tail=100`.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

docker compose -f "$SCRIPT_DIR/../docker-compose.yml" logs -f "$@"
