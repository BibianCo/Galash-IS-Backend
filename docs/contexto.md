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
- `PUT /users/me/photo` guarda en `usuarios.foto_url` (migración `015_user_profile_photo.sql`) el enlace http(s) de la imagen de perfil del usuario de la sesión; `GET /users/me` lo expone como `photo_url`. Solo se guarda la URL; el archivo vive fuera del backend.
- Integrantes del SGI UPTC importados con `database/seed/sgi_uptc_integrantes.sql` quedan como cuentas `firebase_uid LIKE 'sgi-uptc:%'` sin login. En el primer ingreso, `GET /auth/register/matches` las ofrece si el correo verificado `primernombre.apellido[NN]@uptc.edu.co` coincide con el nombre, y `POST /auth/register` con `claim_id` las vincula conservando rol y perfil.
- `proyectos.semillero_id` admite NULL (migración `016_project_group_seedbed.sql`) para los proyectos del grupo importados del SGI UPTC y GrupLAC (`database/seed/proyectos_grupo.sql`, fuera de git; responsables e integrantes son cuentas importadas de `sgi_uptc_integrantes.sql`). Se devuelven con `seedbedId` vacío y solo `Admin` los gestiona. TODO: asignar a qué semillero pertenece cada proyecto.
- Las respuestas de proyectos incluyen `responsibles` y `members` (`[{id,name}]`) resueltos desde `usuarios` en la misma consulta, aunque el perfil esté oculto del directorio público.
- `POST /registration-requests` exige sesión; nombre, documento, código, teléfono y fecha de nacimiento se leen de `usuarios` por `usuario_id` y no se copian en `solicitudes_registro` (esas columnas solo conservan solicitudes anónimas antiguas).
- El módulo 3 implementa las cinco rutas de solicitudes de registro: radicación con token opaco, consulta propia, listado y detalle administrativo, y actualización auditada de estados; la aprobación asigna el rol `estudiante` a la cuenta asociada por sesión o documento y la inscribe como integrante activo del semillero solicitado, en una sola transacción.
- Director (`lider_id`/`responsibleTeacherId`) y hasta 2 delegados por semillero (`semillero_miembros.delegado`) son relaciones, no roles. Director, delegados y `Admin` gestionan solicitudes (que exigen `seedbedId`; la primera decisión gana), actividades y eventos con `seedbedId`; solo director o `Admin` asignan delegados con `PUT /seedbeds/{id}/delegates`. `GET /users/me` expone `managed_seedbed_ids`. Proyectos y edición del semillero siguen siendo solo del director o `Admin`.
- Las consultas de semilleros inexistentes, los duplicados de nombre y los retiros de integrantes no activos se traducen a errores HTTP controlados; el listado de integrantes no continúa después de responder una sesión anónima requerida.
- El módulo 4 (actividades, proyectos, propuestas y reseñas) responde en camelCase; las propuestas inician siempre `Radicada` y solo `Docente`/`Admin` cambian estado u observaciones; cada edición de proyecto o propuesta queda en historial con autor y fecha, y la conversión enlaza propuesta (`projectId`) y proyecto conservando el historial. Los `PATCH` del módulo son parciales; las reseñas nuevas notifican a cada `Admin` activo y su listado administrativo muestra por defecto las pendientes. Una actividad creada sin estado lo deriva de su fecha.
- El adaptador HTTP acepta preflight CORS desde interfaces locales (`localhost`, `127.0.0.1` y `::1`) y prioriza `Authorization` sobre una cookie de sesión antigua.
- El módulo 5 implementa el directorio de integrantes: listado público filtrable, creación y edición administrativa, detalle con campos internos para Admin o el propio integrante, cambio de estado sin borrar relaciones y publicaciones visibles.
- `GET /admin/members` (contract M11) requires `q`, includes inactive accounts and returns `{users: [...]}`; the legacy alias `/find/user` no longer exists. `PUT /users/{userId}/roles` accepts `docente` (contract name) and `profesor` (legacy).
- Seedbeds have `topics` (max 4 `{title, phrase}`, JSONB column `temas`, migration `007_seedbed_topics.sql`); omitted on `PATCH` they keep their value.
- `database/seed/seed.sh [--reset]` loads or deletes idempotent synthetic development data (ids `5eed…`).
- El módulo 8 implementa las siete rutas de comunicados: borradores, publicación,
  adjuntos referenciados PDF/Word/Excel, confirmación idempotente de lectura y
  estado de lectura por docente. `GET /announcements` muestra publicados y los
  borradores propios; Admin puede ver todos los borradores. Docente administra
  sus propios comunicados y Admin puede administrar todos; solo usuarios activos
  con esos roles acceden.
- El módulo 9 implementa el cronograma interno con cinco rutas para consultar,
  crear, detallar, actualizar y cancelar lógicamente reuniones, tutorías e hitos
  por semillero y participantes; conserva historial y restringe la visibilidad
  de estudiantes a sus semilleros. Solo el director, un delegado activo del
  semillero o Admin crean, actualizan y cancelan; los solapes de horario de un
   participante se rechazan al crear y al actualizar.
