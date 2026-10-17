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
- **Roles del sistema:** `Visitante`, `Estudiante`, `Docente` y `Admin`. No se crearán roles adicionales para representar coordinadores, directores o responsables.
- **Visitante:** consulta la información pública y puede enviar solicitudes de registro.
- **Estudiante:** consulta los semilleros y la información autorizada de sus actividades, proyectos y membresías.
- **Docente:** puede ser responsable de semilleros y gestionar la información académica dentro del alcance autorizado.
- **Admin:** administra globalmente usuarios, semilleros, contenidos, permisos y auditoría.
- **Acceso:** cada endpoint indica uno o más de estos roles. Además del rol, el backend debe validar el contexto, por ejemplo, que el `Docente` sea responsable del semillero.

## 3. Módulos funcionales

### Módulo 1. Autenticación y sesión

**Objetivo:** controlar el inicio y cierre de sesión y la recuperación segura de acceso.

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `POST` | `/auth/sessions` | Valida credenciales de una cuenta activa y crea la sesión. Ante credenciales inválidas devuelve un mensaje genérico. | Must / Visitante | HU-AUTH-01 |
| `DELETE` | `/auth/sessions/current` | Invalida la sesión actual. | Must / Estudiante, Docente o Admin | HU-AUTH-02 |
| `GET` | `/users/me` | Devuelve el perfil mínimo y los roles y permisos efectivos del usuario autenticado. | Must / Estudiante, Docente o Admin | HU-AUTH-01, HU-TR-02 |
| `POST` | `/auth/password-recovery-requests` | Solicita recuperación por correo; la respuesta no revela si el correo está registrado. | Should / Visitante | HU-AUTH-03 |
| `POST` | `/auth/password-resets` | Cambia la contraseña con un token vigente, de un solo uso. | Should / Visitante con token | HU-AUTH-03 |

**Reglas clave:** las contraseñas se almacenan con hash seguro; las sesiones expiran y se revocan al cerrar sesión. Solo `Admin` puede asignar o retirar los cuatro roles permitidos.

### Módulo 2. Semilleros (entidad central)

**Objetivo:** crear, publicar y administrar los semilleros, que son la entidad central de la operación académica. Las actividades, proyectos, membresías y solicitudes relacionadas deben referenciar `seedbedId` y no duplicar los datos del semillero.

**Datos mínimos de `Semillero`:** `id`, `name`, `description`, `researchLine`, `objective`, `responsibleTeacherId`, `contactEmail`, `status`, `visibility`, `createdAt` y `updatedAt`. El docente responsable debe ser un usuario activo con rol `Docente`.

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `GET` | `/seedbeds` | Lista semilleros publicados; admite búsqueda, paginación y filtros por línea de investigación. `Admin` puede consultar también ocultos o inactivos. | Must / Visitante; vista completa para `Admin` | HU-SEM-02 |
| `POST` | `/seedbeds` | Crea un semillero en estado `Borrador`. Recibe nombre, descripción, línea, objetivo, docente responsable y contacto; valida nombre único y responsable con rol `Docente`. Devuelve `201` e identificador. | Must / `Docente` o `Admin` | HU-SEM-01 |
| `GET` | `/seedbeds/{id}` | Consulta el detalle del semillero respetando su visibilidad. | Must / Visitante o rol autorizado | HU-SEM-02 |
| `PATCH` | `/seedbeds/{id}` | Actualiza los datos del semillero; registra actor y fecha del cambio. | Must / Docente responsable o `Admin` | HU-SEM-01 |
| `PATCH` | `/seedbeds/{id}/visibility` | Publica, oculta o marca inactivo el semillero sin eliminarlo. Estados mínimos: `Borrador`, `Publicado`, `Oculto` e `Inactivo`. | Must / Docente responsable o `Admin` | HU-SEM-01 |
| `GET` | `/seedbeds/{id}/space` | Devuelve la vista compuesta del semillero: propósito, responsable y resumen de actividades y proyectos, aplicando permisos a información privada. | Must / Visitante o rol autorizado | HU-SEM-02 |
| `GET` | `/seedbeds/{id}/members` | Consulta integrantes actuales; el historial de retiros queda restringido. | Must / Visitante según visibilidad; historial para `Docente` responsable o `Admin` | HU-SEM-02, HU-SEM-03 |
| `POST` | `/seedbeds/{id}/members` | Vincula un usuario existente con rol `Estudiante` o `Docente`; no crea roles internos adicionales. | Must / Docente responsable o `Admin` | HU-SEM-03 |
| `DELETE` | `/seedbeds/{id}/members/{userId}` | Registra la fecha de retiro y quita el acceso operativo, conservando el historial. | Must / Docente responsable o `Admin` | HU-SEM-03 |

