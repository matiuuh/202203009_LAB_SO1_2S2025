#!/usr/bin/env bash
set -euo pipefail

# rangos para variables
RAM_MIN="${P2_RAM_MIN_MB:-256}"
RAM_MAX="${P2_RAM_MAX_MB:-1024}"
CPUW_MIN="${P2_CPUW_MIN:-1}"
CPUW_MAX="${P2_CPUW_MAX:-3}"

# === Config =====
TOTAL="${P2_TOTAL_SPAWN:-10}" 

# Reparto aleatorio entre HIGH y LOW si no viene por env
BATCH_HI="${P2_BATCH_HI:-$((RANDOM % (TOTAL+1)))}"
BATCH_LO="$((TOTAL - BATCH_HI))"
(( BATCH_HI < 0 )) && BATCH_HI=0
(( BATCH_LO < 0 )) && BATCH_LO=0

# Divide los "altos" entre CPU y RAM; si no viene por env, también aleatorio
if [[ -n "${P2_BATCH_HI_CPU:-}" && -n "${P2_BATCH_HI_RAM:-}" ]]; then
  BATCH_HI_CPU="$P2_BATCH_HI_CPU"
  BATCH_HI_RAM="$P2_BATCH_HI_RAM"
else
  BATCH_HI_CPU="$((RANDOM % (BATCH_HI+1)))"
  BATCH_HI_RAM="$((BATCH_HI - BATCH_HI_CPU))"
fi
(( BATCH_HI_CPU < 0 )) && BATCH_HI_CPU=0
(( BATCH_HI_RAM < 0 )) && BATCH_HI_RAM=0

# Imágenes a usar
IMAGE_HI_CPU="${P2_IMAGE_HI_CPU:-p2/high-cpu:1}"   # yes > /dev/null
IMAGE_HI_RAM="${P2_IMAGE_HI_RAM:-p2/high-ram:1}"   # stress-ng
IMAGE_LO="${P2_IMAGE_LO:-p2/low:1}"                # sleep

DOCKER_BIN="$(command -v docker || true)"
if [[ -z "$DOCKER_BIN" ]]; then
  echo "[cron_spawn] docker no encontrado en PATH" >&2
  exit 0
fi

ts="$(date +%Y%m%d%H%M%S)"
rand_range() { local min=$1 max=$2; echo $((RANDOM % (max - min + 1) + min)); }

# === Lanzamiento ===
# --- Altos CPU ---
if (( BATCH_HI_CPU > 0 )); then
  for i in $(seq 1 "$BATCH_HI_CPU"); do
    name="p2-hiCPU-${ts}-${i}"
    cpuw="$(rand_range "$CPUW_MIN" "$CPUW_MAX")"
    "$DOCKER_BIN" run -d --name "$name" \
      --label proyecto2=1 --label tier=high --label mode=cpu --label cpu_workers="$cpuw" \
      -e CPU_WORKERS="$cpuw" \
      "$IMAGE_HI_CPU" \
    && echo "[cron_spawn] high-CPU: $name (CPU_WORKERS=$cpuw)" || echo "[cron_spawn][ERR] $name"
  done
fi

# --- Altos RAM ---
if (( BATCH_HI_RAM > 0 )); then
  for i in $(seq 1 "$BATCH_HI_RAM"); do
    name="p2-hiRAM-${ts}-${i}"
    ram="$(rand_range "$RAM_MIN" "$RAM_MAX")"
    "$DOCKER_BIN" run -d --name "$name" \
      --label proyecto2=1 --label tier=high --label mode=ram --label ram_mb="$ram" \
      -e RAM_MB="$ram" -e VM_WORKERS=1 \
      "$IMAGE_HI_RAM" \
    && echo "[cron_spawn] high-RAM: $name (RAM_MB=${ram})" || echo "[cron_spawn][ERR] $name"
  done
fi

# --- Bajos ---
if (( BATCH_LO > 0 )); then
  for i in $(seq 1 "$BATCH_LO"); do
    name="p2-lo-${ts}-${i}"
    "$DOCKER_BIN" run -d --name "$name" \
      --label proyecto2=1 --label tier=low --label mode=low \
      "$IMAGE_LO" \
    && echo "[cron_spawn] low: $name" || echo "[cron_spawn][ERR] $name"
  done
fi