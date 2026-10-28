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

## Datos sintéticos (desarrollo local)

`database/seed/seed.sh` carga datos ficticios en el PostgreSQL del compose:
14 usuarios (docentes, estudiantes, uno inactivo y un visitante), 3 semilleros
publicados y 1 borrador, integrantes, proyectos, actividades, solicitudes de
registro en todos los estados con su historial, y eventos de agenda. Es
idempotente: puede ejecutarse varias veces.

```bash
database/seed/seed.sh          # cargar
database/seed/seed.sh --reset  # borrar solo los datos sintéticos (ids 5eed…)
```

Los usuarios sintéticos no tienen cuenta Firebase y no pueden iniciar sesión.
La clave de seguimiento de la solicitud sintética N es `synthetic-token-N`.
Nunca ejecutar en una base con datos reales de producción.
