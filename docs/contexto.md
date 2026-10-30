# Active Context

## System

GALASH-UPTC web system for academic management, research, and communication.

## Implemented state

- Versioned OpenCode configuration in `opencode.json`.
- Shared agent rules in `AGENTS.md`.
- Selected skills managed by the repository in `.opencode/skills/`.
- Append-only technical log in `docs/bitacora.ndjson`.
- Append-only testing ledger in `docs/testing/events.ndjson`.
- Vanilla JavaScript testing dashboard in `docs/testing/dashboard/`.
- Private requirements excluded from version control.

## Conventions

- User defines scope and requirements for each task.
- Keep changes small and limited to requested scope.
- Backend validates authorization.
- Critical actions produce audit records.
- Personal data follows least privilege.
- Completed work records files and tests in the log.
- Servicio `Auth` en Go valida ID tokens de Firebase y sincroniza usuarios con PostgreSQL.
- El backend Go está organizado como un monolito con arquitectura hexagonal:
  entidades y errores en `internal/domain`, puertos en `internal/ports`, casos
  de uso en `internal/application`, adaptadores en `internal/adapters` y
  composición en `cmd/api`, con un único `main.go` y `Dockerfile` en la raíz.
- PostgreSQL y Auth se levantan juntos mediante el único `docker-compose.yml` de
  la raíz; PostgreSQL usa `database/init.sql` para inicializar el esquema.
- El directorio de integrantes permite búsqueda pública por nombre/correo y búsqueda administrativa con datos sensibles; la asignación de roles exige sesión de `Admin` y usa el Firebase UID del integrante seleccionado.
- El módulo de semilleros implementa CRUD, publicación/ocultamiento/inactivación y membresías históricas mediante `/seedbeds`, con autorización para `Docente` responsable o `Admin` y validación de docentes activos.
- `/seedbeds/{id}/space` devuelve la vista compuesta del semillero con responsable, actividades y proyectos; la información privada requiere autorización del docente responsable o `Admin`.
- `POST /auth/register` exige `fecha_nacimiento` y `telefono` y acepta `codigo_estudiantil` opcional; se guardan en `usuarios` (migración `008_user_basic_profile.sql`) y `GET /users/me` los devuelve.
- `POST /registration-requests` exige sesión; nombre, documento, código, teléfono y fecha de nacimiento se leen de `usuarios` por `usuario_id` y no se copian en `solicitudes_registro` (esas columnas solo conservan solicitudes anónimas antiguas).
- El módulo 3 implementa las cinco rutas de solicitudes de registro: radicación con token opaco, consulta propia, listado y detalle administrativo, y actualización auditada de estados; la aprobación asigna el rol `estudiante` a la cuenta asociada por sesión o documento.
- Las consultas de semilleros inexistentes, los duplicados de nombre y los retiros de integrantes no activos se traducen a errores HTTP controlados; el listado de integrantes no continúa después de responder una sesión anónima requerida.
- El módulo 4 (actividades, proyectos, propuestas y reseñas) responde en camelCase; las propuestas inician siempre `Radicada` y solo `Docente`/`Admin` cambian estado u observaciones; cada edición de proyecto o propuesta queda en historial con autor y fecha, y la conversión enlaza propuesta (`projectId`) y proyecto conservando el historial. Los `PATCH` del módulo son parciales; las reseñas nuevas notifican a cada `Admin` activo y su listado administrativo muestra por defecto las pendientes. Una actividad creada sin estado lo deriva de su fecha.
- El adaptador HTTP acepta preflight CORS desde interfaces locales (`localhost`, `127.0.0.1` y `::1`) y prioriza `Authorization` sobre una cookie de sesión antigua.
- El módulo 5 implementa el directorio de integrantes: listado público filtrable, creación y edición administrativa, detalle con campos internos para Admin o el propio integrante, cambio de estado sin borrar relaciones y publicaciones visibles.
- `GET /admin/members` (contract M11) requires `q`, includes inactive accounts and returns `{users: [...]}`; the legacy alias `/find/user` no longer exists. `PUT /users/{userId}/roles` accepts `docente` (contract name) and `profesor` (legacy).
- Seedbeds have `topics` (max 4 `{title, phrase}`, JSONB column `temas`, migration `007_seedbed_topics.sql`); omitted on `PATCH` they keep their value.
- `database/seed/seed.sh [--reset]` loads or deletes idempotent synthetic development data (ids `5eed…`).

## Record

- Detailed history: `docs/bitacora.ndjson`.
- This file describes current state, not history.
- Contains no private requirements, credentials, or personal data.
