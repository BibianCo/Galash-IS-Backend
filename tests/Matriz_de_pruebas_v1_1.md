# Matriz de pruebas v1.1
Convenciones: HU=historia de usuario; CA=criterios de aceptación (formato Dado/cuando/entonces); fecha de HU=2026-09-04 salvo indicación; Estimación vacía en todas las HU (omitida).

## Sin título (HU-AUTH-*)

### HU-AUTH-01 · Iniciar Sesión · Must
Como usuario registrado
Quiero iniciar sesión con mis credenciales
Para acceder a las funciones autorizadas del sistema.
CA:
- Dado usuario activo y credenciales válidas, cuando envía el formulario, entonces el sistema crea sesión y muestra el panel correspondiente a su rol.
- Dado usuario o contraseña inválidos, cuando intenta ingresar, entonces el sistema rechaza acceso y muestra mensaje claro sin revelar cuál dato falló.
- Dado usuario inactivo, cuando intenta ingresar, entonces el sistema deniega acceso e informa que debe contactar al administrador.
- El formulario valida campos obligatorios antes de enviarse.
- El sistema debe limitar la asignación del rol 'Administrador' a máximo dos usuarios (Docente y estudiante Líder)

### HU-AUTH-02 · Cerrar sesión · Must
Como usuario autenticado
Quiero cerrar sesión
Para proteger mi cuenta al terminar.
CA:
- Cuando selecciona cerrar sesión, entonces la sesión queda invalidada y el sistema redirige al acceso público.
- Al intentar volver a una ruta protegida, entonces el sistema exige autenticación.

### HU-AUTH-03 · Recuperar acceso · Should
Como usuario registrado
Quiero recuperar mi contraseña
Para volver a ingresar si la olvido.
CA:
- El sistema permite solicitar recuperación mediante correo registrado.
- El enlace de recuperación expira y solo puede utilizarse una vez.
- El sistema no confirma si un correo existe cuando la solicitud es pública.

### HU-AUTH-04 · Gestionar solicitud de registro de estudiante · Must
Como visitante o estudiante interesado
Quiero enviar una solicitud de registro
Para pertenecer al grupo o a un semillero.
CA:
- El formulario solicita datos definidos por GALASH y valida campos obligatorios.
- El sistema asigna identificador y estado inicial Radicada.
- El sistema registra fecha, solicitante y semillero de interés, si aplica.
- El sistema confirma radicación sin exponer información privada.

### HU-AUTH-05 · Consultar estado de solicitud · Must
Como solicitante
Quiero consultar el estado de mi solicitud
Para conocer su avance.
CA:
- El solicitante consulta únicamente sus propias solicitudes.
- El sistema muestra estado actual, fecha de actualización y observaciones autorizadas.
- Estados mínimos: Radicada, En revisión, Aprobada, Rechazada, Pendiente de información

### HU-AUTH-06 · Supervisar solicitudes en tiempo real · Must
Como administrador
Quiero ver un panel de solicitudes actualizado
Para controlar el proceso interno de registro
CA:
- El panel muestra solicitudes filtrables por estado, fecha, semillero y responsable.
- Cada solicitud muestra trazabilidad de cambios, usuario responsable y fecha.
- Una actualización de estado se refleja en el panel sin requerir recarga manual o informa claramente la necesidad de actualizar.
- El cambio de estado exige permiso y registra auditoría.

## Gestión de semilleros

### HU-SEM-01 · Crear y configurar semillero · Must
Como administrador o coordinador
Quiero crear un semillero
Para organizar integrantes, actividades y proyectos.
CA:
- El sistema permite registrar nombre, descripción, línea de investigación, responsable y estado.
- El sistema impide crear semilleros sin nombre ni responsable.
- El semillero puede publicarse, ocultarse o marcarse como inactivo.

### HU-SEM-02 · Consultar espacio del semillero · Must
Como visitante o integrante
Quiero consultar el espacio de un semillero
Para conocer su propósito, integrantes, actividades y proyectos
CA:
- El espacio muestra información pública del semillero y sus responsables.
- El sistema separa proyectos activos, finalizados y radicados.
- La información privada solo aparece a usuarios autorizados.

### HU-SEM-03 · Gestionar integrantes de semillero · Must
Como coordinador
Quiero vincular y retirar integrantes
Para mantener actualizado cada semillero.
CA:
- El coordinador puede buscar usuarios existentes y asignar rol dentro del semillero.
- El sistema registra fecha de vinculación y retiro.
- Un integrante retirado conserva historial de participación, sin acceso operativo posterior.

### HU-SEM-04 · Registrar actividad de semillero · Must
Como coordinador o profesor
Quiero registrar actividades
Para evidenciar el trabajo realizado.
CA:
- La actividad incluye nombre, descripción, fecha, responsables, estado y evidencias opcionales.
- El sistema permite consultar actividades próximas, en curso y finalizadas.
- Solo usuarios autorizados pueden crear o modificar actividades.

