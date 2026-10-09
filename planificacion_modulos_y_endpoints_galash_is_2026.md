# Planificación de módulos y endpoints — GALASH IS 2026

## 1. Propósito

Esta propuesta organiza las historias de usuario en módulos funcionales y define un catálogo inicial de servicios HTTP (API REST). Los endpoints son una base para estimar y diseñar la primera versión; no representan una implementación definitiva.

## 2. Convenciones de la API

- Prefijo sugerido: `/api/v1`.
- Formato de intercambio: JSON; cargas de archivos mediante `multipart/form-data`.
- Las operaciones protegidas validan sesión y permisos en el backend. Ocultar una acción en la interfaz no reemplaza esa validación.
- Las listas aceptan paginación (`page`, `pageSize`) y filtros documentados por módulo.
- Las fechas se envían en formato ISO 8601 y se almacenan con zona horaria definida por el sistema.
- Se recomienda sesión mediante cookie segura `HttpOnly`, `Secure` y `SameSite`; las credenciales y tokens nunca se exponen en registros.
- Los cambios de estado relevantes conservan actor, fecha e historial. Desactivar u ocultar no implica borrar información histórica.
- **Prioridad:** Must = primera versión; Should = versión posterior; Could = mejora deseable.
- **Acceso:** Público, autenticado o restringido a los roles/permisos indicados. Los nombres finales de roles deben configurarse en el módulo de permisos.

## 3. Módulos funcionales

### Módulo 1. Autenticación y sesión

**Objetivo:** controlar el inicio y cierre de sesión y la recuperación segura de acceso.

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `POST` | `/auth/sessions` | Valida credenciales de una cuenta activa y crea la sesión. Ante credenciales inválidas devuelve un mensaje genérico. | Must / Público | HU-AUTH-01 |
| `DELETE` | `/auth/sessions/current` | Invalida la sesión actual. | Must / Autenticado | HU-AUTH-02 |
| `GET` | `/users/me` | Devuelve el perfil mínimo y los roles y permisos efectivos del usuario autenticado. | Must / Autenticado | HU-AUTH-01, HU-TR-02 |
| `POST` | `/auth/password-recovery-requests` | Solicita recuperación por correo; la respuesta no revela si el correo está registrado. | Should / Público | HU-AUTH-03 |
| `POST` | `/auth/password-resets` | Cambia la contraseña con un token vigente, de un solo uso. | Should / Público con token | HU-AUTH-03 |

**Reglas clave:** las contraseñas se almacenan con hash seguro; las sesiones expiran y se revocan al cerrar sesión. El límite de asignación del rol Administrador se aplica al asignar roles, en el módulo 11.

### Módulo 2. Solicitudes de registro de estudiantes

**Objetivo:** recibir solicitudes, permitir el seguimiento seguro del solicitante y soportar la revisión administrativa con trazabilidad.

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `POST` | `/registration-requests` | Registra una solicitud con sus datos, semillero de interés opcional y estado inicial `Radicada`. Devuelve identificador y mecanismo seguro de seguimiento. | Must / Público | HU-AUTH-04 |
| `GET` | `/registration-requests/{id}/status` | Consulta estado, fecha de actualización y observaciones autorizadas. Requiere sesión del solicitante o credencial de seguimiento segura. | Must / Solicitante | HU-AUTH-05 |
| `GET` | `/admin/registration-requests` | Lista solicitudes con filtros por estado, fecha, semillero y responsable. | Must / Administrador o permiso de revisión | HU-AUTH-06 |
| `GET` | `/admin/registration-requests/{id}` | Consulta detalle e historial de cambios de una solicitud. | Must / Administrador o permiso de revisión | HU-AUTH-06 |
| `PATCH` | `/admin/registration-requests/{id}/status` | Actualiza estado, observaciones autorizadas y responsable; valida permisos y registra auditoría. | Must / Administrador o permiso de revisión | HU-AUTH-05, HU-AUTH-06 |

**Estados mínimos:** `Radicada`, `En revisión`, `Aprobada`, `Rechazada` y `Pendiente de información`. La pantalla administrativa puede actualizarse mediante consulta periódica; si se requiere actualización push, se puede añadir SSE/WebSocket sin cambiar el modelo de solicitudes.

