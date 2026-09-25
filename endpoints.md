# Endpoints del proyecto

Catálogo de los endpoints HTTP expuestos actualmente por los servicios del
proyecto. Úsalo como referencia rápida para pruebas e integración.

## Servicio Auth (Go)

- **Ubicación:** `Auth/`
- **Base URL local:** `http://localhost:8080`
- **Formato de datos:** JSON

| Método | Ruta | Descripción | Autenticación | Respuestas |
|---|---|---|---|---|
| `GET` | `/health` | Estado del servicio y de la conexión a PostgreSQL | No requiere | `200` `{"status":"ok"}` · `503` `{"error":"base de datos no disponible"}` |
| `POST` | `/auth/register` | Valida el ID token de Firebase y crea o actualiza el usuario en PostgreSQL | `Authorization: Bearer <FIREBASE_ID_TOKEN>` | `201` `{"id":"<uuid>","firebase_uid":"<uid>"}` · `400` cuerpo JSON inválido o token sin nombre/correo · `401` falta header o token inválido · `500` error al guardar |

### Ejemplo `POST /auth/register`

```bash
curl -X POST http://localhost:8080/auth/register \
  -H "Authorization: Bearer $FIREBASE_ID_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"nombre":"Ana","apellido":"Gómez"}'
```

### Ejemplo `GET /health`

```bash
curl http://localhost:8080/health
```

## Servicio PostgreSQL (Docker)

No expone endpoints HTTP. Solo escucha en el puerto `5432` para conexiones
directas de base de datos:

- **Host:** `localhost` (desde fuera de Docker) / `galash-postgres` (desde la red de Docker)
- **Puerto:** `5432`
- **Base de datos:** `galash`

## Servicios sin endpoints todavía

Los demás dominios del modelo (semilleros, solicitudes de registro, proyectos,
actividades, eventos, noticias, perfil de investigador, seguimiento y
colaboradores externos) existen como tablas en `database/init.sql`, pero aún no
tienen servicios ni endpoints implementados.
