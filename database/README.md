# PostgreSQL

Este directorio contiene `init.sql`, que es utilizado por el compose general
de la raíz para inicializar PostgreSQL la primera vez que se crea el volumen.

```bash
cd ..
docker compose up -d postgres
```

El esquema se aplica automáticamente al iniciar una base de datos nueva. Para
volver a ejecutarlo desde cero, elimina el volumen con `docker compose down -v`
y levanta el servicio de nuevo.

Las bases existentes no vuelven a ejecutar `init.sql` automáticamente. Para
habilitar las membresías del módulo de semilleros en una base ya creada,
ejecuta la migración desde la raíz del repositorio:

```bash
docker compose exec -T postgres psql -U "${POSTGRES_USER:-galash}" \
  -d "${POSTGRES_DB:-galash}" < database/migrations/002_seedbed_members.sql
```

Para habilitar las publicaciones del directorio de integrantes en una base ya
creada, ejecuta:

```bash
docker compose exec -T postgres psql -U "${POSTGRES_USER:-galash}" \
  -d "${POSTGRES_DB:-galash}" < database/migrations/004_member_publications.sql
```

Para guardar los datos básicos en la cuenta (código estudiantil, fecha de
nacimiento y teléfono) y dejar de copiarlos en las solicitudes, ejecuta:

```bash
docker compose exec -T postgres psql -U "${POSTGRES_USER:-galash}" \
  -d "${POSTGRES_DB:-galash}" < database/migrations/008_user_basic_profile.sql
```

Para habilitar los delegados de semillero en una base ya creada, ejecuta:

```bash
docker compose exec -T postgres psql -U "${POSTGRES_USER:-galash}" \
  -d "${POSTGRES_DB:-galash}" < database/migrations/011_seedbed_delegates.sql
```

## Datos sintéticos (desarrollo local)

`database/seed/seed.sh` carga datos ficticios en el PostgreSQL del compose:
14 usuarios (docentes, estudiantes, uno inactivo y un visitante), 3 semilleros
publicados y 1 borrador, integrantes, proyectos, actividades, solicitudes de
registro en todos los estados con su historial, eventos de agenda y reseñas
(4 publicadas por semillero publicado, más pendientes, una oculta y una rechazada). Es
idempotente: puede ejecutarse varias veces.

```bash
database/seed/seed.sh          # cargar
database/seed/seed.sh --reset  # borrar solo los datos sintéticos (ids 5eed…)
```

Los usuarios sintéticos no tienen cuenta Firebase y no pueden iniciar sesión.
La clave de seguimiento de la solicitud sintética N es `synthetic-token-N`.
Nunca ejecutar en una base con datos reales de producción.

### Carga UTF-8 desde Windows

Con PowerShell 7, ejecuta `./database/seed/seed.ps1`. El script transmite los
bytes del SQL sin recodificarlos y fija `PGCLIENTENCODING=UTF8`. En Bash,
`database/seed/seed.sh` conserva la redirección binaria y fija la misma codificación.
No uses `Get-Content ... | docker ...` para importar el SQL: la codificación de
salida de una consola puede sustituir tildes y eñes por signos de interrogación.

### Reparar la carga sintética anterior

En la base local de desarrollo, `./database/seed/seed.ps1 -RepairUtf8 -DryRun`
prueba la reparación en una transacción que termina en rollback. Para aplicarla:
`./database/seed/seed.ps1 -RepairUtf8` (Bash: `database/seed/seed.sh --repair-utf8`).
`repair_utf8.sql` restaura campos públicos de usuarios, perfiles, semilleros,
proyectos, actividades y eventos sintéticos a partir de `synthetic.sql`.
Solo modifica UUID sintéticos concretos cuyo valor coincide exactamente con
la pérdida de bytes esperada; conserva otras ediciones y los registros reales.
No reinicia la base ni cambia asociaciones, permisos o historial. Se puede repetir.
