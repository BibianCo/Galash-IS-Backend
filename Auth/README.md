# Servicio Auth

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

Ejemplo de registro:

```bash
curl -X POST http://localhost:8080/auth/register \
  -H 'Authorization: Bearer FIREBASE_ID_TOKEN' \
  -H 'Content-Type: application/json' \
  -d '{"nombre":"Ana","apellido":"Gómez"}'
```
