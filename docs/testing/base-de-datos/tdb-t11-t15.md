# TDB-T11, T13, T14 y T15 — Resultados (16/11/2026)

Base efímera PostgreSQL 16 (`postgres:16-alpine`) con datos sintéticos
(`database/seed/synthetic.sql`). No se usaron datos reales.

## TDB-T11 — Concurrencia (R116)

| Caso | Resultado |
|---|---|
| Dos sesiones cambian la misma solicitud a `En revisión` a la vez, 20 rondas × 3 repeticiones (`TestConcurrentRegistrationUpdatesHaveOneWinner`, T189) | **Pasa.** Siempre gana una sola sesión; la otra recibe `ErrInvalidRegistrationTransition`. El historial tiene exactamente una fila nueva. `UPDATE … WHERE estado=<leído>` en la misma transacción evita la actualización perdida. |
| Dos editores cargan el mismo proyecto; A cambia el título y B, con la copia vieja, cambia el resumen (`TestStaleProjectSaveDoesNotSilentlyRevertOtherEdit`, T191) | **Falla (D039).** El guardado de B tiene éxito y revierte en silencio el título de A. `UpdateProject` sobrescribe la fila completa sin control de versión. Los eventos (`UPDATE eventos SET …`) siguen el mismo patrón. |

## TDB-T13 — Migración (R115)

Script reproducible: `evidence/R115/tdb-t13-t14.sh`; salida en `tdb-t13-t14.txt`.

1. Se aplica `init.sql` de la versión anterior (`cac60ba`, 10/11/2026) más
   `synthetic.sql`, y se dejan 2 proyectos en `Radicado`, un estado válido en
   esa versión.
2. Volver a ejecutar el `init.sql` actual sobre esa base **falla** a los 51 ms
   con `relation "usuarios" already exists`. Es lo esperado: según
   `database/README.md`, `init.sql` solo inicializa volúmenes nuevos, y las
   bases existentes se actualizan con `database/migrations/`.
3. La migración incremental `017_project_statuses.sql` termina sin errores en
   55 ms. Conserva los conteos de `usuarios` (14), `semilleros` (4),
   `semillero_miembros` (13), `solicitudes_registro` (6), `proyectos` (12),
   `eventos` (4) y `audit_logs` (0). Convierte `Radicado` en `Activo`
   (quedan Activo=9 y Finalizado=3), agrega 2 filas a `proyecto_historial` y
   deja validado `proyectos_estado_check`. La base queda con 31 tablas.
4. **Reversión:** restaurar el `pg_dump -Fc` tomado antes de la migración en
   una base nueva tarda 284 ms y devuelve los mismos conteos, con los 2
   proyectos otra vez en `Radicado`. Procedimiento: detener la API, crear una
   base vacía, ejecutar `pg_restore --exit-on-error` del respaldo previo,
   apuntar `DATABASE_URL` a esa base y desplegar la versión anterior del
   backend.

Decisión sugerida: migrar con `migrations/NNN_*.sql`, siempre tras un
`pg_dump`, y nunca re-aplicar `init.sql` sobre una base con datos.

## TDB-T14 — Respaldo y restauración (R115)

| Medida | Valor |
|---|---|
| Versión | PostgreSQL 16.15 |
| `pg_dump -Fc` | 129 ms, 78 462 bytes |
| `pg_restore` en base nueva | 284 ms, exit 0 |
| Conteo de tablas críticas restauradas | idéntico al previo (8 tablas comparadas) |

## TDB-T15 — Protección de datos (R116)

| Control | Técnica | Hallazgo | Riesgo |
|---|---|---|---|
| Contraseñas | Revisión de código | `password_hash` solo se escribe con `bcrypt` (coste por defecto, 10) en el restablecimiento heredado. El acceso normal es con Firebase, sin contraseña local. | Bajo |
| Tokens | Esquema | `sesiones`, `recuperaciones_password` y `solicitudes_registro` guardan solo el SHA-256 del token. La cookie es `HttpOnly`, `Secure` y `SameSite=Lax`, y dura 24 h. | Bajo |
| Datos personales en respuestas públicas | Prueba live sin sesión (`TestLivePublicResponsesHidePersonalData`, T194) | `GET /members`, `/members?q=`, `/members/{id}`, `/seedbeds/{id}`, `/space` y `/members` del semillero no exponen `dni`, correo, teléfono ni `firebase_uid`. | Pasa |
| Logs | Revisión de código | Hay 6 `log.Printf` y ninguno imprime cuerpos, tokens ni datos personales. Los errores de PostgreSQL se registran con `%v` (sin `DETAIL`). | Bajo |
| Privilegios de BD | Configuración | `DATABASE_URL` de ejemplo usa el mismo usuario que `POSTGRES_USER`, y la imagen oficial lo crea como **superusuario**. La API no necesita DDL ni `SUPERUSER` en ejecución (D040). | Medio |
| Credenciales en el repositorio | `git ls-files` y `.gitignore` | `deploy/auth/firebase-service-account.json` existe localmente, pero está ignorado y nunca se versionó. Los seeds con datos reales también están ignorados. | Bajo |
