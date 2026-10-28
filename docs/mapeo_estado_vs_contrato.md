# Mapeo: estado real vs contrato

Fecha: 2026-10-16. Documento de estado, no de historia: se actualiza o se
reemplaza; no crece indefinidamente.

## Cómo leerlo

- **Contrato:** `planificacion_modulos_y_endpoints_galash_is_2026.md` (84 endpoints, copias idénticas en ambos repos).
- **Real backend:** rutas registradas en `internal/adapters/inbound/http/handler.go` (verificado contra código, no contra documentación).
- **Real frontend:** llamadas HTTP en `src/app/` (verificado contra código).
- El mapeo compara **método y ruta**. La conformidad de campos de respuesta y códigos de error **no está verificada**.
- `endpoints.md` (catálogo que circula como input) está **desactualizado**; ver "Desviaciones".

## Resumen

| Métrica | Valor |
|---|---|
| Endpoints del contrato implementados | **14 / 84 (17 %)** |
| Endpoints implementados fuera del contrato | **4** |
| Endpoints que consume el frontend | **4** |
| Módulos del contrato sin nada implementado | 8 de 13 |

## Estado por módulo

| Módulo | Contrato | Implementados | Pendientes | Estado |
|---|---|---|---|---|
| 1. Autenticación y sesión | 6 | 4 | 2 (recuperación de contraseña, Should) | Parcial |
| 2. Semilleros | 9 | 8 | `GET /seedbeds/{id}/space` | Casi completo |
| 3. Solicitudes de registro | 5 | 0 | todos | Sin iniciar |
| 4. Actividades/proyectos/propuestas | 19 | 0 | todos | Sin iniciar |
| 5. Integrantes y perfiles | 6 | 1 | 5 | Inicial |
| 6. Eventos | 7 | 0 | todos | Sin iniciar |
| 7. Información institucional | 6 | 0 | todos | Sin iniciar |
| 8. Comunicados | 7 | 0 | todos | Sin iniciar |
| 9. Cronograma interno | 5 | 0 | todos | Sin iniciar |
| 10. Enlaces oficiales | 4 | 0 | todos | Sin iniciar |
| 11. Roles, permisos y paneles | 7 | 1 | 6 | Inicial |
| 12. Auditoría | 1 | 0 | `GET /audit-logs` | Sin iniciar |
| 13. Archivos adjuntos | 2 | 0 | todos | Sin iniciar |

### Detalle de lo implementado que sí cumple contrato

- Módulo 1: `POST /auth/register`, `POST /auth/sessions`, `DELETE /auth/sessions/current`, `GET /users/me`.
- Módulo 2: `GET /seedbeds`, `POST /seedbeds`, `GET /seedbeds/{id}`, `PATCH /seedbeds/{id}`, `PATCH /seedbeds/{id}/visibility`, `GET /seedbeds/{id}/members`, `POST /seedbeds/{id}/members`, `DELETE /seedbeds/{id}/members/{userId}`.
- Módulo 5: `GET /members`.
- Módulo 11: `PUT /users/{userId}/roles` (ver nota sobre identificador).

## Desviaciones detectadas (acción requerida)

1. **`endpoints.md` está obsoleto.** Dice que solo existen `GET /health` y
   `POST /auth/register` y que los demás dominios "aún no tienen servicios ni
   endpoints implementados". El código registra 18 rutas (semilleros, miembros
   y roles ya existen). Este archivo circula como input al equipo de frontend:
   **mentirle al front cuesta dinero**. Actualizarlo o eliminarlo.
2. **`GET /admin/members` y `GET /find/user`**: resuelto 2026-10-27.
   `GET /admin/members` entra al contrato (M11, solo `Admin`, búsqueda
   obligatoria). `GET /find/user` se eliminó.
3. **`PUT /users/roles`** (sin `{userId}`): alias de
   `PUT /users/{userId}/roles`, fuera del contrato. Eliminar o documentar.
4. **`GET /health`**: infraestructura, legítimamente fuera del contrato.
   Registrarlo como excepción conocida para no volver a reportarlo.
5. **Identificador de usuario en roles**: el contrato dice `{userId}`; la
   implementación usa el Firebase UID (bitácora LOG-0036). Verificar que
   ambos lados hablan del mismo identificador antes de consumirlo desde el
   front.

## Estado del frontend (consumo real)

| Llamada | Existe en backend | En contrato |
|---|---|---|
| `POST /auth/register` | Sí | Sí |
| `POST /auth/sessions` | Sí | Sí |
| `DELETE /auth/sessions/current` | Sí | Sí |
| `GET /users/me` | Sí | Sí |

El front no consume aún semilleros, miembros ni roles. El contenido público usa
`dashboard-content.ts` estático, no `/content/pages`.

## Tareas siguientes sugeridas (una por tarea)

1. Actualizar `endpoints.md` contra las rutas reales de `handler.go`.
2. Resolver las rutas fuera de contrato: `/admin/members`, `/find/user`,
   `/users/roles` (incorporar, eliminar o marcar como alias documentado).
3. Implementar `GET /seedbeds/{id}/space` (único hueco del módulo casi completo).
4. A partir de ahí, seguir el orden sugerido del contrato §5.