### HU-SEM-05 · Gestionar proyecto de investigación · Must
Como profesor o coordinador
Quiero registrar y actualizar proyectos
Para controlar investigación activa, finalizada o radicada.
CA:
- El proyecto incluye título, resumen, línea, responsables, integrantes, fechas, estado y porcentaje de avance.
- El porcentaje acepta valores entre 0 y 100.
- Estados mínimos: Radicado, Activo, Finalizado.
- El sistema conserva fecha y autor de cada actualización relevante.

### HU-SEM-06 · Gestionar propuesta de investigación · Should
Como integrante autorizado
Quiero registrar una propuesta
Para someter una nueva investigación a evaluación.
CA:
- La propuesta incluye título, problema, objetivos, metodología, responsables y documentos anexos, si aplica.
- El sistema registra estado y observaciones de revisión.
- La propuesta puede convertirse en proyecto radicado sin perder su historial.

### HU-SEM-07 · Consultar avance de proyectos · Must
Como integrante o coordinador
Quiero visualizar porcentaje y estado de avance
Para conocer evolución de cada proyecto.
CA:
- El porcentaje se presenta de forma numérica y visual.
- El avance puede consultarse por proyecto, semillero y periodo.
- Solo responsables autorizados pueden modificar el porcentaje.

### HU-SEM-08 · Publicar reseña de semillero · Could · 2026-09-06
Como usuario registrado
Quiero escribir y enviar una reseña o sugerencia acerca de un semillero
Para aportar retroalimentación que garantice la mejora continua en la construcción del grupo.
CA:
- El sistema provee un formulario de texto accesible desde el perfil del semillero para enviar la reseña, registrando autor, fecha y semillero evaluado.
- El administrador recibe una notificación.
- El sistema permite al administrador moderar reseñas que incumplan las normas de convivencia del grupo.

## Eventos, talleres y convocatorias

### HU-EVT-01 · Crear evento · Must
Como administrador o coordinador
Quiero crear un evento
Para anunciar actividades académicas del grupo
CA:
- El evento incluye título, descripción, tipo, fecha, hora, lugar o enlace virtual, organizador e imagen opcional.
- El sistema valida que fecha y hora tengan formato válido.
- El evento se guarda como borrador hasta su publicación.

### HU-EVT-02 · Publicar y actualizar evento · Must
Como administrador
Quiero publicar y actualizar eventos
Para mantener información vigente.
CA:
- Solo usuarios con permiso pueden publicar, editar, ocultar o cancelar un evento.
- Los cambios publicados muestran fecha de actualización.
- Un evento cancelado conserva información histórica y muestra su estado.

### HU-EVT-03 · Explorar agenda interactiva · Must
Como visitante
Quiero explorar eventos mediante lista o calendario
Para encontrar actividades de mi interés.
CA:
- La agenda permite filtrar por tipo, fecha y estado.
- El detalle muestra información completa y enlace de inscripción, si existe.
- Eventos vencidos quedan diferenciados de eventos próximos.

## Gestión de integrantes y perfiles

### HU-INT-01 · Gestionar integrante · Must
Como administrador
Quiero crear, editar, activar o desactivar integrantes
Para mantener el directorio institucional actualizado.
CA:
- El registro distingue estudiante y profesor.
- El sistema permite asociar integrante con uno o varios semilleros.
- Los datos obligatorios se validan antes de guardar.
- Desactivar perfil no elimina historial relacionado.

### HU-INT-02 · Consultar perfil público · Must
Como visitante
Quiero consultar perfiles de integrantes, Conocer Artículos y Publicaciones
Para conocer sus roles y experiencia.
CA:
- El perfil muestra únicamente datos marcados como públicos
- El directorio permite buscar y filtrar por tipo, semillero o área.
- El perfil inactivo no aparece en el directorio público.

### HU-INT-03 · Acceder a Cvlac de profesor · Must
Como visitante
Quiero acceder al Cvlac de cada profesor
Para consultar su trayectoria académica.
CA:
- El perfil de profesor muestra enlace Cvlac cuando está registrado.
- El enlace abre una URL válida y externa.
- El administrador puede actualizar o retirar el enlace.

## Información institucional

### HU-INFO-01 · Acceder a Cvlac de profesor · Must
Como visitante
Quiero consultar historia, objetivos, misión, visión y áreas de conocimiento
Para comprender identidad y alcance de GALASH.
CA:
- Cada contenido aparece en sección identificable y navegable.
- El contenido es legible en dispositivos móviles y escritorio.
- El administrador puede actualizar contenidos sin modificar código.

### HU-INFO-02 · Gestionar contenido institucional · Must
Como administrador
Quiero editar y publicar contenido institucional
Para mantenerlo vigente.
CA:
- El sistema soporta guardar borrador y publicar.
- Se registra autor y fecha de actualización.
- El contenido publicado conserva formato seguro y consistente.