**Contrato de alta:** el servicio debe rechazar nombres duplicados, responsables inexistentes o usuarios que no tengan rol `Docente`. La respuesta debe incluir el `id`, el estado inicial y el `responsibleTeacherId` para que los demás módulos puedan establecer la relación con el semillero.

### Módulo 3. Solicitudes de registro de estudiantes

**Objetivo:** recibir solicitudes, permitir el seguimiento seguro del solicitante y soportar la revisión administrativa con trazabilidad.

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `POST` | `/registration-requests` | Registra una solicitud con sus datos y un `seedbedId` opcional validado contra un semillero existente; estado inicial `Radicada`. Devuelve identificador y mecanismo seguro de seguimiento. | Must / Visitante | HU-AUTH-04 |
| `GET` | `/registration-requests/{id}/status` | Consulta estado, fecha de actualización y observaciones autorizadas. Requiere credencial de seguimiento segura o sesión del solicitante. | Must / Visitante o `Estudiante` autorizado | HU-AUTH-05 |
| `GET` | `/admin/registration-requests` | Lista solicitudes con filtros por estado, fecha, semillero y responsable. | Must / `Admin` | HU-AUTH-06 |
| `GET` | `/admin/registration-requests/{id}` | Consulta detalle e historial de cambios de una solicitud. | Must / `Admin` | HU-AUTH-06 |
| `PATCH` | `/admin/registration-requests/{id}/status` | Actualiza estado, observaciones autorizadas y responsable; valida permisos y registra auditoría. | Must / `Admin` | HU-AUTH-05, HU-AUTH-06 |

**Estados mínimos:** `Radicada`, `En revisión`, `Aprobada`, `Rechazada` y `Pendiente de información`. La pantalla administrativa puede actualizarse mediante consulta periódica; si se requiere actualización push, se puede añadir SSE/WebSocket sin cambiar el modelo de solicitudes.

### Módulo 4. Actividades, proyectos y propuestas de investigación

**Objetivo:** registrar el trabajo de los semilleros y consultar el estado y avance de las investigaciones.

#### Actividades

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `GET` | `/seedbeds/{seedbedId}/activities` | Lista actividades; permite filtrar por periodo y estado (próxima, en curso o finalizada). | Must / Visitante según visibilidad; detalle privado restringido | HU-SEM-02, HU-SEM-04 |
| `POST` | `/seedbeds/{seedbedId}/activities` | Registra actividad con responsables y evidencias opcionales. | Must / Docente responsable o Admin | HU-SEM-04 |
| `GET` | `/activities/{id}` | Consulta el detalle de una actividad. | Must / Visitante según visibilidad | HU-SEM-04 |
| `PATCH` | `/activities/{id}` | Modifica datos, estado o responsables de una actividad. | Must / Docente responsable o Admin | HU-SEM-04 |

#### Proyectos y avance

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `GET` | `/projects` | Lista proyectos filtrables por semillero, estado y periodo. | Must / Visitante según visibilidad; vista completa para el rol autorizado | HU-SEM-02, HU-SEM-05 |
| `POST` | `/projects` | Registra proyecto con título, resumen, línea, responsables, integrantes, fechas, estado y avance entre 0 y 100. | Must / Docente responsable o Admin | HU-SEM-05 |
| `GET` | `/projects/{id}` | Consulta el proyecto y su avance según permisos. | Must / Visitante según visibilidad | HU-SEM-02, HU-SEM-07 |
| `PATCH` | `/projects/{id}` | Actualiza datos del proyecto y registra autor y fecha del cambio. | Must / Docente responsable o Admin | HU-SEM-05 |
| `PATCH` | `/projects/{id}/progress` | Actualiza el porcentaje de avance; valida rango 0–100 y autorización del responsable. | Must / Docente responsable o Admin | HU-SEM-05, HU-SEM-07 |
| `GET` | `/projects/progress` | Resume avance por proyecto, semillero o periodo. | Must / Estudiante, Docente o Admin autorizado | HU-SEM-07 |
| `GET` | `/projects/{id}/history` | Consulta actualizaciones relevantes del proyecto. | Must / Estudiante, Docente o Admin autorizado | HU-SEM-05 |

