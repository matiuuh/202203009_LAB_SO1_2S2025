#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SPAWN="${SCRIPT_DIR}/cron_spawn_containers.sh"

# Si no hay crontab previo, no falla
if crontab -l >/dev/null 2>&1; then
  crontab -l | grep -v "$SPAWN" | crontab -
fi

echo "[remove_cron] Eliminada entrada del cron para ${SPAWN}"