## Tablero de Comunicados

### HU-COM-01 · Publicar comunicado oficial (Emisor) · Must · 2026-09-07
Como director del grupo de investigación
Quiero publicar un anuncio adjuntando documentos dentro de un tablero centralizado
Para emitir directrices institucionales y evitar el uso de canales de mensajería informales.
CA:
- El sistema provee un formulario para título, descripción y carga de archivo adjunto (PDF, Word, Excel).
- El director puede visualizar un panel de control con el estado de lectura de cada profesor (Ej: Leído / Pendiente).

### HU-COM-02 · Confirmar lectura de documentos (Receptor) · Must · 2026-09-07
Como profesor o encargado de un semillero
Quiero visualizar los comunicados del director y confirmar su recepción
Para mantener el cumplimiento de mis responsabilidades con trazabilidad.
CA:
- El profesor visualiza el comunicado y un botón obligatorio de "Confirmar Lectura",
- Al confirmar, el sistema registra la fecha y hora de la acción, actualizando el panel del director.

## Cronograma

### HU-CRO-01 · Gestionar agendamiento interno · Must · 2026-09-07
Como integrante, profesor o director
Quiero visualizar e interactuar con un cronograma dinámico del grupo
Para planificar reuniones, tutorías o hitos internos de los semilleros.
CA:
- La interfaz presenta un calendario (mensual/semanal) donde se visualizan los eventos internos.
- El sistema permite a los profesores y directores crear reuniones, definiendo título, fecha, hora y participantes.
- Los estudiantes pueden visualizar los eventos agendados para su semillero específico, sin permisos para modificarlos.

## Enlaces, redes y repositorios

### HU-LIN-01 · Consultar enlaces oficiales · Must
Como visitante
Quiero acceder a enlaces oficiales centralizados
Para encontrar plataformas externas del grupo.
CA:
- El sistema organiza enlaces por categoría: redes sociales, Cvlac, GrupLAC, Minciencias y repositorios.
- Cada enlace muestra nombre, descripción breve y destino.
- Los enlaces inactivos u obsoletos pueden ocultarse sin borrar historial.

### HU-LIN-02 · Administrar enlaces externos · Must
Como administrador
Quiero crear, actualizar, ordenar y desactivar enlaces
Para mantener accesos vigentes.
CA:
- El sistema valida URL antes de guardar.
- El administrador puede definir visibilidad y orden.
- El sistema registra usuario y fecha del último cambio.

## Gestión de roles y permisos

### HU-TR-01 · Gestionar roles y permisos · Must
Como administrador
Quiero asignar roles y privilegios
Para limitar acciones según responsabilidad.
CA:
- El sistema permite asignar, retirar y consultar roles.
- Cada operación protegida valida permisos en backend, no solo en interfaz.
- Un usuario no puede concederse privilegios a sí mismo.
- Se registra auditoría de cambios de rol y permiso.

### HU-TR-02 · Gestionar roles y permisos · Must
Como usuario autenticado
Quiero ver un panel adaptado a mis permisos
Para acceder rápidamente a mis tareas.
CA:
- El panel no muestra acciones no autorizadas.
- Las rutas protegidas rechazan acceso directo sin permiso.
- El contenido cambia según rol y contexto del usuario.

### HU-TR-03 · Registrar auditoría · Must
Como administrador
Quiero consultar acciones críticas
Para investigar cambios y garantizar trazabilidad.
CA:
- El sistema registra actor, acción, entidad, fecha, resultado y origen disponible.
- Los registros no pueden ser modificados por usuarios operativos.
- La consulta permite filtrar por actor, acción, entidad y periodo.

## Aplicar identidad visual

### HU-NF-01 · Aplicar identidad visual · Must
Como representante de GALASH y UPTC Seccional Sogamoso
Quiero que la interfaz use identidad institucional aprobada
Para comunicar pertenencia y coherencia visual.
CA:
- Colores, logotipos, tipografías e iconografía siguen manual o lineamientos aprobados.
- Contraste y estados visuales permiten lectura accesible.
- La identidad se mantiene en páginas públicas, paneles, formularios y mensajes.

### HU-NF-02 · Usar interfaz responsive y dinámica · Must
Como usuario
Quiero interactuar con una interfaz adaptable
Para usar el sistema desde escritorio, tableta o móvil.
CA:
- Las vistas principales se adaptan a distintos tamaños de pantalla.
- Acciones muestran estados de carga, éxito y error.
- Cambios de datos se reflejan sin comportamientos ambiguos.
- El sistema conserva consistencia entre frontend y backend.

