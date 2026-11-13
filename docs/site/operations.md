# Operación

## Variables y dependencias

El servicio requiere PostgreSQL y la configuración de Firebase definida por el
entorno de ejecución. El `docker-compose.yml` de la raíz levanta PostgreSQL y el
servicio de autenticación local.

## Verificación básica

```bash
go test ./...
```

La salud del servicio se puede comprobar con:

```bash
curl -i http://localhost:8080/health
```

## Contrato API

`docs/openapi.yaml` es la referencia navegable de los endpoints implementados.
Los cambios de método, ruta, payload, autorización, estado HTTP o formato de error
deben reflejarse también en el contrato compartido del proyecto y en `servicios.md`.

## Seguridad

- Las rutas protegidas requieren una sesión válida.
- La autorización se valida en el backend según rol y relación con el semillero.
- Los archivos se guardan en PostgreSQL y no se exponen mediante URLs públicas predecibles.
- Las operaciones críticas generan registros de auditoría.
