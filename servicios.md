# Servicios

Registro compartido backend <-> front. Diseno objetivo:
`planificacion_modulos_y_endpoints_galash_is_2026.md` (contrato).
Desviaciones: `docs/mapeo_estado_vs_contrato.md`. Mantenimiento y handoff:
`AGENTS.md` -> Shared contract.

Columnas: `Contrato` = modulo del contrato, `-` = fuera de contrato (motivos al
pie). `Uso` = consumo del front: `[ ]` sin usar, `[x]` usado. Todo endpoint del
contrato ausente de esta tabla NO existe: no construir UI que lo llame; si la
tarea lo necesita, reportar bloqueo. Marcar `[x]` solo al consumir. Agregar
filas o marcar disponible solo con registro/handoff del backend.

## Auth (Go, `:8080`)

Sesion por cookie HttpOnly. `POST /auth/register` crea o sincroniza la cuenta,
independiente de las solicitudes del M3. Requiere `Authorization: Bearer
<FIREBASE_ID_TOKEN>` y `{"nombre","apellido","tipo_dni","dni","email",
"fecha_nacimiento","telefono"}` obligatorios (`fecha_nacimiento` en `AAAA-MM-DD`)
más `codigo_estudiantil` opcional; sin password ni rol, email igual al token.
`409` si `dni` o `codigo_estudiantil` ya son de otra cuenta. El rol nuevo es
`visitante` y el re-registro conserva el rol existente. `GET /users/me`
devuelve además `document_type`, `document_number`, `student_code`,
`birth_date` y `phone` para prellenar formularios, y `managed_seedbed_ids`
(semilleros que el usuario dirige o en los que es delegado; `[]` si ninguno)
y `photo_url` (enlace de la imagen de perfil; `""` si no hay).
`PUT /users/me/photo` con `{"photo_url":"https://..."}` guarda ese enlace
en el usuario de la sesión: `200` `{"photo_url":...}`, `400` JSON o URL
inválida (solo `http`/`https` absoluta, máx. 2048), `401` sin sesión.
`GET /auth/register/matches` (Bearer Firebase) devuelve
`{"matches":[{"id","name","role"}]}` con integrantes importados del SGI sin
reclamar que coinciden con el correo verificado `primernombre.apellido[NN]@uptc.edu.co`;
lista vacía en otro caso, `401` token inválido. Usar tras `403 REGISTRATION_REQUIRED`
para preguntar "¿Eres X?". Si confirma, `POST /auth/register` con `claim_id`
vincula esa cuenta (conserva rol y perfil); `409` si no coincide o ya fue reclamada.

`POST /registration-requests` (Únete) exige sesión (`401` sin ella) y
`seedbedId` obligatorio (`400` si falta o no existe). No envía
`nombre`, `cc`, `telefono` ni `fechaNacimiento`: salen de la cuenta. `cod` es
opcional (vacío se ignora; con valor actualiza la cuenta, `409` si es de otra).
`400` si falta un campo académico o la cuenta no tiene documento, teléfono o
fecha de nacimiento. Rutas en `internal/adapters/inbound/http/handler.go`.

