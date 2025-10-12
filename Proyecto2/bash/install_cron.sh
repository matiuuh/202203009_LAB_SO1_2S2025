#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SPAWN="${SCRIPT_DIR}/cron_spawn_containers.sh"
LOGFILE="/var/log/cron_spawn.log"

chmod +x "$SPAWN"
touch "$LOGFILE"
chmod 0644 "$LOGFILE"

# Construye el nuevo crontab en un archivo temporal, robusto aunque no exista crontab previo
TMP="$(mktemp)"
# Si no hay crontab, esta línea no falla por el '|| true'
crontab -l 2>/dev/null | grep -v -F "$SPAWN" >"$TMP" || true
echo "* * * * * ${SPAWN} >> ${LOGFILE} 2>&1" >>"$TMP"

crontab "$TMP"
rm -f "$TMP"

echo "[install_cron] OK: * * * * * ${SPAWN} >> ${LOGFILE} 2>&1"
