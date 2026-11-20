# TDB-T01 — Revisión del modelo de datos (reemplaza §11.1 del informe)

Corrida: `R110` (16/11/2026). Evidencia: `docs/testing/evidence/R110/revision-esquema.sql`
(consultas al catálogo, reproducibles) y `revision-esquema.txt` (salida).
Base: PostgreSQL 16 efímero (`postgres:16-alpine`) inicializado con `database/init.sql`.
Sin datos reales; las sondas se ejecutan dentro de una transacción revertida.

## 11.1 Revisión del modelo de datos

La versión anterior de esta sección evaluó un modelo documental en Firebase. Ese
modelo ya no aplica: Firebase solo autentica (verifica el token de Google) y toda
la información del sistema vive en PostgreSQL 16, definido en `database/init.sql`.
La revisión se rehízo sobre el catálogo de la base creada con ese script.

**Alcance.** 31 tablas base: usuarios y sesiones (`usuarios`, `sesiones`,
`recuperaciones_password`, `perfil_investigador`), semilleros (`semilleros`,
`semillero_miembros`, `semillero_resenas`, `lineas_investigacion`,
`colaboradores_externos`), solicitudes (`solicitudes_registro`,
`seguimiento_solicitud`), investigación (`proyectos`, `proyecto_historial`,
`actividades`, `propuestas_investigacion`, `propuesta_historial`,
`publicaciones`), comunicación (`eventos`, `comunicados`, `comunicados_lecturas`,
`noticias`, `notificaciones`, `contenido_paginas`), agenda interna
(`cronograma_interno`, `cronograma_interno_participantes`,
`cronograma_interno_historial`, `cronograma_reuniones`), enlaces oficiales
(`enlaces_oficiales`, `enlaces_oficiales_historial`), archivos
(`archivos_adjuntos`) y auditoría (`audit_logs`).

**Unicidad.** Hay restricciones únicas en `usuarios.firebase_uid`,
`usuarios.dni`, `usuarios.codigo_estudiantil`, `semilleros.nombre`,
`lineas_investigacion.nombre`, `contenido_paginas.slug`,
`perfil_investigador.usuario_id` y en los `token_hash` de `sesiones`,
`recuperaciones_password` y `solicitudes_registro`. Una segunda cuenta con el
mismo `dni` se rechaza (código 23505), y la API lo traduce a 409.
**Hallazgo:** `usuarios.email` es `NOT NULL` pero solo tiene un índice no único
(`usuarios_email_idx`). La base acepta dos cuentas con el mismo correo, también
si difiere en mayúsculas. La identidad real es `firebase_uid`, pero el correo se
usa para asociar integrantes importados (defecto D033).

**Integridad referencial.** 48 claves foráneas. 42 usan `NO ACTION`, que
impide borrar un registro referenciado. Así, un semillero o un usuario con
historial no se elimina físicamente: se desactiva (`estado`, `visibilidad`,
`retirado_en`). Seis usan `ON DELETE CASCADE`, solo en tablas dependientes
que no tienen sentido sin su padre: `proyecto_historial`, `propuesta_historial`,
`comunicados_lecturas` (por comunicado y por usuario),
`cronograma_interno_participantes` y `cronograma_interno_historial`.
No hay `SET NULL`. Observación de rendimiento: 39 de las 48 FKs no tienen un
índice que empiece por la columna referenciante (p. ej. `sesiones.usuario_id`,
`proyectos.semillero_id`, `eventos.semillero_id`). Con el volumen actual
(cientos de filas) no afecta; se recomienda indexar las que se filtran en
listados.

**Reglas de dominio (CHECK).** Existen CHECKs en `proyectos.estado`
(`Activo`/`Finalizado`), `proyectos.avance` (0–100), `archivos_adjuntos.tamano`
(1 B–10 MiB), `cronograma_interno.tipo` y `.estado`, `enlaces_oficiales.estado`
y `.orden ≥ 0`. **Hallazgo:** 22 columnas de estado, rol o tipo son texto libre
sin CHECK; entre ellas `usuarios.rol`, `usuarios.estado`,
`solicitudes_registro.estado`, `semilleros.estado` y `.visibilidad`,
`eventos.estado` y `.tipo`, `comunicados.estado`, `contenido_paginas.estado`,
`semillero_resenas.estado` y `actividades.estado`. La sonda insertó
`rol='rol-inexistente'` sin error. Hoy la validación depende solo de la capa de
aplicación (defecto D034).

**Campos de auditoría.** Las tablas de negocio tienen fecha de creación
(`creado_en`, `fecha_radicacion`, `ingreso_en`, `fecha_registro`, etc.), y las
editables principales tienen `actualizado_en` (`semilleros`, `proyectos`,
`actividades`, `eventos`, `comunicados`, `contenido_paginas`,
`solicitudes_registro`, `enlaces_oficiales`, `cronograma_interno`). Los cambios
de estado relevantes quedan en tablas de historial (`proyecto_historial`,
`propuesta_historial`, `seguimiento_solicitud`, `cronograma_interno_historial`,
`enlaces_oficiales_historial`; esta última se llena con un trigger
`AFTER INSERT/UPDATE`). `audit_logs` (actor, acción, entidad, resultado,
origen, `created_at`) es inmutable: el trigger `audit_logs_immutable_trigger`
rechaza `UPDATE` y `DELETE` con el error «audit_logs es inmutable»
(verificado). `usuarios` no tiene `actualizado_en`.

**Sesiones y credenciales.** `sesiones` guarda solo el hash del token
(`token_hash`, único) y `expira_en NOT NULL`, con revocación lógica
(`revocada_en`). `usuarios.password_hash` admite nulos porque la autenticación
la hace Firebase; ver TDB-T15.

**Conclusión.** El modelo relacional es coherente con los módulos del contrato:
las relaciones están protegidas por FKs, el borrado es lógico y la auditoría es
inmutable. Quedan dos defectos por decidir con el equipo (cambian datos
existentes): unicidad de `usuarios.email` (D033) y CHECKs de estados y roles
(D034). Antes de añadir esas restricciones hay que depurar duplicados y valores
fuera de catálogo en la base real.