### HU-NF-03 · Mantener experiencia moderna y profesional · Must
Como usuario
Quiero una interfaz clara, estética y consistente
Para completar tareas sin fricción.
CA:
- Navegación, botones, formularios y mensajes usan patrones consistentes.
- La interfaz evita saturación visual y prioriza contenido relevante.
- Errores de validación indican campo y corrección requerida.

### HU-NF-04 · Proteger datos y sesiones · Must
Como administrador
Quiero controles de seguridad
Para proteger cuentas, datos personales y contenidos internos.
CA:
- Contraseñas nunca se almacenan en texto plano.
- Sesiones tienen expiración y protección contra acceso no autorizado.
- Entradas de usuario se validan y sanitizan en backend.
- Datos personales se muestran según permisos y necesidad de acceso.
- Comunicaciones usan HTTPS en ambientes desplegados.

# Pruebas funcionales
Para cada fila: Historia=HU-<sufijo del ID>, Matriz=TC-<sufijo del ID> (p.ej. PT-AUTH-01→HU-AUTH-01, TC-AUTH-01). Estado inicial: Pendiente en todas.
|ID|Tarea o prueba específica|Responsable sugerido|Reporte requerido|
|---|---|---|---|
|PT-AUTH-01|Ejecutar inicio de sesión con campos vacíos, credenciales inválidas, usuario inactivo y cuentas válidas de cada rol.|QA funcional y backend|Reporte de autenticación con perfil, datos usados, respuesta, acceso obtenido o rechazado, evidencia y defecto asociado.|
|PT-AUTH-02|Cerrar sesión y comprobar invalidación de sesión, redirección y bloqueo de rutas protegidas por historial o URL directa.|QA funcional y seguridad|Reporte de cierre de sesión con sesión inicial, ruta protegida, resultado de redirección, respuesta posterior y evidencia.|
|PT-AUTH-03|Solicitar recuperación con correo existente e inexistente y probar enlace válido, expirado y reutilizado.|QA de seguridad y backend|Reporte de recuperación con solicitudes, tiempos de expiración, mensajes públicos, resultado del enlace y evidencia sin exponer correos reales.|
|PT-AUTH-04|Registrar solicitudes incompletas y completas, con y sin semillero, y verificar identificador, estado inicial, fecha y confirmación.|QA funcional y backend|Reporte de radicación con campos, datos anonimizados, identificador, estado, trazabilidad, respuesta y evidencia.|
|PT-AUTH-05|Consultar solicitudes propias en todos los estados y probar acceso a la solicitud de otro solicitante.|QA funcional y seguridad|Reporte de consulta de solicitud con perfil, solicitud propia/ajena, datos visibles, autorización, resultado y evidencia.|
|PT-AUTH-06|Filtrar solicitudes, cambiar estado con y sin permiso, observar actualización y revisar trazabilidad y auditoría.|QA funcional, backend y auditoría|Reporte de supervisión con filtros, cambio realizado, actor, tiempo de actualización, auditoría, evidencia y defecto.|
|PT-SEM-01|Crear semillero válido, rechazar nombre o responsable faltante y probar publicación, ocultamiento e inactivación.|QA funcional y backend|Reporte de semillero con datos, validaciones, estados probados, resultado de visibilidad y evidencia.|
|PT-SEM-02|Consultar semillero como visitante e integrante y comparar información pública, privada y clasificación de proyectos.|QA funcional y seguridad|Reporte de espacio de semillero con perfil, contenido visible, datos ocultos, estados de proyectos, evidencia y defecto.|
|PT-SEM-03|Vincular y retirar integrante, comprobar fechas, pérdida de acceso operativo y conservación del historial.|QA funcional y autorización|Reporte de integrantes con usuario de prueba, rol, fechas, permisos antes/después, historial y evidencia.|
|PT-SEM-04|Crear y editar actividades con y sin evidencia, consultar estados por fecha y probar permisos de coordinador, profesor y usuario no autorizado.|QA funcional|Reporte de actividades con datos, periodo, estado calculado, evidencia adjunta, perfil ejecutor y resultado de autorización.|
|PT-SEM-05|Registrar proyecto, validar porcentajes fuera y dentro de rango, cambiar estados y comprobar autoría y fecha de actualizaciones.|QA funcional y auditoría|Reporte de proyecto con valores probados, estado, historial, actor, fecha, respuesta y evidencia.|
|PT-SEM-06|Registrar propuesta con y sin anexos, revisar observaciones, convertirla a proyecto radicado y verificar conservación del historial.|QA funcional e integración|Reporte de propuesta con documentos de prueba, estados, conversión, identificadores relacionados, historial y evidencia.|
|PT-SEM-07|Consultar avance por proyecto, semillero y periodo y probar modificación por responsable y no responsable.|QA funcional y autorización|Reporte de avance con filtros, porcentaje numérico/visual, perfil, permiso de edición, resultado y evidencia.|
|PT-SEM-08|Enviar reseña desde el perfil del semillero con usuario registrado y sin permiso, verificar notificación a administrador y moderar publicando, ocultando y rechazando.|QA funcional y autorización|Reporte de reseñas con autor, fecha, semillero, notificación recibida, decisión de moderación, perfil ejecutor, resultado y evidencia.|
|PT-EVT-01|Crear eventos presenciales y virtuales, validar fecha/hora inválidas, adjuntar imagen y comprobar estado borrador.|QA funcional y backend|Reporte de creación de evento con modalidad, datos, validaciones, estado de publicación, respuesta y evidencia.|
|PT-EVT-02|Publicar, editar, ocultar y cancelar evento con administrador y repetir acciones con usuario sin permiso.|QA funcional, autorización y auditoría|Reporte de ciclo de evento con acción, actor, fecha de actualización, estado histórico, autorización, evidencia y defecto.|
|PT-EVT-03|Explorar agenda en lista y calendario, aplicar filtros, abrir detalle y verificar diferencia entre eventos vencidos y próximos.|QA funcional y usabilidad|Reporte de agenda con filtros, evento consultado, enlace de inscripción, dispositivo, resultado y evidencia.|
|PT-INT-01|Crear y editar perfiles de estudiante y profesor, asociar semilleros, validar obligatorios y desactivar sin borrar historial.|QA funcional y backend|Reporte de integrante con tipo, asociaciones, validaciones, estado antes/después, historial y evidencia.|
|PT-INT-02|Consultar directorio, buscar y filtrar perfiles y verificar privacidad y exclusión de perfiles inactivos.|QA funcional y privacidad|Reporte de directorio con filtros, campos públicos observados, perfil inactivo, perfil usado y evidencia.|
|PT-INT-03|Abrir, validar, actualizar y retirar enlaces Cvlac de perfiles de profesor.|QA funcional y validación|Reporte de Cvlac con URL de prueba, validación externa, cambios, visibilidad final y evidencia.|
|PT-INFO-01|Navegar por todos los contenidos institucionales como visitante y verificar legibilidad en móvil, tableta y escritorio.|QA funcional y responsive|Reporte institucional con secciones, dispositivo, navegador, legibilidad, navegación y capturas.|
|PT-INFO-02|Guardar borrador, publicar y consultar contenido institucional; comprobar autoría, fecha, formato seguro y rechazo sin permiso.|QA funcional, seguridad y auditoría|Reporte de contenido con versión, estado, autor, fecha, sanitización, perfil y evidencia.|
|PT-CON-01|Enviar contacto válido, probar validaciones, repetir envíos rápidamente y verificar antispam y entrega al destinatario.|QA funcional, integración y seguridad|Reporte de contacto con datos anonimizados, respuesta, entrega, límite aplicado, proveedor/ambiente y evidencia.|
|PT-LIN-01|Consultar categorías de enlaces, abrir destinos, ocultar enlace obsoleto y comprobar conservación del historial.|QA funcional y navegación|Reporte de enlaces públicos con categoría, destino, estado, visibilidad, historial y evidencia.|
|PT-LIN-02|Crear, editar, ordenar, ocultar y desactivar enlaces con URL válida e inválida y perfiles con y sin permiso.|QA funcional, backend y autorización|Reporte administrativo de enlaces con URLs probadas, orden, visibilidad, actor, fecha, autorización y evidencia.|
|PT-TR-01|Asignar, consultar y retirar roles; intentar autoelevar privilegios y ejecutar operaciones protegidas desde interfaz y API.|QA de seguridad y backend|Reporte de roles y permisos con matriz esperada/observada, solicitudes directas, respuestas, auditoría y evidencia.|
|PT-TR-02|Iniciar sesión con cada rol, comparar paneles y probar acceso directo a acciones no permitidas.|QA funcional y autorización|Reporte de panel por rol con menú esperado/observado, rutas probadas, contexto, resultado y capturas.|
|PT-TR-03|Generar acciones críticas, filtrar auditoría y probar modificación o eliminación por usuario operativo.|QA de auditoría y seguridad|Reporte de auditoría con actor, acción, entidad, fecha, resultado, origen, filtros y evidencia de integridad.|
|PT-NF-01|Comparar páginas, paneles, formularios y mensajes con la identidad visual aprobada y revisar contraste y estados.|QA visual y accesibilidad|Informe de conformidad visual con referencia, elemento, resultado, contraste, evidencia antes/después y defecto.|
|PT-NF-02|Recorrer vistas y acciones en móvil, tableta y escritorio con conexión normal y lenta; comprobar estados y sincronización.|QA responsive e integración|Reporte responsive/dinámico con dispositivo, resolución, navegador, latencia, estado de interfaz, consistencia y capturas.|
|PT-NF-03|Completar flujos principales, provocar errores y revisar consistencia de navegación, botones, formularios, mensajes y jerarquía visual.|QA de usabilidad y accesibilidad|Reporte de experiencia con flujo, error provocado, mensaje, pasos, hallazgo, severidad, recomendación y evidencia.|
|PT-NF-04|Revisar almacenamiento de contraseñas, expiración de sesiones, sanitización, acceso a datos personales y uso de HTTPS.|QA de seguridad y backend|Informe de seguridad con control evaluado, técnica/dato de prueba, respuesta, riesgo, evidencia no sensible y defecto.|

