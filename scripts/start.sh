#!/usr/bin/env bash
# Resumes a previously-created but stopped Redis container. Use up.sh
# instead if it's never been created yet.
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

docker compose -f "$SCRIPT_DIR/../docker-compose.yml" start