**Estados mínimos de proyecto:** `Radicado`, `Activo` y `Finalizado`.

#### Propuestas (fase posterior)

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `GET` | `/research-proposals` | Lista propuestas y sus estados de revisión. | Should / Estudiante, Docente o Admin autorizado | HU-SEM-06 |
| `POST` | `/research-proposals` | Registra título, problema, objetivos, metodología, responsables y anexos opcionales. | Should / Estudiante o Docente autorizado | HU-SEM-06 |
| `GET` | `/research-proposals/{id}` | Consulta propuesta, observaciones e historial. | Should / Estudiante, Docente o Admin autorizado | HU-SEM-06 |
| `PATCH` | `/research-proposals/{id}` | Actualiza propuesta u observaciones de revisión conforme a permisos. | Should / Estudiante, Docente o Admin autorizado | HU-SEM-06 |
| `POST` | `/research-proposals/{id}/convert-to-project` | Convierte una propuesta aprobada en proyecto radicado conservando su historial. | Should / Docente o Admin autorizado | HU-SEM-06 |

#### Reseñas de semillero (mejora deseable)

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `POST` | `/seedbeds/{seedbedId}/reviews` | Envía reseña con autor, fecha y semillero asociado; genera notificación interna a `Admin`. | Could / Estudiante, Docente o Admin | HU-SEM-08 |
| `GET` | `/admin/seedbed-reviews` | Lista reseñas pendientes de moderación. | Could / Admin | HU-SEM-08 |
| `PATCH` | `/admin/seedbed-reviews/{id}/moderation` | Publica, oculta o rechaza una reseña y registra la decisión. | Could / Admin | HU-SEM-08 |

### Módulo 5. Integrantes y perfiles

**Objetivo:** mantener el directorio institucional y exponer solo la información pública de cada perfil.

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `GET` | `/members` | Busca y filtra integrantes por tipo, semillero o área; excluye perfiles inactivos de la vista pública. | Must / Visitante | HU-INT-02 |
| `POST` | `/members` | Crea perfil de `Estudiante` o `Docente` y valida campos obligatorios. | Must / Admin | HU-INT-01 |
| `GET` | `/members/{id}` | Consulta el perfil público o, con permiso, los campos internos. | Must / Visitante o rol autorizado | HU-INT-02 |
| `PATCH` | `/members/{id}` | Edita datos, asociación a semilleros, visibilidad y enlace CvLAC. | Must / Admin | HU-INT-01, HU-INT-03 |
| `PATCH` | `/members/{id}/status` | Activa o desactiva perfil sin borrar historial relacionado. | Must / Admin | HU-INT-01 |
| `GET` | `/members/{id}/publications` | Consulta artículos y publicaciones asociados al perfil cuando existan. | Must / Visitante según visibilidad | HU-INT-02 |

### Módulo 6. Eventos, talleres y convocatorias

**Objetivo:** administrar eventos y ofrecer una agenda pública filtrable.

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `GET` | `/events` | Lista eventos en formato de agenda; permite filtros por tipo, fecha y estado, y diferencia vencidos de próximos. | Must / Visitante | HU-EVT-03 |
| `POST` | `/events` | Crea evento en borrador con fecha, hora, organizador, lugar/enlace e imagen opcional. | Must / Docente o Admin autorizado | HU-EVT-01 |
| `GET` | `/events/{id}` | Consulta detalle e información de inscripción, si existe. | Must / Visitante si está publicado | HU-EVT-03 |
| `PATCH` | `/events/{id}` | Actualiza los datos del evento y registra fecha de modificación. | Must / Docente responsable o Admin | HU-EVT-02 |
| `POST` | `/events/{id}/publish` | Publica un evento en borrador. | Must / Docente responsable o Admin | HU-EVT-01, HU-EVT-02 |
| `POST` | `/events/{id}/cancel` | Cancela el evento conservando su información histórica. | Must / Docente responsable o Admin | HU-EVT-02 |
| `POST` | `/events/{id}/hide` | Oculta el evento sin borrarlo. | Must / Docente responsable o Admin | HU-EVT-02 |

### Módulo 7. Información institucional