## Trazabilidad de pruebas automatizadas: integrantes y roles
IDs asignados reutilizan casos existentes de esta matriz; no crean casos nuevos.
|Prueba automatizada|Archivo|ID(s) de matriz|Correspondencia|
|---|---|---|---|
|`TestUpdateUserRoleRequiresAdminAndUsesTargetUID`|`tests/auth/service_test.go`|PT-TR-01|Asigna rol a UID seleccionado y rechaza autoasignación.|
|`TestUpdateUserRoleRejectsNonAdminAndInvalidRequests`|`tests/auth/service_test.go`|PT-TR-01|Rechaza actor sin permiso, UID vacío y rol no permitido.|
|`TestSearchUsersRequiresQueryAndBoundsLimit`|`tests/auth/service_test.go`|PT-INT-02|Valida búsqueda del directorio y límite de resultados.|
|`TestAdminMembersRejectsAuthenticatedNonAdmin`|`tests/auth/http_test.go`|PT-TR-01|Rechaza búsqueda administrativa a usuario sin rol admin.|
|`TestPublicMembersRequiresQueryAndOmitsPrivateFields`|`tests/auth/http_test.go`|PT-INT-02, TDB-T09|Valida búsqueda pública y evita exposición de campos privados.|
|`TestAdminMembersRequiresSessionAndReturnsAdminFields`|`tests/auth/http_test.go`|PT-TR-01, TDB-T09|Exige sesión administrativa para exponer campos restringidos.|
|`TestUpdateRoleAcceptsHeaderAndPathTargets`|`tests/auth/http_test.go`|PT-TR-01|Actualiza rol usando UID en header o ruta.|
|`TestUpdateRoleRejectsUnauthenticatedAndConflictingTargets`|`tests/auth/http_test.go`|PT-TR-01|Rechaza solicitudes sin sesión o con UID contradictorios.|
|`TestSearchReturnsOnlyActiveMatchingUsers`|`tests/auth/user_repository_test.go`|PT-INT-02, TDB-T12|Busca por nombre/correo y excluye integrantes inactivos.|
|`TestUpdateRoleUpdatesActiveUserAndRejectsInactiveOrMissingUID`|`tests/auth/user_repository_test.go`|PT-TR-01, TDB-T05|Actualiza usuario activo y rechaza usuario inactivo o inexistente.|

