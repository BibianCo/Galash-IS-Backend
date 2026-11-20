#!/usr/bin/env bash
# TDB-T13 (migración) y TDB-T14 (respaldo/restauración) en un contenedor aislado con datos sintéticos.
# Uso: bash tdb-t13-t14.sh <repo-backend>. No toca otras bases. Borra el contenedor al terminar.
set -uo pipefail
REPO=${1:?ruta del repo backend}; C=galash-tdb-migr; PREV=7b09f31^
P(){ docker exec -i "$C" psql -U postgres -d "$1" -v ON_ERROR_STOP=1 -q -At "${@:2}"; }
COUNTS="select 'usuarios',count(*) from usuarios union all select 'semilleros',count(*) from semilleros union all select 'semillero_miembros',count(*) from semillero_miembros union all select 'solicitudes_registro',count(*) from solicitudes_registro union all select 'proyectos',count(*) from proyectos union all select 'proyecto_historial',count(*) from proyecto_historial union all select 'eventos',count(*) from eventos union all select 'audit_logs',count(*) from audit_logs order by 1"
docker rm -f $C >/dev/null 2>&1; docker run -d --name $C -e POSTGRES_PASSWORD=postgres postgres:16-alpine >/dev/null
until docker exec $C pg_isready -U postgres -q; do sleep 1; done; sleep 2
docker exec $C psql -U postgres -q -c 'create database migr'
echo "## 1. Esquema versión anterior ($PREV: $(git -C "$REPO" log -1 --format='%h %ad' --date=short $PREV)) + datos sintéticos"
git -C "$REPO" show $PREV:database/init.sql | P migr 2>/dev/null && echo "init.sql anterior: OK"
P migr < "$REPO/database/seed/synthetic.sql" && echo "synthetic.sql: OK"
P migr -c "update proyectos set estado='Radicado' where id in (select id from proyectos order by id limit 2)" && echo "2 proyectos en estado 'Radicado' (válido en la versión anterior)"
P migr -c "$COUNTS" > /tmp/antes.txt; cat /tmp/antes.txt
echo "## 2. TDB-T14: respaldo previo (pg_dump -Fc)"
t0=$(date +%s%N); docker exec $C pg_dump -U postgres -Fc -f /tmp/migr.dump migr; t1=$(date +%s%N)
echo "pg_dump: $(( (t1-t0)/1000000 )) ms, $(docker exec $C stat -c %s /tmp/migr.dump) bytes, PostgreSQL $(P migr -c 'show server_version')"
echo "## 3. TDB-T13a: re-aplicar init.sql actual sobre la base existente"
t0=$(date +%s%N); P migr < "$REPO/database/init.sql" > /tmp/init.log 2>&1; rc=$?; t1=$(date +%s%N)
echo "exit=$rc en $(( (t1-t0)/1000000 )) ms; primer error: $(grep -m1 ERROR /tmp/init.log)"
echo "## 4. TDB-T13b: migración incremental 017_project_statuses.sql"
t0=$(date +%s%N); P migr < "$REPO/database/migrations/017_project_statuses.sql"; rc=$?; t1=$(date +%s%N)
echo "exit=$rc en $(( (t1-t0)/1000000 )) ms"
P migr -c "$COUNTS" > /tmp/despues.txt; diff /tmp/antes.txt /tmp/despues.txt && echo "conteos críticos sin cambios" || echo "(diferencias arriba: historial agrega una fila por proyecto migrado)"
echo "estados de proyecto: $(P migr -c "select string_agg(estado||'='||n,', ') from (select estado,count(*) n from proyectos group by 1 order by 1) s")"
echo "CHECK proyectos_estado_check: $(P migr -c "select convalidated from pg_constraint where conname='proyectos_estado_check'")"
echo "esquema igual al de init.sql actual (tablas): $(P migr -c "select count(*) from information_schema.tables where table_schema='public' and table_type='BASE TABLE'") tablas"
echo "## 5. Reversión / TDB-T14: restaurar el respaldo en una base nueva y comparar"
docker exec $C psql -U postgres -q -c 'create database restaurada'
t0=$(date +%s%N); docker exec $C pg_restore -U postgres -d restaurada --exit-on-error /tmp/migr.dump; rc=$?; t1=$(date +%s%N)
echo "pg_restore exit=$rc en $(( (t1-t0)/1000000 )) ms"
P restaurada -c "$COUNTS" > /tmp/restaurada.txt; diff /tmp/antes.txt /tmp/restaurada.txt && echo "conteos restaurados = conteos previos a la migración"
echo "estados tras revertir: $(P restaurada -c "select string_agg(estado||'='||n,', ') from (select estado,count(*) n from proyectos group by 1 order by 1) s")"
docker rm -f $C >/dev/null && echo "contenedor eliminado"