- El módulo 10 implementa los cuatro endpoints de enlaces oficiales: consulta pública filtrable por categoría y orden, creación y edición exclusiva de Admin con validación HTTP/HTTPS, cambios de estado sin borrado y registro PostgreSQL de cada alta o modificación.
- En `GET /links`, visitantes y usuarios no administradores solo reciben enlaces activos y visibles; un `Admin` activo autenticado puede consultar enlaces en cualquier estado.
- El módulo 12 implementa `GET /audit-logs`, restringido a `Admin`, con filtros por actor, acción, entidad y fechas RFC3339; la tabla `audit_logs` es append-only mediante trigger y las escrituras críticas, incluidas las de `/admin/content/*`, se registran internamente.
- El módulo 13 implementa carga multipart y descarga autorizada de archivos asociados a comunicados, evidencias, propuestas y eventos; conserva el contenido en PostgreSQL, limita a 10 MB y evita URLs públicas predecibles. Valida extensiones con una tabla propia (independiente del host) y solo acepta el tipo declarado para contenedores Office; un ID mal formado responde 404. Carga y descarga aplican la regla de lectura de la entidad asociada: Admin; comunicados y eventos publicados o propios; actividades, evidencias y proyectos existentes; propuestas solo para su autor, docentes y Admin. Cargar a una entidad inexistente o inaccesible responde 403.

## Record

- Detailed history: `docs/bitacora.ndjson`.
- This file describes current state, not history.
- Contains no private requirements, credentials, or personal data.

- PostgreSQL local: aplicada la migración existente 008_user_basic_profile.sql el 2026-11-04; las columnas básicas de usuarios ya corresponden al backend desplegado y la consulta de perfiles funciona.

- Local synthetic public text was repaired from canonical synthetic.sql after a prior lossy import replaced UTF-8 bytes with question marks. Guarded repair_utf8.sql only updates exact known damaged values at fixed synthetic UUIDs, preserving real records and other edits. seed.ps1 (PowerShell 7) streams raw UTF-8 and supports -RepairUtf8/-DryRun; seed.sh enforces UTF8 and supports --repair-utf8. Applied and verified rollback, idempotency, preserved fields and accented names/descriptions through API and Angular. No schema/API change. Existing frontend/backend M2 contract divergence (delegates) is recorded pending synchronization.

- Registration requests: tituloMaximoNivelEstudio is optional in frontend and backend validation; omitted/empty values accepted. M3 contract clause and servicios.md updated on both repos. Other required fields unchanged. Existing unrelated contract-copy drift remains pending.

- M11 admin account directory now accepts omitted/blank q and lists all stored accounts, including inactive ones, without the filtered-search cap. Admin session remains mandatory; public search behavior unchanged. Auth tests and read-only real PostgreSQL count comparison pass. Servicios handoff updated; pre-existing unrelated contract drift remains pending.

- Local galash-auth container rebuilt/restarted for optional admin q. Live HTTP GET /admin/members returns200 and every stored account, verified using a disposable synthetic account/session that was cleaned up. Frontend servicios.md handoff applied.

- PR preparation integrates origin/main through b4a0ad2 and retains the branch fixes. Full Go suite passes. Both full shared-contract copies are now synchronized, superseding earlier drift-pending notes.

- Local development database loaded using the user-supplied real-data package:170imported accounts,22group projects and5published real seedbeds. Existing volume/data preserved,15migrations applied, auth rebuilt, HTTP checks pass. Private SQL remains gitignored and no personal rows or screenshots are recorded. This is local environment state only; no API/contract change.

- Autogestión: Mi perfil edita área, formación, ORCID, CvLAC y visibilidad; guarda URL de foto. Backend autoriza solo perfil propio, sin identidad, roles ni semilleros; Admin conserva administración. Requiere desplegar ambos cambios juntos.

- Corrección por solicitud del usuario: se retiró la autogestión del perfil investigador; POST/PATCH /members regresan a permisos Admin. Mi perfil conserva información de solo lectura y edición de URL de foto únicamente. Sustituye el estado de autogestión descrito anteriormente.

- Datos locales: retirados los 4 semilleros y 12 proyectos sintéticos; permanecen 5 semilleros reales y 22 proyectos reales. Solicitud y evento manuales conservados y reasignados a Complejidad Algorítmica y Programación Competitiva. Cuentas sin eliminar.

- Proyectos: solo Activo y Finalizado en interfaz, backend y CHECK de PostgreSQL. Proyectos nuevos y conversiones desde propuestas se crean Activo; migración 017 conserva datos e historial. Base local mantiene 22 proyectos reales finalizados. Se entregó servicios.md actualizado al frontend.

- Comunicados: subida real de adjuntos PDF/Word/Excel desde el equipo (10 MiB por archivo, 10 adjuntos); guarda borrador antes de POST /files y persiste cada referencia con PATCH. Descarga desde detalle por GET /files/{id} mediante Blob y sesión, con estado/error. Enlaces externos previos conservados. Sin cambios de backend operativo; contrato y servicios.md sincronizados con el frontend.

- Admin → Contenido: gestión de secciones institucionales y enlaces con lista/editor/vista previa. Crea/edita borradores y publica; una sección publicada se actualiza al guardar. Enlaces con nombre, descripción, URL, categoría, orden, visibilidad y activación/ocultación. Nosotros lee contenido publicado y enlaces visibles del backend, incluidos filtros al visitar como Admin; conserva hero, galería y ubicación. Sin endpoint de despublicación ni editor de ubicación/galería. Contratos M7/M10 y servicios.md sincronizados, catálogo de consumo frontend actualizado. Compilación de producción completada; sin pruebas por instrucción del usuario.

- Proyectos (2026-11-10): lista y detalle omiten progress para Visitante o sin sesión; estudiantes, docentes y admin pueden leer el avance. Backend local actualizado.
