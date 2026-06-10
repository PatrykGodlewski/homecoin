#!/bin/sh
set -eu

ROOT="$(CDPATH= cd "$(dirname "$0")/../.." && pwd)"
SRC="${ROOT}/migrations"
DST="${ROOT}/internal/infrastructure/postgres/migrations"

mkdir -p "${DST}"
cp -f "${SRC}"/*.sql "${DST}/"
echo "Synced migrations -> internal/infrastructure/postgres/migrations"