### Módulo 3. Semilleros y membresías

**Objetivo:** administrar los semilleros, su información visible y el historial de integrantes vinculados.

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `GET` | `/seedbeds` | Lista semilleros publicados; admite búsqueda y filtros. Usuarios autorizados pueden consultar también ocultos o inactivos. | Must / Público; vista completa restringida | HU-SEM-02 |
| `POST` | `/seedbeds` | Crea un semillero validando nombre y responsable obligatorios. | Must / Administrador o coordinador | HU-SEM-01 |
| `GET` | `/seedbeds/{id}` | Consulta detalle respetando visibilidad de campos y permisos. | Must / Público o autorizado | HU-SEM-02 |
| `PATCH` | `/seedbeds/{id}` | Actualiza descripción, línea, responsable u otros datos configurables. | Must / Administrador o coordinador | HU-SEM-01 |
| `PATCH` | `/seedbeds/{id}/visibility` | Publica, oculta o marca inactivo el semillero sin eliminarlo. | Must / Administrador o coordinador | HU-SEM-01 |
| `GET` | `/seedbeds/{id}/space` | Devuelve la vista compuesta del semillero: propósito, responsables y resumen de actividades y proyectos, aplicando permisos a información privada. | Must / Público o integrante autorizado | HU-SEM-02 |
| `GET` | `/seedbeds/{id}/members` | Consulta integrantes actuales; usuarios autorizados pueden consultar historial. | Must / Público según visibilidad; historial restringido | HU-SEM-02, HU-SEM-03 |
| `POST` | `/seedbeds/{id}/members` | Vincula un usuario existente y asigna su rol dentro del semillero. | Must / Coordinador autorizado | HU-SEM-03 |
| `DELETE` | `/seedbeds/{id}/members/{userId}` | Registra fecha de retiro y quita acceso operativo, conservando historial. | Must / Coordinador autorizado | HU-SEM-03 |

### Módulo 4. Actividades, proyectos y propuestas de investigación

**Objetivo:** registrar el trabajo de los semilleros y consultar el estado y avance de las investigaciones.

#### Actividades

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `GET` | `/seedbeds/{seedbedId}/activities` | Lista actividades; permite filtrar por periodo y estado (próxima, en curso o finalizada). | Must / Público según visibilidad; detalle privado restringido | HU-SEM-02, HU-SEM-04 |
| `POST` | `/seedbeds/{seedbedId}/activities` | Registra actividad con responsables y evidencias opcionales. | Must / Coordinador o profesor autorizado | HU-SEM-04 |
| `GET` | `/activities/{id}` | Consulta el detalle de una actividad. | Must / Público según visibilidad | HU-SEM-04 |
| `PATCH` | `/activities/{id}` | Modifica datos, estado o responsables de una actividad. | Must / Coordinador o profesor autorizado | HU-SEM-04 |

#### Proyectos y avance

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `GET` | `/projects` | Lista proyectos filtrables por semillero, estado y periodo. | Must / Público según visibilidad; vista completa autenticada | HU-SEM-02, HU-SEM-05 |
| `POST` | `/projects` | Registra proyecto con título, resumen, línea, responsables, integrantes, fechas, estado y avance entre 0 y 100. | Must / Profesor o coordinador autorizado | HU-SEM-05 |
| `GET` | `/projects/{id}` | Consulta el proyecto y su avance según permisos. | Must / Público según visibilidad | HU-SEM-02, HU-SEM-07 |
| `PATCH` | `/projects/{id}` | Actualiza datos del proyecto y registra autor y fecha del cambio. | Must / Responsable autorizado | HU-SEM-05 |
| `PATCH` | `/projects/{id}/progress` | Actualiza el porcentaje de avance; valida rango 0–100 y autorización del responsable. | Must / Responsable autorizado | HU-SEM-05, HU-SEM-07 |
| `GET` | `/projects/progress` | Resume avance por proyecto, semillero o periodo. | Must / Integrante o coordinador autorizado | HU-SEM-07 |
| `GET` | `/projects/{id}/history` | Consulta actualizaciones relevantes del proyecto. | Must / Integrante autorizado | HU-SEM-05 |

