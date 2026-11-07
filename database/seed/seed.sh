#!/usr/bin/env sh
# Loads (or with --reset, deletes) the synthetic development data in the compose PostgreSQL.
# --real / --real-reset load / delete the real GALASH data (SGI members, group projects, seedbeds); those files are not in git.
# Usage from any directory: database/seed/seed.sh [--reset | --repair-utf8 | --real | --real-reset]
set -eu

dir=$(cd "$(dirname "$0")" && pwd)
case "${1:-}" in
  "") set -- "$dir/synthetic.sql" ;;
  --reset) set -- "$dir/synthetic_reset.sql" ;;
  --repair-utf8) set -- "$dir/repair_utf8.sql" ;;
  --real) set -- "$dir/sgi_uptc_integrantes.sql" "$dir/proyectos_grupo.sql" "$dir/semilleros_galash.sql" ;;
  --real-reset) set -- "$dir/semilleros_galash_reset.sql" "$dir/proyectos_grupo_reset.sql" "$dir/sgi_uptc_integrantes_reset.sql" ;;
  *) echo 'Usage: seed.sh [--reset | --repair-utf8 | --real | --real-reset]' >&2; exit 1 ;;
esac

cd "$dir/../.."
for file; do
  docker compose exec -T -e PGCLIENTENCODING=UTF8 postgres psql -X -v ON_ERROR_STOP=1 -q \
    -U "${POSTGRES_USER:-galash}" -d "${POSTGRES_DB:-galash}" < "$file"
  echo "OK: $(basename "$file")"
done