# Pruebas de Diseño
|ID|Tarea o prueba específica|Relación con matriz|Responsable sugerido|Estado inicial|Reporte requerido|
|---|---|---|---|---|---|
|TD-T01|Evaluar diagramas de casos de uso frente a actores, objetivos y permisos definidos.|TD-01, TD-02, TD-04, TD-10|Analista funcional y diseñador UX|En curso|Matriz de revisión con diagrama, actor, caso de uso, inconsistencia encontrada, decisión y versión corregida.|
|TD-T02|Revisar que cada flujo principal tenga inicio, pasos, resultado, cancelación y recuperación de error.|TD-02, TD-03, TD-06, TD-09|Diseñador UX y líder técnico|En curso|Ficha de flujo con objetivo, precondiciones, recorrido, pantallas, puntos de abandono y acciones de mejora.|
|TD-T03|Verificar arquitectura de información, menú, nombres de secciones y rutas de navegación.|TD-01, TD-05|Diseñador UX|Pendiente|Reporte de navegación con tarea, ruta esperada, ruta observada, tiempo, dificultad y recomendación.|
|TD-T04|Revisar wireframes y prototipos de páginas públicas, paneles y formularios.|TD-04, TD-05, TD-06, TD-11|Diseñador UX y representante funcional|ok|Lista de hallazgos por pantalla: componente, problema, impacto, severidad, captura y corrección propuesta.|
|TD-T05|Comparar identidad visual: colores, logotipos, tipografías, iconos, espaciado y tono.|TD-11, TD-14|Diseñador UI y representante GALASH-UPTC|ok|Informe de conformidad visual con referencia del manual, elemento evaluado, resultado y evidencia antes/después.|
|TD-T06|Probar formularios con datos válidos, campos vacíos, formatos incorrectos y mensajes de error.|TD-03, TD-06, TD-09|QA y diseñador UX|Pendiente|Reporte de formulario con campo, dato usado, mensaje esperado, mensaje obtenido, resultado y captura.|
|TD-T07|Evaluar estados de carga, vacío, éxito, error, guardado, publicación, cancelación y sesión expirada.|TD-09|Diseñador UX y desarrollador frontend|Pendiente|Catálogo de estados con pantalla, condición, mensaje, acción disponible, resultado y estado de aprobación.|
|TD-T08|Comprobar panel y acciones según rol: visitante, estudiante, profesor, coordinador y administrador.|TD-04, TD-10|Analista funcional y QA|Pendiente|Matriz rol-permiso con acción, visibilidad esperada, acceso observado, evidencia y defecto asociado.|
|TD-T09|Evaluar diagramas y prototipos en móvil, tableta, escritorio y cambio de orientación.|TD-07|QA de interfaz y diseñador UI|Pendiente|Reporte responsive con dispositivo, resolución, navegador, pantalla, problema visual, severidad y captura.|
|TD-T10|Ejecutar navegación completa con teclado y revisión de foco, etiquetas, contraste y lector de pantalla.|TD-08|QA de accesibilidad|Pendiente|Informe WCAG con criterio revisado, componente, resultado, evidencia, impacto y corrección recomendada.|
|TD-T11|Realizar prueba moderada con cinco usuarios representativos sobre tareas Must.|TD-12|Investigador UX|Pendiente|Reporte de sesión anonimizado con tarea, perfil, éxito, tiempo, errores, comentario, hallazgo y prioridad.|
|TD-T12|Presentar prototipo o incremento en reunión interna dos veces por semana.|TD-13, TD-14|Líder de proyecto|Pendiente|Acta breve con fecha, asistentes, pantallas revisadas, decisiones, pendientes, responsable y fecha compromiso.|
|TD-T13|Validar prototipo o incremento con cliente cada dos o tres semanas.|TD-13|Líder de proyecto y representante funcional|Pendiente|Acta de validación con requisito, observación del cliente, aprobación o cambio solicitado, prioridad y responsable.|
|TD-T14|Comparar versión nueva contra pantallas aprobadas después de cada cambio relevante.|TD-14|QA visual|Pendiente|Reporte de regresión visual con versión base, versión evaluada, pantalla, diferencia, justificación y resultado.|
|TD-T15|Consolidar hallazgos, asignar severidad y verificar correcciones.|TD-08, TD-09, TD-10, TD-14|QA líder y líder de proyecto|Pendiente|Registro de hallazgos con ID, descripción, evidencia, severidad, responsable, fecha objetivo, estado y prueba de cierre.|