**Objetivo:** gestionar páginas institucionales editables sin cambios de código.

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `GET` | `/content/pages` | Lista secciones institucionales publicadas (historia, objetivos, misión, visión y áreas). | Must / Visitante | HU-INFO-01 |
| `GET` | `/content/pages/{slug}` | Consulta una página publicada por identificador legible. | Must / Visitante | HU-INFO-01 |
| `GET` | `/admin/content/pages` | Lista borradores y páginas publicadas. | Must / Admin | HU-INFO-02 |
| `POST` | `/admin/content/pages` | Crea página en borrador y registra autor. | Must / Admin | HU-INFO-02 |
| `PATCH` | `/admin/content/pages/{id}` | Edita título y contenido; sanitiza formato permitido y registra actualización. | Must / Admin | HU-INFO-02 |
| `POST` | `/admin/content/pages/{id}/publish` | Publica la versión vigente de una página. | Must / Admin | HU-INFO-02 |

### Módulo 8. Tablero de comunicados y lectura

**Objetivo:** publicar directrices con adjuntos y ofrecer trazabilidad de lectura por `Docente`.

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `GET` | `/announcements` | Lista comunicados visibles para el usuario autenticado e indica si ya confirmó lectura. | Must / Docente o Admin autorizado | HU-COM-02 |
| `POST` | `/announcements` | Crea comunicado con título, descripción y referencias a archivos adjuntos permitidos. | Must / Docente o Admin autorizado | HU-COM-01 |
| `GET` | `/announcements/{id}` | Consulta comunicado y descarga autorizada de sus adjuntos. | Must / Docente o Admin autorizado | HU-COM-01, HU-COM-02 |
| `PATCH` | `/announcements/{id}` | Edita comunicado conforme a permisos y estado de publicación. | Must / Docente o Admin autorizado | HU-COM-01 |
| `POST` | `/announcements/{id}/publish` | Publica el comunicado para sus destinatarios. | Must / Docente o Admin autorizado | HU-COM-01 |
| `POST` | `/announcements/{id}/read-confirmations` | Confirma lectura de forma idempotente y registra fecha y hora. | Must / Docente o Admin autorizado | HU-COM-02 |
| `GET` | `/announcements/{id}/read-status` | Muestra al autor el estado Leído/Pendiente por docente. | Must / Docente o Admin autorizado | HU-COM-01 |

### Módulo 9. Cronograma interno

**Objetivo:** consultar y agendar reuniones, tutorías e hitos internos según semillero y participantes.

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `GET` | `/internal-calendar/events` | Consulta eventos internos por rango de fechas, vista semanal/mensual y semillero. Los estudiantes solo ven los de su semillero. | Must / Estudiante, Docente o Admin | HU-CRO-01 |
| `POST` | `/internal-calendar/events` | Crea reunión o hito con título, fechas, horas y participantes. | Must / Docente o Admin | HU-CRO-01 |
| `GET` | `/internal-calendar/events/{id}` | Consulta detalle y participantes según permisos. | Must / Estudiante, Docente o Admin autorizado | HU-CRO-01 |
| `PATCH` | `/internal-calendar/events/{id}` | Actualiza evento interno o participantes. | Must / Docente o Admin autorizado | HU-CRO-01 |
| `DELETE` | `/internal-calendar/events/{id}` | Cancela/elimina lógicamente un evento interno conservando trazabilidad. | Must / Docente o Admin autorizado | HU-CRO-01 |

### Módulo 10. Enlaces oficiales

**Objetivo:** centralizar enlaces externos por categoría y mantener su vigencia sin perder historial.

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `GET` | `/links` | Lista enlaces visibles; filtra por categoría y orden. | Must / Visitante | HU-LIN-01 |
| `POST` | `/links` | Crea enlace y valida esquema/formato de URL. | Must / Admin | HU-LIN-02 |
| `PATCH` | `/links/{id}` | Actualiza nombre, descripción, destino, categoría, visibilidad u orden; registra autor y fecha. | Must / Admin | HU-LIN-02 |
| `PATCH` | `/links/{id}/status` | Activa, desactiva u oculta un enlace sin borrar historial. | Must / Admin | HU-LIN-01, HU-LIN-02 |

### Módulo 11. Roles, permisos y paneles

