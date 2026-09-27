# Servicios disponibles

Este archivo resume los servicios existentes para que otro agente pueda
integrarlos sin tener que reconstruir el contexto del proyecto.

## 1. Auth

**Ubicación:** raíz del backend Go
**Tecnología:** Go 1.22, Firebase Admin SDK y PostgreSQL  
**Puerto predeterminado:** `8080`

El servicio es un monolito con arquitectura hexagonal. `cmd/api` realiza la
composición; `internal/application` contiene los casos de uso; `internal/domain`
define entidades y errores de negocio; `internal/ports` define interfaces; y
`internal/adapters` contiene los adaptadores HTTP, Firebase y PostgreSQL. El
único `main.go` y el único `Dockerfile` están en la raíz del backend.

### Responsabilidad

Valida los ID tokens emitidos por Firebase y sincroniza el usuario con la
tabla `usuarios` de PostgreSQL. El `uid` de Firebase se guarda en la columna
`firebase_uid`, que es única y sirve como referencia externa del usuario.

### Endpoints

| Método | Ruta | Descripción |
|---|---|---|
| `GET` | `/health` | Comprueba que el servicio y la conexión a PostgreSQL estén disponibles. |
| `POST` | `/auth/register` | Valida el token de Firebase y crea o actualiza el usuario local. |

El registro requiere el header:

```text
Authorization: Bearer <FIREBASE_ID_TOKEN>
```

El cuerpo JSON acepta:

```json
{
  "nombre": "Ana",
  "apellido": "Gómez"
}
```

El servicio obtiene el correo y el `uid` desde el token de Firebase. No se
envían contraseñas, `dni` ni `tipo_dni` al endpoint.

### Configuración

Copiar `deploy/auth/.env.example` como referencia y definir las variables en
`.env` en la raíz del proyecto:

```env
PORT=8080
DATABASE_URL=postgres://galash:galash@localhost:5432/galash?sslmode=disable
FIREBASE_CREDENTIALS_FILE=/ruta/segura/firebase-service-account.json
```

El archivo de cuenta de servicio de Firebase es secreto y no debe subirse al
repositorio.

### Ejecución

```bash
cd .
go mod tidy
go run .
```

Alternativa con Docker, sin instalar Go:

```bash
# Levanta PostgreSQL y Auth desde la raíz del proyecto
cd ..
docker compose up -d --build
```

## 2. Base de datos PostgreSQL

**Ubicación:** `database/`  
**Tecnología:** Docker Compose y PostgreSQL 16  
**Puerto predeterminado del host:** `5433` (PostgreSQL escucha en `5432` dentro de Docker)

### Responsabilidad

Levanta la instancia PostgreSQL local y carga `database/init.sql` cuando se
crea el volumen por primera vez. El esquema contiene `usuarios` y las tablas
relacionadas con líneas de investigación, semilleros, solicitudes, proyectos,
actividades, eventos, noticias y colaboradores.

### Configuración y ejecución

```bash
cd ..
docker compose up -d postgres
```

Variables disponibles en `database/.env.example`:

```env
POSTGRES_DB=galash
POSTGRES_USER=galash
POSTGRES_PASSWORD=galash
POSTGRES_PORT=5432
```

Para reiniciar la base de datos desde cero:

```bash
docker compose down -v
docker compose up -d
```

## Orden recomendado de inicio

1. Configurar las credenciales de Firebase Admin para `deploy/auth/`.
2. Ejecutar `docker compose up -d --build` desde la raíz.
4. Consumir `POST /auth/register` con un ID token válido de Firebase.

## Relación entre servicios

```text
Cliente / Frontend
        │
        │ Firebase ID token
        ▼
Auth (Go)
        │ valida token y guarda firebase_uid
        ▼
PostgreSQL (Docker)
```