# Pruebas de Base de datos
Estado inicial: Pendiente en todas.
|ID|Tarea o prueba específica|Relación con matriz|Responsable sugerido|Reporte requerido|
|---|---|---|---|---|
|TDB-T01|Revisar modelo, diccionario, migraciones, claves y relaciones contra las historias.|TDB-01|Arquitecto de datos y backend|Matriz de trazabilidad con entidad, campo, historia, regla, resultado y evidencia.|
|TDB-T02|Probar referencias válidas, inexistentes y eliminación o actualización de registros relacionados.|TDB-02|QA de base de datos|Reporte de integridad con operación, datos anonimizados, respuesta y registros afectados.|
|TDB-T03|Intentar nulos, duplicados, estados inválidos, porcentajes fuera de rango y fechas inconsistentes.|TDB-03|QA de datos y backend|Reporte de restricciones con valor usado, regla esperada, resultado y defecto asociado.|
|TDB-T04|Crear solicitudes y recorrer todos sus estados verificando identificador, fechas, solicitante y semillero.|TDB-04|QA funcional y de datos|Reporte de solicitudes con identificadores de prueba, estados, historial, conteos y evidencia.|
|TDB-T05|Validar asociaciones de usuarios, roles, estados activo/inactivo y rechazo de autoelevación.|TDB-05|QA de seguridad y DBA|Matriz de roles de datos con operación, perfil, permiso esperado/observado, respuesta y evidencia.|
|TDB-T06|Vincular y retirar integrantes y comprobar historial, relaciones y acceso posterior.|TDB-06|QA de datos|Reporte de ciclo de integrante con fechas, relaciones antes/después, historial y resultado.|
|TDB-T07|Verificar persistencia de proyectos, propuestas, anexos, avances y conversión a proyecto.|TDB-07|QA de integración y datos|Reporte de trazabilidad con IDs relacionados, versiones, autoría, fechas y evidencia.|
|TDB-T08|Probar estados de borrador, publicación, ocultamiento y cancelación en eventos y contenidos.|TDB-08|QA de datos y contenidos|Reporte de estados con transición, visibilidad esperada/observada, fecha y evidencia.|
|TDB-T09|Comparar consultas públicas y privadas de perfiles, enlaces y datos institucionales.|TDB-09|QA de privacidad|Reporte de exposición con consulta, perfil, campos esperados/observados y evidencia sin datos reales.|
|TDB-T10|Generar acciones críticas y verificar campos, inmutabilidad y filtros de auditoría.|TDB-10|QA de auditoría y DBA|Reporte de auditoría con actor, acción, entidad, resultado, filtros y prueba de integridad.|
|TDB-T11|Ejecutar operaciones concurrentes y fallos intermedios sobre solicitudes, proyectos y eventos.|TDB-11|QA de integración y backend|Reporte de transacciones con sesiones, secuencia, rollback, conflicto y evidencia.|
|TDB-T12|Comparar resultados de filtros, conteos y relaciones con el conjunto de datos preparado.|TDB-12|QA de datos|Reporte de consultas con resultado esperado/obtenido, conteos y evidencia.|
|TDB-T13|Ejecutar migración en copia, revisar datos críticos y probar compatibilidad o reversión.|TDB-13|DBA y líder técnico|Acta de migración con versiones, duración, validaciones, diferencias y decisión.|
|TDB-T14|Crear, restaurar y comparar un respaldo en ambiente aislado.|TDB-14|DBA|Reporte de respaldo con fecha, versión, tamaño, duración, comparación y resultado.|
|TDB-T15|Revisar almacenamiento de contraseñas, privilegios, logs, datos personales y credenciales.|TDB-15|Seguridad y DBA|Informe de protección con control, técnica, hallazgo, riesgo, evidencia no sensible y corrección.|