**Estados mínimos de proyecto:** `Radicado`, `Activo` y `Finalizado`.

#### Propuestas (fase posterior)

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `GET` | `/research-proposals` | Lista propuestas y sus estados de revisión. | Should / Integrante autorizado | HU-SEM-06 |
| `POST` | `/research-proposals` | Registra título, problema, objetivos, metodología, responsables y anexos opcionales. | Should / Integrante autorizado | HU-SEM-06 |
| `GET` | `/research-proposals/{id}` | Consulta propuesta, observaciones e historial. | Should / Participante o revisor autorizado | HU-SEM-06 |
| `PATCH` | `/research-proposals/{id}` | Actualiza propuesta u observaciones de revisión conforme a permisos. | Should / Participante o revisor autorizado | HU-SEM-06 |
| `POST` | `/research-proposals/{id}/convert-to-project` | Convierte una propuesta aprobada en proyecto radicado conservando su historial. | Should / Revisor autorizado | HU-SEM-06 |

#### Reseñas de semillero (mejora deseable)

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `POST` | `/seedbeds/{seedbedId}/reviews` | Envía reseña con autor, fecha y semillero asociado; genera notificación interna al administrador. | Could / Usuario autenticado | HU-SEM-08 |
| `GET` | `/admin/seedbed-reviews` | Lista reseñas pendientes de moderación. | Could / Administrador | HU-SEM-08 |
| `PATCH` | `/admin/seedbed-reviews/{id}/moderation` | Publica, oculta o rechaza una reseña y registra la decisión. | Could / Administrador | HU-SEM-08 |

### Módulo 5. Integrantes y perfiles

**Objetivo:** mantener el directorio institucional y exponer solo la información pública de cada perfil.

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `GET` | `/members` | Busca y filtra integrantes por tipo, semillero o área; excluye perfiles inactivos de la vista pública. | Must / Público | HU-INT-02 |
| `POST` | `/members` | Crea perfil de estudiante o profesor y valida campos obligatorios. | Must / Administrador | HU-INT-01 |
| `GET` | `/members/{id}` | Consulta el perfil público o, con permiso, los campos internos. | Must / Público o autorizado | HU-INT-02 |
| `PATCH` | `/members/{id}` | Edita datos, asociación a semilleros, visibilidad y enlace CvLAC. | Must / Administrador | HU-INT-01, HU-INT-03 |
| `PATCH` | `/members/{id}/status` | Activa o desactiva perfil sin borrar historial relacionado. | Must / Administrador | HU-INT-01 |
| `GET` | `/members/{id}/publications` | Consulta artículos y publicaciones asociados al perfil cuando existan. | Must / Público según visibilidad | HU-INT-02 |

### Módulo 6. Eventos, talleres y convocatorias

**Objetivo:** administrar eventos y ofrecer una agenda pública filtrable.

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `GET` | `/events` | Lista eventos en formato de agenda; permite filtros por tipo, fecha y estado, y diferencia vencidos de próximos. | Must / Público | HU-EVT-03 |
| `POST` | `/events` | Crea evento en borrador con fecha, hora, organizador, lugar/enlace e imagen opcional. | Must / Administrador o coordinador autorizado | HU-EVT-01 |
| `GET` | `/events/{id}` | Consulta detalle e información de inscripción, si existe. | Must / Público si está publicado | HU-EVT-03 |
| `PATCH` | `/events/{id}` | Actualiza los datos del evento y registra fecha de modificación. | Must / Usuario con permiso | HU-EVT-02 |
| `POST` | `/events/{id}/publish` | Publica un evento en borrador. | Must / Usuario con permiso | HU-EVT-01, HU-EVT-02 |
| `POST` | `/events/{id}/cancel` | Cancela el evento conservando su información histórica. | Must / Usuario con permiso | HU-EVT-02 |
| `POST` | `/events/{id}/hide` | Oculta el evento sin borrarlo. | Must / Usuario con permiso | HU-EVT-02 |

### Módulo 7. Información institucional

