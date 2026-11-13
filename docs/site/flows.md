# Flujos principales

## Autenticación y sesión

```mermaid
sequenceDiagram
    participant C as Cliente
    participant API as API GALASH
    participant F as Firebase
    participant DB as PostgreSQL
    C->>API: POST /auth/sessions (ID token)
    API->>F: Verificar ID token
    F-->>API: UID y claims
    API->>DB: Sincronizar usuario y sesión
    DB-->>API: Sesión creada
    API-->>C: Cookie/sesión
```

## Solicitud de registro

```mermaid
flowchart TD
    A[Estudiante autenticado] --> B[POST /registration-requests]
    B --> C{Solicitud válida}
    C -- No --> D[400 con error controlado]
    C -- Sí --> E[Solicitud Radicada]
    E --> F[Director, delegado o Admin revisa]
    F --> G{Decisión}
    G -- Aprobar --> H[Asignar rol estudiante e inscribir]
    G -- Rechazar --> I[Guardar estado y auditoría]
```

## Archivos asociados

```mermaid
flowchart LR
    U[Usuario autorizado] --> UP[POST /files]
    UP --> V[Validar entidad, extensión y 10 MB]
    V --> P[(PostgreSQL)]
    U --> DL[GET /files/{id}]
    DL --> AUTH[Revalidar acceso a entidad]
    AUTH --> P
```