| Metodo | Ruta | Descripcion | Contrato | Uso |
|---|---|---|---|---|
| GET | `/health` | salud y bd | - | [ ] |
| POST | `/auth/register` | alta por token (`claim_id` opcional) | M1 | [ ] |
| GET | `/auth/register/matches` | integrantes SGI que coinciden con el correo | M1 | [ ] |
| POST | `/auth/sessions` | crea sesion | M1 | [ ] |
| DELETE | `/auth/sessions/current` | cierra sesion | M1 | [ ] |
| GET | `/users/me` | perfil y roles | M1 | [ ] |
| PUT | `/users/me/photo` | guarda URL de foto de perfil | M1 | [ ] |
| GET | `/members` | dir. publico | M5 | [ ] |
| POST | `/members` | crea perfil | M5 | [ ] |
| GET | `/members/{id}` | detalle público/privado autorizado | M5 | [ ] |
| PATCH | `/members/{id}` | edita perfil | M5 | [ ] |
| PATCH | `/members/{id}/status` | activa o desactiva perfil | M5 | [ ] |
| GET | `/members/{id}/publications` | publicaciones visibles | M5 | [ ] |
| GET | `/admin/members` | busq. admin (requiere `q`) | M11 | [ ] |
| PUT | `/users/roles` | alias rol | - | [ ] |
| PUT | `/users/{userId}/roles` | asigna rol (id = Firebase UID) | M11 | [ ] |
| GET | `/seedbeds` | lista | M2 | [ ] |
| POST | `/seedbeds` | crea | M2 | [ ] |
| GET | `/seedbeds/{id}` | detalle | M2 | [ ] |
| GET | `/seedbeds/{id}/space` | vista compuesta | M2 | [ ] |
| PATCH | `/seedbeds/{id}` | edita | M2 | [ ] |
| PATCH | `/seedbeds/{id}/visibility` | estado | M2 | [ ] |
| GET | `/seedbeds/{id}/members` | integrantes | M2 | [ ] |
| POST | `/seedbeds/{id}/members` | vincula | M2 | [ ] |
| DELETE | `/seedbeds/{id}/members/{userId}` | retiro (quita delegación) | M2 | [ ] |
| PUT | `/seedbeds/{id}/delegates` | reemplaza delegados (0-2), director o Admin | M2 | [ ] |
| POST | `/registration-requests` | radica solicitud de la cuenta en sesión | M3 | [ ] |
| GET | `/registration-requests/{id}/status` | consulta propia con token/sesión | M3 | [ ] |
| GET | `/admin/registration-requests` | lista; Admin todas, director/delegado las de sus semilleros | M3 | [ ] |
| GET | `/admin/registration-requests/{id}` | detalle e historial; Admin o gestor del semillero | M3 | [ ] |
| PATCH | `/admin/registration-requests/{id}/status` | actualiza estado; Admin o gestor del semillero | M3 | [ ] |
| GET | `/seedbeds/{seedbedId}/activities` | lista actividades | M4 | [ ] |
| POST | `/seedbeds/{seedbedId}/activities` | registra actividad | M4 | [ ] |
| GET | `/activities/{id}` | detalle de actividad | M4 | [ ] |
| PATCH | `/activities/{id}` | modifica actividad | M4 | [ ] |
| GET | `/projects` | lista proyectos | M4 | [ ] |
| POST | `/projects` | registra proyecto | M4 | [ ] |
| GET | `/projects/{id}` | detalle de proyecto | M4 | [ ] |
| PATCH | `/projects/{id}` | modifica proyecto | M4 | [ ] |
| PATCH | `/projects/{id}/progress` | actualiza avance | M4 | [ ] |
| GET | `/projects/progress` | resumen de avance | M4 | [ ] |
| GET | `/projects/{id}/history` | historial de proyecto | M4 | [ ] |
| GET | `/research-proposals` | lista propuestas | M4 | [ ] |
| POST | `/research-proposals` | registra propuesta | M4 | [ ] |
| GET | `/research-proposals/{id}` | detalle de propuesta | M4 | [ ] |
| PATCH | `/research-proposals/{id}` | modifica propuesta | M4 | [ ] |
| POST | `/research-proposals/{id}/convert-to-project` | convierte propuesta aprobada | M4 | [ ] |
| POST | `/seedbeds/{seedbedId}/reviews` | crea reseña | M4 | [ ] |
| GET | `/admin/seedbed-reviews` | lista reseñas | M4 | [ ] |
| PATCH | `/admin/seedbed-reviews/{id}/moderation` | modera reseña | M4 | [ ] |
| GET | `/events` | agenda pública filtrable; Admin autenticado ve todos los estados | M6 | [ ] |
| POST | `/events` | crea evento en borrador | M6 | [ ] |
| GET | `/events/{id}` | detalle publicado/autorizado | M6 | [ ] |
| PATCH | `/events/{id}` | edita evento | M6 | [ ] |
| POST | `/events/{id}/publish` | publica evento | M6 | [ ] |
| POST | `/events/{id}/cancel` | cancela evento | M6 | [ ] |
| POST | `/events/{id}/hide` | oculta evento | M6 | [ ] |
| GET | `/content/pages` | páginas institucionales publicadas | M7 | [ ] |
| GET | `/content/pages/{slug}` | página institucional publicada | M7 | [ ] |
| GET | `/admin/content/pages` | páginas públicas y borradores | M7 | [ ] |
| POST | `/admin/content/pages` | crea página en borrador | M7 | [ ] |
| PATCH | `/admin/content/pages/{id}` | edita página y sanitiza contenido | M7 | [ ] |
| POST | `/admin/content/pages/{id}/publish` | publica página vigente | M7 | [ ] |
| GET | `/announcements` | comunicados publicados, borradores propios y todos los borradores para Admin; lectura propia | M8 | [ ] |
| POST | `/announcements` | crea comunicado con adjuntos permitidos | M8 | [ ] |
| GET | `/announcements/{id}` | detalle y adjuntos autorizados | M8 | [ ] |
| PATCH | `/announcements/{id}` | edita comunicado en borrador | M8 | [ ] |
| POST | `/announcements/{id}/publish` | publica comunicado | M8 | [ ] |
| POST | `/announcements/{id}/read-confirmations` | confirma lectura idempotente | M8 | [ ] |
| GET | `/announcements/{id}/read-status` | trazabilidad por docente para autor/admin | M8 | [ ] |
| GET | `/internal-calendar/events` | agenda interna por rango, vista y semillero | M9 | [ ] |
| POST | `/internal-calendar/events` | crea reunión, tutoría o hito | M9 | [ ] |
| GET | `/internal-calendar/events/{id}` | detalle y participantes autorizado | M9 | [ ] |
| PATCH | `/internal-calendar/events/{id}` | actualiza evento o participantes | M9 | [ ] |
| DELETE | `/internal-calendar/events/{id}` | cancela lógicamente conservando trazabilidad | M9 | [ ] |
| GET | `/links` | enlaces oficiales visibles para público; Admin activo consulta todos los estados, con filtro por categoría y orden | M10 | [ ] |
| POST | `/links` | crea enlace oficial validando URL externa | M10 | [ ] |
| PATCH | `/links/{id}` | actualiza enlace oficial y registra historial | M10 | [ ] |
| PATCH | `/links/{id}/status` | activa, desactiva u oculta enlace sin borrar historial | M10 | [ ] |
| GET | `/audit-logs` | consulta bitácora filtrable por actor, acción, entidad y periodo; Admin | M12 | [ ] |
| POST | `/files` | carga multipart segura asociada a una entidad | M13 | [ ] |
| GET | `/files/{id}` | descarga autorizada de un archivo adjunto | M13 | [ ] |

