#!/usr/bin/env sh
# Loads (or with --reset, deletes) the synthetic development data in the compose PostgreSQL.
# Usage from any directory: database/seed/seed.sh [--reset | --repair-utf8]
set -eu

dir=$(cd "$(dirname "$0")" && pwd)
file="$dir/synthetic.sql"
case "${1:-}" in
  "") ;;
  --reset) file="$dir/synthetic_reset.sql" ;;
  --repair-utf8) file="$dir/repair_utf8.sql" ;;
  *) echo 'Usage: seed.sh [--reset | --repair-utf8]' >&2; exit 1 ;;
esac

cd "$dir/../.."
docker compose exec -T -e PGCLIENTENCODING=UTF8 postgres psql -X -v ON_ERROR_STOP=1 -q \
  -U "${POSTGRES_USER:-galash}" -d "${POSTGRES_DB:-galash}" < "$file"
echo "OK: $(basename "$file")"
