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
<FIREBASE_ID_TOKEN>` y `{"nombre","apellido","tipo_dni","dni","email"}`;
sin password ni rol, email igual al token. El rol nuevo es `visitante` y el
re-registro conserva el rol existente. Rutas en `internal/adapters/inbound/http/handler.go`.

| Metodo | Ruta | Descripcion | Contrato | Uso |
|---|---|---|---|---|
| GET | `/health` | salud y bd | - | [ ] |
| POST | `/auth/register` | alta por token | M1 | [ ] |
| POST | `/auth/sessions` | crea sesion | M1 | [ ] |
| DELETE | `/auth/sessions/current` | cierra sesion | M1 | [ ] |
| GET | `/users/me` | perfil y roles | M1 | [ ] |
| GET | `/members` | dir. publico | M5 | [ ] |
| GET | `/admin/members` | busq. admin | - | [ ] |
| GET | `/find/user` | alias busq. | - | [ ] |
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
| DELETE | `/seedbeds/{id}/members/{userId}` | retiro | M2 | [ ] |
| POST | `/registration-requests` | radica solicitud pública | M3 | [ ] |
| GET | `/registration-requests/{id}/status` | consulta propia con token/sesión | M3 | [ ] |
| GET | `/admin/registration-requests` | lista administrativa | M3 | [ ] |
| GET | `/admin/registration-requests/{id}` | detalle e historial administrativo | M3 | [ ] |
| PATCH | `/admin/registration-requests/{id}/status` | actualiza estado y auditoría | M3 | [ ] |
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

Fuera de contrato: `- /health` infraestructura · `- /admin/members` datos
sensibles, decision pendiente · `- /find/user` alias legacy · `- /users/roles`
alias de `PUT /users/{userId}/roles`.

## PostgreSQL (Docker, host `5433`)

`database/init.sql` al crear el volumen; vars en `database/.env.example`.

## Ejecucion (backend)

`.env` en la raiz (ver `deploy/auth/.env.example`). `go mod tidy && go run .`
(Auth) o `docker compose up -d --build` (todo). Credencial de Firebase: nunca
al repositorio. Front consume via proxy `/api` -> `http://localhost:8080`.

## Pendiente en contrato (sin implementar)

M1: password-recovery-requests, password-resets (S) | M4 actividades/proyectos/
propuestas/resenas: 19 | M5 miembros: 5 (CRUD, status, publications) | M6
eventos: 7 | M7 contenidos: 6 | M8 comunicados: 7 | M9 cronograma: 5 | M10
enlaces: 4 | M11 roles: 6 (roles, permissions, GET users/{userId}/roles,
users/me/dashboard) | M12: audit-logs | M13 files: 2
