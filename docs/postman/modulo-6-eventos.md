# Postman — módulo 6: eventos, talleres y convocatorias

## Variables de la colección

Crear estas variables en el Environment de Postman:

| Variable | Valor de ejemplo | Uso |
|---|---|---|
| `baseUrl` | `http://localhost:8080` | URL del backend |
| `firebaseToken` | `eyJ...` | ID token vigente de Firebase |
| `sessionToken` | vacío | Cookie/token creado por Auth |
| `eventId` | vacío | UUID devuelto al crear un evento |

No se debe enviar el token de Firebase directamente en las rutas de eventos.
Primero se convierte en una sesión del backend. Postman debe conservar la
cookie `galash_session` automáticamente; si no conserva cookies, copie su valor
a la variable `sessionToken` y envíela como `Authorization: Bearer {{sessionToken}}`.

## 1. Crear sesión

**POST** `{{baseUrl}}/auth/sessions`

Headers:

```text
Content-Type: application/json
```

Body → raw → JSON:

```json
{
  "token": "{{firebaseToken}}"
}
```

Respuesta correcta: `201`. La respuesta contiene la sesión y el servidor envía
la cookie HttpOnly `galash_session`.

Si la cuenta no está registrada responde `403` con:

```json
{
  "code": "REGISTRATION_REQUIRED",
  "error": "registro incompleto",
  "action": "complete_registration"
}
```

La cuenta usada para crear eventos debe tener rol `Docente`, `Profesor` o
`Admin`. Un usuario nuevo inicia como `visitante`; un administrador debe
asignarle el rol antes de probar los endpoints protegidos.

## 2. Consultar la agenda pública

**GET** `{{baseUrl}}/events`

No requiere body, sesión ni headers de autorización.

Respuesta `200`:

```json
{
  "events": []
}
```

Filtros opcionales:

| Query parameter | Ejemplo | Descripción |
|---|---|---|
| `type` | `?type=taller` | Filtra por tipo exacto |
| `status` | `?status=proximo` | `publicado`, `proximo` o `vencido` |
| `from` | `?from=2026-11-01T00:00:00Z` | Inicio mínimo en RFC3339 |
| `to` | `?to=2026-12-01T23:59:59Z` | Inicio máximo en RFC3339 |

Ejemplos:

```text
GET {{baseUrl}}/events?type=taller
GET {{baseUrl}}/events?status=proximo
GET {{baseUrl}}/events?status=vencido
GET {{baseUrl}}/events?type=convocatoria&status=proximo
GET {{baseUrl}}/events?from=2026-11-01T00:00:00Z&to=2026-12-01T23:59:59Z
```

La agenda pública solo devuelve eventos con estado `publicado`. Los estados
`proximo` y `vencido` son filtros calculados sobre la fecha `startAt`.

### Listado administrativo

Un usuario con rol `Admin` puede enviar su sesión en:

```http
GET {{baseUrl}}/events
Authorization: Bearer {{sessionToken}}
```

Sin el filtro `status`, la respuesta incluye todos los estados: `borrador`,
`publicado`, `cancelado` y `oculto`. También puede filtrar un estado concreto:

```http
GET {{baseUrl}}/events?status=borrador
GET {{baseUrl}}/events?status=publicado
GET {{baseUrl}}/events?status=cancelado
GET {{baseUrl}}/events?status=oculto
```

Un visitante sigue viendo únicamente eventos publicados. Un usuario que no sea
Admin no puede consultar estados internos y recibe `403` al solicitar
`borrador`, `cancelado` u `oculto`.

## 3. Crear evento en borrador

**POST** `{{baseUrl}}/events`

Headers:

```text
Content-Type: application/json
Authorization: Bearer {{sessionToken}}
```

También funciona usando la cookie `galash_session` creada en el paso 1.

Body → raw → JSON:

```json
{
  "title": "Taller de investigación",
  "description": "Taller abierto para integrantes.",
  "type": "taller",
  "startAt": "2026-11-10T14:00:00-05:00",
  "endAt": "2026-11-10T16:00:00-05:00",
  "venue": "Sala 2",
  "meetingUrl": "https://meet.example.org/taller",
  "imageUrl": "https://example.org/taller.png",
  "registrationUrl": "https://example.org/registro"
}
```

Respuesta correcta: `201`. Guardar el campo `id` como `eventId`.

Campos:

| Campo | Tipo | Obligatorio | Regla |
|---|---|---:|---|
| `title` | string | Sí | No puede estar vacío |
| `description` | string | No | Descripción libre |
| `type` | string | Sí | Ej.: `evento`, `taller`, `convocatoria` |
| `startAt` | string | Sí | Fecha RFC3339 |
| `endAt` | string | No | RFC3339; no puede ser anterior a `startAt` |
| `venue` | string | No | Lugar físico |
| `meetingUrl` | string | No | URL `http` o `https` |
| `imageUrl` | string | No | URL `http` o `https` |
| `registrationUrl` | string | No | URL `http` o `https` |

`organizerId`, `status`, `createdAt` y `updatedAt` no se envían: el backend
calcula esos valores. Todo evento nuevo queda en `borrador`.