# Pruebas de Usabilidad
Estado inicial: Pendiente en todas.
|ID|Tarea o prueba específica|Relación con matriz|Responsable sugerido|Reporte requerido|
|---|---|---|---|---|
|TU-T01|Preparar guiones, criterios de éxito, perfiles, datos anonimizados y tiempos objetivo.|TU-01 a TU-15|Investigador UX y analista funcional|Plan de sesión con historia, tarea, perfil, precondiciones, métrica, consentimiento y versión.|
|TU-T02|Ejecutar inicio, cierre y recuperación de sesión con perfiles representativos.|TU-01|Investigador UX y QA|Reporte con pasos, éxito, tiempo, errores, ayuda requerida, comentario y evidencia anonimizada.|
|TU-T03|Probar radicación y consulta de solicitud con errores y estados distintos.|TU-02|Investigador UX y QA funcional|Reporte con campos problemáticos, tasa de éxito, tiempo, abandono, hallazgos y recomendación.|
|TU-T04|Evaluar comprensión del panel y acciones por rol.|TU-03, TU-11|Investigador UX y analista funcional|Matriz de comprensión por rol con tarea, acción identificada, error, permiso entendido y evidencia.|
|TU-T05|Observar consulta y gestión de semilleros, integrantes, proyectos y propuestas.|TU-04, TU-05|Investigador UX y representante funcional|Reporte de recorrido con ruta, pasos, bloqueos, datos perdidos, comentarios y prioridad.|
|TU-T06|Probar creación, publicación, consulta y filtrado de eventos.|TU-06|Investigador UX y QA|Reporte de agenda con tarea, dispositivo, éxito, filtros, tiempo, errores y capturas.|
|TU-T07|Evaluar búsqueda de perfiles, Cvlac, contenido, contacto y enlaces.|TU-07, TU-08, TU-09|Investigador UX|Informe de descubribilidad con tarea, ruta, tiempo, dificultad, resultado y recomendación.|
|TU-T08|Provocar validaciones, estados vacíos, carga lenta, error de red y cancelación.|TU-10|Investigador UX y frontend|Catálogo de recuperación con condición, mensaje, acción, resultado y evidencia.|
|TU-T09|Realizar prueba de privacidad visible y comprensión de permisos.|TU-11|Investigador UX y seguridad|Reporte de privacidad percibida con vista, dato, interpretación, riesgo y corrección.|
|TU-T10|Ejecutar tareas Must con teclado, zoom y tecnologías de apoyo disponibles.|TU-12|QA de accesibilidad e investigador UX|Informe de accesibilidad con control, foco, etiqueta, bloqueo, criterio y evidencia.|
|TU-T11|Repetir tareas prioritarias en móvil, tableta, escritorio y orientaciones.|TU-13|QA responsive e investigador UX|Reporte comparativo con tarea, resolución, éxito, tiempo, problema y captura.|
|TU-T12|Aplicar encuesta posterior y entrevista sobre confianza, claridad y satisfacción.|TU-14|Investigador UX|Resumen anonimizado con métricas, comentarios agrupados, patrones y acciones.|
|TU-T13|Repetir tareas después de correcciones y comparar con línea base.|TU-15|QA líder e investigador UX|Reporte de regresión con versiones, métricas, hallazgos cerrados y resultado.|
|TU-T14|Consolidar hallazgos por frecuencia, impacto, perfil, dispositivo y severidad.|TU-01 a TU-15|QA líder y líder de proyecto|Registro con ID, descripción, evidencia, frecuencia, severidad, responsable, fecha y estado.|
