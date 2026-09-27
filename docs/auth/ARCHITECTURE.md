# Arquitectura hexagonal del monolito

El backend se mantiene como un único despliegue monolítico. La separación por
capas evita que el dominio y los casos de uso dependan de HTTP, Firebase o
PostgreSQL.

```text
main.go
    └── cmd/api                 composición de la aplicación
        ├── internal/config      configuración y variables de entorno
        ├── internal/domain      entidades y errores de negocio
        ├── internal/ports       contratos de entrada y salida
        ├── internal/application casos de uso
        └── internal/adapters
            ├── inbound/http     API REST
            └── outbound
                ├── firebase     verificación de identidad
                └── postgres     persistencia y salud de la base de datos
```

## Flujo actual

`POST /auth/register` entra por el adaptador HTTP, extrae el token y el JSON,
ejecuta el caso de uso `Register` y utiliza los puertos para verificar la
identidad y guardar el usuario. La composición en `cmd/api` conecta las
implementaciones concretas de Firebase y PostgreSQL con esos puertos.

`GET /health` usa el puerto de salud implementado por el adaptador PostgreSQL.

La raíz `main.go` es el único punto de entrada y es compatible con `go run .` y
con el `Dockerfile`; únicamente delega la composición a `cmd/api`. No se
eliminó la lógica funcional, sino que se distribuyó en sus responsabilidades.
