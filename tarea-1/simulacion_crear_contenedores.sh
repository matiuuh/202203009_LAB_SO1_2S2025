#!/usr/bin/env bash
set -euo pipefail

CARNET="202203009"

rand_name() {
  uuidgen | tr -d '-' | cut -c1-8
}

COUNT="$(shuf -i 1-4 -n 1)"

echo "Creando ${COUNT} contenedores en: $(pwd)"
for ((i=1; i<=COUNT; i++)); do
  NAME="$(rand_name)"
  FILENAME="contenedor_${CARNET}_${NAME}.txt"
  printf '%s\n' "$FILENAME" > "$FILENAME"
  echo "Creado: $FILENAME"
done

echo "Listo."
