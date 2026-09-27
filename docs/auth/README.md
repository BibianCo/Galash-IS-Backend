# Servicio Auth

La aplicación es un monolito organizado con arquitectura hexagonal. La
composición está en `cmd/api`, el dominio y los puertos en `internal/core`,
los casos de uso en `internal/application` y las integraciones en
`internal/adapters`. Consulta `ARCHITECTURE.md` para el mapa de dependencias.

El endpoint `POST /auth/register` recibe el ID token de Firebase en el header
`Authorization: Bearer <token>`, lo valida con Firebase Admin SDK y crea o
actualiza el usuario local usando el `uid` de Firebase como referencia única.

El servicio necesita credenciales de una cuenta de servicio de Firebase. No
guardes ese JSON en el repositorio.

```bash
cp .env.example .env
go mod tidy
go run .
```

### Ejecución con Docker (sin instalar Go)

1. Ejecuta el compose general desde la raíz del proyecto.
2. Coloca la cuenta de servicio de Firebase en
   `deploy/auth/firebase-service-account.json`
   o exporta `FIREBASE_CREDENTIALS_FILE` con la ruta absoluta del archivo.
3. Ejecuta:

```bash
docker compose up -d --build
```

El contenedor se conecta a la red `database_default` creada por el compose de
PostgreSQL y expone el servicio en el puerto 8080.

Ejemplo de registro:

```bash
curl -X POST http://localhost:8080/auth/register \
  -H 'Authorization: Bearer FIREBASE_ID_TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{"nombre":"Ana","apellido":"Gómez"}'
```
