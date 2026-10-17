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

Sesion por cookie HttpOnly. `POST /auth/register`: `Authorization: Bearer
<FIREBASE_ID_TOKEN>`, cuerpo `{"nombre","apellido"}`; nunca password, dni ni
tipo_dni. Rutas en `internal/adapters/inbound/http/handler.go`.

| Metodo | Ruta | Descripcion | Contrato | Uso |
|---|---|---|---|---|
| GET | `/health` | salud y bd | - | [ ] |
| POST | `/auth/register` | alta por token | - | [ ] |
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
| PATCH | `/seedbeds/{id}` | edita | M2 | [ ] |
| PATCH | `/seedbeds/{id}/visibility` | estado | M2 | [ ] |
| GET | `/seedbeds/{id}/members` | integrantes | M2 | [ ] |
| POST | `/seedbeds/{id}/members` | vincula | M2 | [ ] |
| DELETE | `/seedbeds/{id}/members/{userId}` | retiro | M2 | [ ] |

Fuera de contrato: `- /health` infraestructura · `- /auth/register` decision
pendiente (el contrato define `POST /registration-requests`) · `-
/admin/members` datos sensibles, decision pendiente · `- /find/user` alias
legacy · `- /users/roles` alias de `PUT /users/{userId}/roles`.

## PostgreSQL (Docker, host `5433`)

`database/init.sql` al crear el volumen; vars en `database/.env.example`.

## Ejecucion (backend)

`.env` en la raiz (ver `deploy/auth/.env.example`). `go mod tidy && go run .`
(Auth) o `docker compose up -d --build` (todo). Credencial de Firebase: nunca
al repositorio. Front consume via proxy `/api` -> `http://localhost:8080`.

## Pendiente en contrato (sin implementar)

M1: password-recovery-requests, password-resets (S) | M2: `GET
/seedbeds/{id}/space` | M3 solicitudes: 5 | M4 actividades/proyectos/
propuestas/resenas: 19 | M5 miembros: 5 (CRUD, status, publications) | M6
eventos: 7 | M7 contenidos: 6 | M8 comunicados: 7 | M9 cronograma: 5 | M10
enlaces: 4 | M11 roles: 6 (roles, permissions, GET users/{userId}/roles,
users/me/dashboard) | M12: audit-logs | M13 files: 2