**Objetivo:** gestionar páginas institucionales editables sin cambios de código.

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `GET` | `/content/pages` | Lista secciones institucionales publicadas (historia, objetivos, misión, visión y áreas). | Must / Público | HU-INFO-01 |
| `GET` | `/content/pages/{slug}` | Consulta una página publicada por identificador legible. | Must / Público | HU-INFO-01 |
| `GET` | `/admin/content/pages` | Lista borradores y páginas publicadas. | Must / Administrador | HU-INFO-02 |
| `POST` | `/admin/content/pages` | Crea página en borrador y registra autor. | Must / Administrador | HU-INFO-02 |
| `PATCH` | `/admin/content/pages/{id}` | Edita título y contenido; sanitiza formato permitido y registra actualización. | Must / Administrador | HU-INFO-02 |
| `POST` | `/admin/content/pages/{id}/publish` | Publica la versión vigente de una página. | Must / Administrador | HU-INFO-02 |

### Módulo 8. Tablero de comunicados y lectura

**Objetivo:** publicar directrices con adjuntos y ofrecer trazabilidad de lectura por profesor.

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `GET` | `/announcements` | Lista comunicados visibles para el usuario autenticado e indica si ya confirmó lectura. | Must / Profesor o encargado autorizado | HU-COM-02 |
| `POST` | `/announcements` | Crea comunicado con título, descripción y referencias a archivos adjuntos permitidos. | Must / Director autorizado | HU-COM-01 |
| `GET` | `/announcements/{id}` | Consulta comunicado y descarga autorizada de sus adjuntos. | Must / Profesor o encargado autorizado | HU-COM-01, HU-COM-02 |
| `PATCH` | `/announcements/{id}` | Edita comunicado conforme a permisos y estado de publicación. | Must / Director autorizado | HU-COM-01 |
| `POST` | `/announcements/{id}/publish` | Publica el comunicado para sus destinatarios. | Must / Director autorizado | HU-COM-01 |
| `POST` | `/announcements/{id}/read-confirmations` | Confirma lectura de forma idempotente y registra fecha y hora. | Must / Profesor o encargado autorizado | HU-COM-02 |
| `GET` | `/announcements/{id}/read-status` | Muestra al director el estado Leído/Pendiente por profesor. | Must / Director autorizado | HU-COM-01 |

### Módulo 9. Cronograma interno

**Objetivo:** consultar y agendar reuniones, tutorías e hitos internos según semillero y participantes.

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `GET` | `/internal-calendar/events` | Consulta eventos internos por rango de fechas, vista semanal/mensual y semillero. Los estudiantes solo ven los de su semillero. | Must / Usuario autenticado | HU-CRO-01 |
| `POST` | `/internal-calendar/events` | Crea reunión o hito con título, fechas, horas y participantes. | Must / Profesor o director | HU-CRO-01 |
| `GET` | `/internal-calendar/events/{id}` | Consulta detalle y participantes según permisos. | Must / Usuario autorizado | HU-CRO-01 |
| `PATCH` | `/internal-calendar/events/{id}` | Actualiza evento interno o participantes. | Must / Profesor o director autorizado | HU-CRO-01 |
| `DELETE` | `/internal-calendar/events/{id}` | Cancela/elimina lógicamente un evento interno conservando trazabilidad. | Must / Profesor o director autorizado | HU-CRO-01 |

### Módulo 10. Enlaces oficiales

**Objetivo:** centralizar enlaces externos por categoría y mantener su vigencia sin perder historial.

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `GET` | `/links` | Lista enlaces visibles; filtra por categoría y orden. | Must / Público | HU-LIN-01 |
| `POST` | `/links` | Crea enlace y valida esquema/formato de URL. | Must / Administrador | HU-LIN-02 |
| `PATCH` | `/links/{id}` | Actualiza nombre, descripción, destino, categoría, visibilidad u orden; registra autor y fecha. | Must / Administrador | HU-LIN-02 |
| `PATCH` | `/links/{id}/status` | Activa, desactiva u oculta un enlace sin borrar historial. | Must / Administrador | HU-LIN-01, HU-LIN-02 |

### Módulo 11. Roles, permisos y paneles

