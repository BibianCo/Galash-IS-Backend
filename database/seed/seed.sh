#!/usr/bin/env sh
# Loads (or with --reset, deletes) the synthetic development data in the compose PostgreSQL.
# Usage from any directory: database/seed/seed.sh [--reset]
set -eu

dir=$(cd "$(dirname "$0")" && pwd)
file="$dir/synthetic.sql"
[ "${1:-}" = "--reset" ] && file="$dir/synthetic_reset.sql"

cd "$dir/../.."
docker compose exec -T postgres psql -v ON_ERROR_STOP=1 -q \
  -U "${POSTGRES_USER:-galash}" -d "${POSTGRES_DB:-galash}" < "$file"
echo "OK: $(basename "$file")"
