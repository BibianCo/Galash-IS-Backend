# Swagger UI

La interfaz interactiva se encuentra en [`docs/swagger/index.html`](../swagger/)
cuando se sirve el directorio de documentación como archivos estáticos.

También puede abrirse directamente desde el explorador de archivos o desde un
servidor estático local:

```bash
python -m http.server 8000 --directory docs
```

Luego visita `http://localhost:8000/swagger/`. La interfaz carga la especificación
`docs/openapi.yaml` y permite explorar las operaciones y enviar solicitudes al
servidor configurado.
