# GALASH UPTC

Documentación técnica del backend del Sistema de Información y Gestión Académica GALASH.

## Propósito

El sistema centraliza la gestión de integrantes, semilleros, proyectos, actividades,
solicitudes de registro, eventos, comunicados y contenido institucional para el grupo
GALASH de la UPTC.

## Inicio rápido

```bash
go run .
```

Con el servidor ejecutándose:

- API: `http://localhost:8080`
- Especificación: [OpenAPI](../openapi.yaml)
- Interfaz interactiva: [Swagger UI](swagger.md)

Para levantar esta documentación:

```bash
pip install -r docs/requirements.txt
mkdocs serve
```

## Alcance documentado

La especificación OpenAPI cubre las rutas registradas por el adaptador HTTP,
incluyendo autenticación, autorización, semilleros, investigación, calendario,
comunicados, contenido, enlaces, auditoría y archivos.