### Validación de agenda y lugar

No se permite crear ni editar un evento si existe otro evento no cancelado en el
mismo lugar y con horario superpuesto. El solapamiento también se valida cuando
los eventos están en `borrador`, porque un borrador ya reserva ese espacio.
Los eventos `cancelado` no bloquean nuevas reservas. Si dos eventos son en línea
y no tienen `venue`, esta validación de lugar no aplica.

Cuando `endAt` se omite, el evento reserva el resto del día desde su hora de
inicio en ese lugar. Por eso se permite un evento anterior que termine antes
de esa hora, pero no otro evento posterior o superpuesto en el mismo lugar.

La misma regla aplica si un evento tenía `endAt` y luego se actualiza con
`{"endAt": null}`: desde ese momento queda tratado como evento que ocupa el
resto de ese día. Un evento anterior que termine antes de su inicio, o un
evento en otra sala el mismo día, sí puede crearse.

Ejemplo de segundo evento que debe fallar si el primero usa `Sala 2` entre
14:00 y 16:00:

```json
{
  "title": "Otro taller",
  "type": "taller",
  "startAt": "2026-11-10T15:00:00-05:00",
  "endAt": "2026-11-10T17:00:00-05:00",
  "venue": "Sala 2"
}
```

Respuesta:

```http
409 Conflict
```

```json
{
  "error": "ya existe otro evento en el mismo lugar y horario"
}
```

## 4. Consultar detalle

**GET** `{{baseUrl}}/events/{{eventId}}`

No requiere body. Un visitante solamente puede consultar eventos publicados.
El Docente responsable o un Admin también puede consultar borradores, eventos
ocultos y cancelados.

Respuesta `200`:

```json
{
  "id": "uuid",
  "title": "Taller de investigación",
  "description": "Taller abierto para integrantes.",
  "type": "taller",
  "startAt": "2026-11-10T19:00:00Z",
  "endAt": "2026-11-10T21:00:00Z",
  "organizerId": "uuid",
  "organizerName": "Nombre Apellido",
  "venue": "Sala 2",
  "meetingUrl": "https://meet.example.org/taller",
  "imageUrl": "https://example.org/taller.png",
  "registrationUrl": "https://example.org/registro",
  "status": "borrador",
  "createdAt": "2026-10-24T00:00:00Z",
  "updatedAt": "2026-10-24T00:00:00Z"
}
```

## 5. Editar parcialmente

**PATCH** `{{baseUrl}}/events/{{eventId}}`

Headers:

```text
Content-Type: application/json
Authorization: Bearer {{sessionToken}}
```

Body de ejemplo:

```json
{
  "title": "Taller de investigación actualizado",
  "venue": "Auditorio principal",
  "endAt": "2026-11-10T22:00:00Z"
}
```

Los campos omitidos conservan su valor. El estado no se modifica mediante
`PATCH`; use las acciones específicas siguientes. Para quitar una fecha final
existente, envíe explícitamente:

```json
{
  "endAt": null
}
```

## 6. Publicar

**POST** `{{baseUrl}}/events/{{eventId}}/publish`

Headers: `Authorization: Bearer {{sessionToken}}`.
Sin body. Solo publica eventos en `borrador` u `oculto`.

## 7. Cancelar

**POST** `{{baseUrl}}/events/{{eventId}}/cancel`

Headers: `Authorization: Bearer {{sessionToken}}`.
Sin body. Conserva el evento y cambia su estado a `cancelado`.

## 8. Ocultar

**POST** `{{baseUrl}}/events/{{eventId}}/hide`

Headers: `Authorization: Bearer {{sessionToken}}`.
Sin body. El evento deja de aparecer en la agenda pública, pero no se elimina.

## Validaciones y respuestas esperadas

`201` al crear, `200` al consultar/editar/cambiar estado, `409` cuando existe
conflicto de horario/lugar y `204` para preflight.
Un body inválido, URL no HTTP(S), título/tipo/fecha faltante, estado inválido o
fecha final anterior a la inicial responde `400` con `{"error":"..."}`.
Sin sesión responde `401`; un rol no autorizado o un Docente que intenta usar
el evento de otro responde `403`; un evento inexistente o no visible responde
`404`. Los eventos nuevos siempre comienzan en `borrador` y nunca se borran
físicamente.

## Error 500 en `GET /events` por base existente

Si el volumen PostgreSQL fue creado antes del módulo 6, `init.sql` no se vuelve
a ejecutar automáticamente. En ese caso faltan columnas como
`organizador_id`, `inicia_en` o `estado`, y el endpoint devuelve `500`.

Aplicar una sola vez la migración desde la raíz del proyecto:

```bash
docker compose exec -T postgres psql -U galash -d galash \
  -v ON_ERROR_STOP=1 < database/migrations/006_events.sql
```

Después validar:

```bash
curl -i http://localhost:8080/events
```

Debe responder `HTTP/1.1 200 OK` y, si no hay publicados:

```json
{"events":[]}
```

No es necesario eliminar el volumen ni perder datos. La migración es idempotente
porque utiliza `ADD COLUMN IF NOT EXISTS` y `CREATE INDEX IF NOT EXISTS`.