**Objetivo:** aplicar autorización en backend y entregar al frontend el contexto necesario para adaptar el panel.

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `GET` | `/roles` | Consulta roles configurados. | Must / Administrador | HU-TR-01 |
| `GET` | `/permissions` | Consulta permisos disponibles para administrar roles. | Must / Administrador | HU-TR-01 |
| `GET` | `/roles/{roleId}/permissions` | Consulta los permisos asociados a un rol. | Must / Administrador | HU-TR-01 |
| `PUT` | `/roles/{roleId}/permissions` | Asigna o retira permisos del rol; comprueba autorización y evita que el actor se conceda privilegios a sí mismo. | Must / Administrador autorizado | HU-TR-01 |
| `GET` | `/users/{userId}/roles` | Consulta roles asignados a un usuario. | Must / Administrador o permiso delegado | HU-TR-01 |
| `PUT` | `/users/{userId}/roles` | Reemplaza/asigna roles con validación de permisos, prevención de autoasignación y límite de máximo dos usuarios con rol Administrador. | Must / Administrador autorizado | HU-AUTH-01, HU-TR-01 |
| `GET` | `/users/me/dashboard` | Devuelve resumen de tareas y accesos permitidos según rol y contexto. | Must / Autenticado | HU-TR-02 |

**Regla:** todos los endpoints de escritura deben comprobar el permiso requerido y el contexto (por ejemplo, pertenencia al semillero). La interfaz solo consume los permisos; no los concede.

### Módulo 12. Auditoría

**Objetivo:** conservar una bitácora inmutable de acciones críticas y permitir su consulta autorizada.

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `GET` | `/audit-logs` | Consulta bitácora filtrable por actor, acción, entidad y periodo. | Must / Administrador con permiso de auditoría | HU-TR-03 |

La bitácora se genera internamente al modificar roles, solicitudes, semilleros, proyectos, eventos, comunicados y contenidos. Registra actor, acción, entidad, fecha, resultado y origen disponible. No se expone endpoint de edición o eliminación para usuarios operativos.

### Módulo 13. Archivos adjuntos (servicio transversal)

**Objetivo:** reutilizar una forma segura de cargar y consultar archivos de comunicados, evidencias, propuestas y eventos.

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `POST` | `/files` | Sube un archivo, valida tamaño y tipo permitido y devuelve un identificador para asociarlo a una entidad. | Must / Usuario autorizado según contexto | HU-COM-01, HU-SEM-04, HU-SEM-06, HU-EVT-01 |
| `GET` | `/files/{id}` | Descarga o consulta metadatos del archivo tras validar permisos de acceso a la entidad asociada. | Must / Autorizado | HU-COM-01, HU-SEM-04, HU-SEM-06, HU-EVT-01 |

Para comunicados se permiten PDF, Word y Excel, según la historia de usuario. Los archivos privados no deben quedar disponibles mediante URL pública predecible.

## 4. Requisitos transversales y fuera del catálogo de endpoints

- **Seguridad (HU-NF-04):** HTTPS en despliegues; validación y sanitización en backend; control de sesión; mínimos privilegios; datos personales filtrados por autorización. Aplican a todos los módulos.
- **Auditoría:** registrar operaciones críticas de manera uniforme, sin permitir alteración desde la API operativa.
- **Interfaz e identidad visual (HU-NF-01, HU-NF-02, HU-NF-03):** identidad institucional, accesibilidad, diseño responsive, estados de carga/éxito/error y consistencia visual son responsabilidades de frontend/diseño; no requieren endpoints específicos. La API debe entregar errores de validación claros y consistentes.
- **Notificaciones:** avisos de revisión de reseñas y otros eventos pueden generarse internamente. La primera versión no requiere una API pública de notificaciones salvo que se decida construir una bandeja de notificaciones.
- **Errores:** usar códigos HTTP coherentes y un formato común de error, sin revelar credenciales, existencia de cuentas ni datos no autorizados.

## 5. Orden sugerido de implementación

1. **Base de Must:** autenticación/sesión, solicitudes de registro, roles/permisos, auditoría y archivos.
2. **Operación académica Must:** semilleros/membresías, integrantes/perfiles, actividades/proyectos, eventos, comunicados, cronograma, contenidos institucionales y enlaces.
3. **Should:** recuperación de contraseña y propuestas de investigación.
4. **Could:** reseñas y moderación de semilleros.

El orden es una sugerencia de dependencias: permisos y auditoría deben estar disponibles desde el inicio para proteger los módulos administrativos.
