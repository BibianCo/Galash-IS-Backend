-- TDB-T01: revisión del esquema real (database/init.sql) sobre PostgreSQL 16 efímero.
\echo '## Tablas base'
select count(*) from information_schema.tables where table_schema='public' and table_type='BASE TABLE';
\echo '## UNIQUE (sin PK)'
select tc.table_name, string_agg(kcu.column_name,',') from information_schema.table_constraints tc join information_schema.key_column_usage kcu using(constraint_name,table_schema) where tc.constraint_type='UNIQUE' and tc.table_schema='public' group by tc.constraint_name,tc.table_name order by 1;
\echo '## FKs y regla ON DELETE'
select conrelid::regclass, a.attname, confrelid::regclass, case confdeltype when 'a' then 'NO ACTION' when 'r' then 'RESTRICT' when 'c' then 'CASCADE' when 'n' then 'SET NULL' end from pg_constraint c join pg_attribute a on a.attrelid=c.conrelid and a.attnum=c.conkey[1] where contype='f' order by 1,2;
\echo '## FKs sin índice que las encabece'
select conrelid::regclass, a.attname from pg_constraint c join pg_attribute a on a.attrelid=c.conrelid and a.attnum=c.conkey[1] where contype='f' and not exists (select 1 from pg_index i where i.indrelid=c.conrelid and i.indkey[0]=c.conkey[1]) order by 1;
\echo '## CHECKs de tabla'
select conrelid::regclass, pg_get_constraintdef(oid) from pg_constraint where contype='c' and conrelid<>0 order by 1;
\echo '## Columnas de estado/rol/tipo sin CHECK'
select c.table_name||'.'||c.column_name||coalesce(' default '||c.column_default,'') from information_schema.columns c where table_schema='public' and column_name ~ '^(estado|rol|tipo|visibilidad|resultado|result|origin|tipo_dni|categoria|vinculacion)$' and not exists (select 1 from pg_constraint k where k.contype='c' and k.conrelid=(c.table_name)::regclass and pg_get_constraintdef(k.oid) like '%'||c.column_name||'%') order by 1;
\echo '## Tablas sin actualizado_en'
select string_agg(t.table_name,', ' order by 1) from information_schema.tables t where table_schema='public' and table_type='BASE TABLE' and not exists(select 1 from information_schema.columns c where c.table_name=t.table_name and c.column_name in ('actualizado_en','updated_at'));
\echo '## Triggers'
select event_object_table, trigger_name, action_timing, string_agg(event_manipulation,'/') from information_schema.triggers group by 1,2,3 order by 1;
\echo '## Sondas (dentro de transacción revertida, datos sintéticos)'
begin;
insert into usuarios(firebase_uid,nombre,apellido,email) values ('tdb-a','A','A','dup@example.test'),('tdb-b','B','B','DUP@example.test');
select 'emails duplicados aceptados', count(*) from usuarios where lower(email)='dup@example.test';
insert into usuarios(firebase_uid,nombre,apellido,email,rol,estado) values ('tdb-f','A','A','x@example.test','rol-inexistente','estado-inexistente');
select 'rol/estado libres aceptados', rol, estado from usuarios where firebase_uid='tdb-f';
rollback;