**Objetivo:** aplicar autorización en backend y entregar al frontend el contexto necesario para adaptar el panel.

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `GET` | `/roles` | Consulta el catálogo fijo de roles: `Visitante`, `Estudiante`, `Docente` y `Admin`. | Must / Admin | HU-TR-01 |
| `GET` | `/permissions` | Consulta permisos disponibles para administrar los cuatro roles del sistema. | Must / Admin | HU-TR-01 |
| `GET` | `/roles/{roleId}/permissions` | Consulta los permisos asociados a un rol existente. | Must / Admin | HU-TR-01 |
| `PUT` | `/roles/{roleId}/permissions` | Asigna o retira permisos del rol; no permite crear roles fuera del catálogo ni autoasignarse privilegios. | Must / Admin | HU-TR-01 |
| `GET` | `/users/{userId}/roles` | Consulta roles asignados a un usuario. | Must / Admin | HU-TR-01 |
| `PUT` | `/users/{userId}/roles` | Reemplaza o asigna uno de los cuatro roles con validación de permisos y prevención de autoasignación. | Must / Admin | HU-AUTH-01, HU-TR-01 |
| `GET` | `/users/me/dashboard` | Devuelve resumen de tareas y accesos permitidos según rol y contexto. | Must / Estudiante, Docente o Admin | HU-TR-02 |

**Regla:** todos los endpoints de escritura deben comprobar el permiso requerido y el contexto (por ejemplo, pertenencia al semillero). La interfaz solo consume los permisos; no los concede.

### Módulo 12. Auditoría

**Objetivo:** conservar una bitácora inmutable de acciones críticas y permitir su consulta autorizada.

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `GET` | `/audit-logs` | Consulta bitácora filtrable por actor, acción, entidad y periodo. | Must / Admin con permiso de auditoría | HU-TR-03 |

La bitácora se genera internamente al modificar roles, solicitudes, semilleros, proyectos, eventos, comunicados y contenidos. Registra actor, acción, entidad, fecha, resultado y origen disponible. No se expone endpoint de edición o eliminación para usuarios operativos.

### Módulo 13. Archivos adjuntos (servicio transversal)

**Objetivo:** reutilizar una forma segura de cargar y consultar archivos de comunicados, evidencias, propuestas y eventos.

| Método | Endpoint | Descripción | Prioridad / acceso | Historias |
|---|---|---|---|---|
| `POST` | `/files` | Sube un archivo, valida tamaño y tipo permitido y devuelve un identificador para asociarlo a una entidad. | Must / Estudiante, Docente o Admin según contexto | HU-COM-01, HU-SEM-04, HU-SEM-06, HU-EVT-01 |
| `GET` | `/files/{id}` | Descarga o consulta metadatos del archivo tras validar permisos de acceso a la entidad asociada. | Must / Rol autorizado según contexto | HU-COM-01, HU-SEM-04, HU-SEM-06, HU-EVT-01 |

Para comunicados se permiten PDF, Word y Excel, según la historia de usuario. Los archivos privados no deben quedar disponibles mediante URL pública predecible.

## 4. Requisitos transversales y fuera del catálogo de endpoints

- **Seguridad (HU-NF-04):** HTTPS en despliegues; validación y sanitización en backend; control de sesión; mínimos privilegios; datos personales filtrados por autorización. Aplican a todos los módulos.
- **Auditoría:** registrar operaciones críticas de manera uniforme, sin permitir alteración desde la API operativa.
- **Interfaz e identidad visual (HU-NF-01, HU-NF-02, HU-NF-03):** identidad institucional, accesibilidad, diseño responsive, estados de carga/éxito/error y consistencia visual son responsabilidades de frontend/diseño; no requieren endpoints específicos. La API debe entregar errores de validación claros y consistentes.
- **Notificaciones:** avisos de revisión de reseñas y otros eventos pueden generarse internamente. La primera versión no requiere una API pública de notificaciones salvo que se decida construir una bandeja de notificaciones.
- **Errores:** usar códigos HTTP coherentes y un formato común de error, sin revelar credenciales, existencia de cuentas ni datos no autorizados.

## 5. Orden sugerido de implementación

1. **Base de Must:** autenticación/sesión, semilleros, solicitudes de registro, roles/permisos, auditoría y archivos.
2. **Operación académica Must:** membresías, integrantes/perfiles, actividades/proyectos, eventos, comunicados, cronograma, contenidos institucionales y enlaces, siempre relacionados con el `seedbedId` cuando aplique.
3. **Should:** recuperación de contraseña y propuestas de investigación.
4. **Could:** reseñas y moderación de semilleros.

El orden es una sugerencia de dependencias: permisos y auditoría deben estar disponibles desde el inicio para proteger los módulos administrativos.
