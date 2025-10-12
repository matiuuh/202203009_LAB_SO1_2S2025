#!/usr/bin/env bash
set -euo pipefail

# Ubicación del directorio con tus .c y el Makefile
MODDIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../modulos-kernel" && pwd)"
KVER="$(uname -r)"

echo "[load_modules] Kernel: $KVER"
if [[ ! -d "/lib/modules/$KVER/build" ]]; then
  echo "[load_modules] ERROR: No están instalados los headers del kernel ($KVER)."
  echo "  -> sudo apt install -y linux-headers-$(uname -r)"
  exit 2
fi

cd "$MODDIR"

echo "[load_modules] Compilando módulos..."
# Usa gcc-12 si existe; si no, usa el gcc por defecto
CCBIN="$(command -v gcc-12 || command -v gcc)"
make CC="$CCBIN"

# Nombres de tus módulos (.ko)
SYSKO="./sysinfo_so1_202203009.ko"
CONTKO="./continfo_so1_202203009.ko"

# Descargar si ya están
for MOD in sysinfo_so1_202203009 continfo_so1_202203009; do
  if lsmod | grep -q "^${MOD}\b"; then
    echo "[load_modules] rmmod $MOD"
    sudo rmmod "$MOD"
  fi
done

# Cargar en orden (da igual el orden, pero así queda claro)
echo "[load_modules] insmod $SYSKO"
sudo insmod "$SYSKO"

echo "[load_modules] insmod $CONTKO"
sudo insmod "$CONTKO"

# Verificación de /proc (hasta 10 intentos rápidos)
check_proc() {
  local p="$1"
  for i in {1..10}; do
    [[ -r "$p" ]] && return 0
    sleep 0.2
  done
  return 1
}

SYS_PROC="/proc/sysinfo_so1_202203009"
CONT_PROC="/proc/continfo_so1_202203009"

if check_proc "$SYS_PROC" && check_proc "$CONT_PROC"; then
  echo "[load_modules] OK: $SYS_PROC y $CONT_PROC están disponibles."
else
  echo "[load_modules] WARNING: no se ven las entradas /proc esperadas."
  echo "  SYS:  $SYS_PROC"
  echo "  CONT: $CONT_PROC"
  exit 3
fi