Payloads M4 en camelCase (`id`, `seedbedId`, `responsibleIds`, `memberIds`,
`progress`, ...). `GET /research-proposals/{id}` incluye `history` y
`projectId` (null hasta convertir); el estado de una propuesta solo lo cambia
`Docente`/`Admin`. Los `PATCH` de M4 son parciales: los campos omitidos
conservan su valor. `GET /research-proposals` devuelve todas a
`Docente`/`Admin` y solo propias o como responsable a `Estudiante`.
`GET /admin/seedbed-reviews` lista `pendiente` salvo `?status=`. Ids
malformados o semillero inexistente responden `404`. En `POST
/seedbeds/{seedbedId}/activities`, `status` es opcional si hay `date`: se
calcula (`próxima`, `en curso`, `finalizada`).

Director y delegados (relaciones, no roles): el director es el
`responsibleTeacherId`; puede designar hasta 2 delegados entre integrantes
activos con `PUT /seedbeds/{id}/delegates` `{"userIds":[...]}` (`204`; `400`
lista inválida, `403` si no es director/Admin). `GET /seedbeds/{id}/members`
marca `delegate: true`. Director y delegados gestionan solicitudes (M3),
actividades (M4), eventos de su semillero (M6) y cronograma (M9, sin
implementar). Aprobar una solicitud inscribe al solicitante como integrante
activo del semillero. Una solicitud la decide el primero que la procese; si otro ya
cambió el estado responde `400`. Sin permiso: `403`.

M2: los semilleros incluyen `topics` (máx. 4 `{title, phrase}`); en `PATCH
/seedbeds/{id}` omitirlo conserva los temas. `GET /admin/members` responde
`{"users": [...]}`, exige `q` e incluye cuentas inactivas.

M6 usa `title`, `description`, `type`, `startAt`, `endAt`, `venue`, `meetingUrl`,
`imageUrl` y `registrationUrl`. Las fechas son RFC3339; los estados son
`borrador`, `publicado`, `oculto` y `cancelado`. La agenda pública solo muestra
`publicado`; `status=proximo` y `status=vencido` permiten separar eventos según
la fecha de inicio. `POST /events` acepta `seedbedId` opcional (fijo tras
crearlo; `400` si no existe). Sin `seedbedId` crea `Docente` o `Admin`; con
él, solo `Admin` o director/delegado de ese semillero. Administran el evento:
`Admin`, el docente organizador o director/delegado del semillero del evento.

M9 usa `seedbedId`, `type` (`reunion`, `tutoria` o `hito`), `title`, `description`,
`startAt`, `endAt` y `participantIds`. Las fechas son RFC3339. Los estudiantes
solo consultan eventos del semillero al que pertenecen; el director o un
delegado del semillero (o Admin) crean, actualizan y cancelan (`403` en otro
caso). Un solape de horario de un participante responde `409`. La eliminación
es lógica y conserva historial.

Fuera de contrato: `- /health` infraestructura · `- /users/roles` alias de
`PUT /users/{userId}/roles`. `GET /admin/members` responde `400` sin `q`.

## PostgreSQL (Docker, host `5433`)

`database/init.sql` al crear el volumen; vars en `database/.env.example`.

## Ejecucion (backend)

`.env` en la raiz (ver `deploy/auth/.env.example`). `go mod tidy && go run .`
(Auth) o `docker compose up -d --build` (todo). Credencial de Firebase: nunca
al repositorio. Front consume via proxy `/api` -> `http://localhost:8080`.

## Pendiente en contrato (sin implementar)

M1: password-recovery-requests, password-resets (S) | M4 actividades/proyectos/
propuestas/resenas: 19 | M5 completado | M6
eventos: 7 | M7 contenidos: 6 | M8 comunicados: 7 | M9 cronograma: 5 | M10
enlaces: 4 | M11 roles: 6 (roles, permissions, GET users/{userId}/roles,
users/me/dashboard) | M12: audit-logs | M13 files: 2
