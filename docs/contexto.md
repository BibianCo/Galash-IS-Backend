# Active Context

## System

GALASH-UPTC web system for academic management, research, and communication.

## Implemented state

- Versioned OpenCode configuration in `opencode.json`.
- Shared agent rules in `AGENTS.md`.
- Selected skills managed by the repository in `.opencode/skills/`.
- Append-only technical log in `docs/bitacora.ndjson`.
- Append-only testing ledger in `docs/testing/events.ndjson`.
- Vanilla JavaScript testing dashboard in `docs/testing/dashboard/`.
- Private requirements excluded from version control.

## Conventions

- User defines scope and requirements for each task.
- Keep changes small and limited to requested scope.
- Backend validates authorization.
- Critical actions produce audit records.
- Personal data follows least privilege.
- Completed work records files and tests in the log.
- Servicio `Auth` en Go valida ID tokens de Firebase y sincroniza usuarios con PostgreSQL.
- El backend Go está organizado como un monolito con arquitectura hexagonal:
  entidades y errores en `internal/domain`, puertos en `internal/ports`, casos
  de uso en `internal/application`, adaptadores en `internal/adapters` y
  composición en `cmd/api`, con un único `main.go` y `Dockerfile` en la raíz.
- PostgreSQL y Auth se levantan juntos mediante el único `docker-compose.yml` de
  la raíz; PostgreSQL usa `database/init.sql` para inicializar el esquema.

## Record

- Detailed history: `docs/bitacora.ndjson`.
- This file describes current state, not history.
- Contains no private requirements, credentials, or personal data.
