-- Synthetic data for local development. Every name, document, email and link is fictional.
-- All ids start with 5eed so synthetic_reset.sql can delete only this data.
-- Idempotent: running it again inserts nothing new. Dates are relative to CURRENT_DATE.
BEGIN;

-- Users: 3 teachers (one leads each seedbed), 9 active students, 1 inactive student, 1 visitor.
-- firebase_uid values are not real Firebase accounts: these users cannot log in.
INSERT INTO usuarios (id, firebase_uid, nombre, apellido, tipo_dni, dni, email, rol, estado) VALUES
  ('5eed0001-0000-4000-8000-000000000001', 'synthetic-uid-01', 'Daniela', 'Ríos Ejemplo',    'CC', '9000000001', 'daniela.rios@example.test',    'profesor',   'activo'),
  ('5eed0001-0000-4000-8000-000000000002', 'synthetic-uid-02', 'Mateo',   'Suárez Ejemplo',  'CC', '9000000002', 'mateo.suarez@example.test',    'profesor',   'activo'),
  ('5eed0001-0000-4000-8000-000000000003', 'synthetic-uid-03', 'Laura',   'Gómez Ejemplo',   'CC', '9000000003', 'laura.gomez@example.test',     'profesor',   'activo'),
  ('5eed0001-0000-4000-8000-000000000004', 'synthetic-uid-04', 'Camila',  'Pardo Ejemplo',   'CC', '9000000004', 'camila.pardo@example.test',    'estudiante', 'activo'),
  ('5eed0001-0000-4000-8000-000000000005', 'synthetic-uid-05', 'Andrés',  'Vargas Ejemplo',  'CC', '9000000005', 'andres.vargas@example.test',   'estudiante', 'activo'),
  ('5eed0001-0000-4000-8000-000000000006', 'synthetic-uid-06', 'Valeria', 'Moreno Ejemplo',  'CC', '9000000006', 'valeria.moreno@example.test',  'estudiante', 'activo'),
  ('5eed0001-0000-4000-8000-000000000007', 'synthetic-uid-07', 'Julián',  'Castro Ejemplo',  'CC', '9000000007', 'julian.castro@example.test',   'estudiante', 'activo'),
  ('5eed0001-0000-4000-8000-000000000008', 'synthetic-uid-08', 'Sofía',   'Rojas Ejemplo',   'CC', '9000000008', 'sofia.rojas@example.test',     'estudiante', 'activo'),
  ('5eed0001-0000-4000-8000-000000000009', 'synthetic-uid-09', 'Tomás',   'Niño Ejemplo',    'CC', '9000000009', 'tomas.nino@example.test',      'estudiante', 'activo'),
  ('5eed0001-0000-4000-8000-000000000010', 'synthetic-uid-10', 'Isabela', 'Cruz Ejemplo',    'CC', '9000000010', 'isabela.cruz@example.test',    'estudiante', 'activo'),
  ('5eed0001-0000-4000-8000-000000000011', 'synthetic-uid-11', 'Samuel',  'Peña Ejemplo',    'CC', '9000000011', 'samuel.pena@example.test',     'estudiante', 'activo'),
  ('5eed0001-0000-4000-8000-000000000012', 'synthetic-uid-12', 'Mariana', 'León Ejemplo',    'CC', '9000000012', 'mariana.leon@example.test',    'estudiante', 'activo'),
  ('5eed0001-0000-4000-8000-000000000013', 'synthetic-uid-13', 'Felipe',  'Ortiz Ejemplo',   'CC', '9000000013', 'felipe.ortiz@example.test',    'estudiante', 'inactivo'),
  ('5eed0001-0000-4000-8000-000000000014', 'synthetic-uid-14', 'Paula',   'Méndez Ejemplo',  'CC', '9000000014', 'paula.mendez@example.test',    'visitante',  'activo')
ON CONFLICT DO NOTHING;

INSERT INTO perfil_investigador (id, usuario_id, area_investigacion, formacion, orcid, cvlac, visible_publicamente)
SELECT ('5eed0002-0000-4000-8000-0000000000' || lpad(n::text, 2, '0'))::uuid,
       ('5eed0001-0000-4000-8000-0000000000' || lpad(n::text, 2, '0'))::uuid,
       area, formacion, 'https://example.test/orcid/' || n, 'https://example.test/cvlac/' || n, n <> 12
FROM (VALUES
  (1, 'Realidad aumentada', 'Maestría'), (2, 'Inteligencia artificial', 'Doctorado'), (3, 'Ciberseguridad', 'Maestría'),
  (4, 'Realidad aumentada', 'Pregrado'), (5, 'Realidad aumentada', 'Pregrado'), (6, 'Inteligencia artificial', 'Pregrado'),
  (7, 'Inteligencia artificial', 'Pregrado'), (8, 'Ciberseguridad', 'Pregrado'), (9, 'Ciberseguridad', 'Pregrado'),
  (10, 'Inteligencia artificial', 'Pregrado'), (11, 'Ciberseguridad', 'Pregrado'), (12, 'Realidad aumentada', 'Pregrado'),
  (13, 'Ciberseguridad', 'Pregrado')
) AS p(n, area, formacion)
ON CONFLICT DO NOTHING;

-- Seedbeds: 3 published (names match the frontend knowledge graphs) and 1 draft that stays hidden.
INSERT INTO semilleros (id, nombre, descripcion, linea_investigacion, objetivo, lider_id, email_contacto, estado, visibilidad, temas) VALUES
  ('5eed0003-0000-4000-8000-000000000001', 'Realidad aumentada',
   'Exploramos experiencias de realidad aumentada que conectan el mundo físico con contenidos digitales interactivos.',
   'Experiencias inmersivas', 'Diseñar prototipos inmersivos que apoyen el aprendizaje y la divulgación.',
   '5eed0001-0000-4000-8000-000000000001', 'ra@example.test', 'Publicado', 'Publicado',
   '[{"title":"Experiencias","phrase":"¿Cómo se siente aprender dentro de un objeto?"},{"title":"Entorno","phrase":"El campus como interfaz."},{"title":"Personas","phrase":"Diseño centrado en quien mira."},{"title":"Interacción","phrase":"Gestos que cambian la escena."}]'),
  ('5eed0003-0000-4000-8000-000000000002', 'Inteligencia artificial',
   'Estudiamos modelos de inteligencia artificial y análisis de datos para construir soluciones para la comunidad.',
   'Aprendizaje automático', 'Aplicar modelos de datos de forma responsable a problemas de la universidad.',
   '5eed0001-0000-4000-8000-000000000002', 'ia@example.test', 'Publicado', 'Publicado',
   '[{"title":"Modelos","phrase":"¿Qué aprende una máquina de nuestros datos?"},{"title":"Datos","phrase":"Calidad antes que cantidad."},{"title":"Personas","phrase":"IA que explica sus decisiones."},{"title":"Aprendizaje","phrase":"Modelos al servicio del aula."}]'),
  ('5eed0003-0000-4000-8000-000000000003', 'Ciberseguridad',
   'Investigamos cómo proteger aplicaciones, datos y personas frente a los riesgos del entorno digital.',
   'Seguridad digital', 'Evaluar riesgos y promover prácticas seguras en el desarrollo de software.',
   '5eed0001-0000-4000-8000-000000000003', 'ciber@example.test', 'Publicado', 'Publicado',
   '[{"title":"Aplicaciones","phrase":"¿Dónde falla el código que usamos a diario?"},{"title":"Datos","phrase":"Proteger lo que nos identifica."},{"title":"Personas","phrase":"El eslabón que más se ataca."},{"title":"Riesgos","phrase":"Medir antes de reaccionar."}]'),
  ('5eed0003-0000-4000-8000-000000000004', 'Robótica educativa (borrador)',
   'Semillero en preparación; no debe aparecer en la vista pública.',
   'Robótica', 'Validar que los borradores permanecen ocultos.',
   '5eed0001-0000-4000-8000-000000000001', 'robotica@example.test', 'Borrador', 'Borrador', '[]')
ON CONFLICT DO NOTHING;

INSERT INTO semillero_miembros (semillero_id, usuario_id, ingreso_en, retirado_en) VALUES
  ('5eed0003-0000-4000-8000-000000000001', '5eed0001-0000-4000-8000-000000000001', now() - interval '400 days', NULL),
  ('5eed0003-0000-4000-8000-000000000001', '5eed0001-0000-4000-8000-000000000004', now() - interval '200 days', NULL),
  ('5eed0003-0000-4000-8000-000000000001', '5eed0001-0000-4000-8000-000000000005', now() - interval '150 days', NULL),
  ('5eed0003-0000-4000-8000-000000000001', '5eed0001-0000-4000-8000-000000000012', now() - interval '90 days',  NULL),
  ('5eed0003-0000-4000-8000-000000000002', '5eed0001-0000-4000-8000-000000000002', now() - interval '380 days', NULL),
  ('5eed0003-0000-4000-8000-000000000002', '5eed0001-0000-4000-8000-000000000006', now() - interval '210 days', NULL),
  ('5eed0003-0000-4000-8000-000000000002', '5eed0001-0000-4000-8000-000000000007', now() - interval '160 days', NULL),
  ('5eed0003-0000-4000-8000-000000000002', '5eed0001-0000-4000-8000-000000000010', now() - interval '60 days',  NULL),
  ('5eed0003-0000-4000-8000-000000000003', '5eed0001-0000-4000-8000-000000000003', now() - interval '420 days', NULL),
  ('5eed0003-0000-4000-8000-000000000003', '5eed0001-0000-4000-8000-000000000008', now() - interval '230 days', NULL),
  ('5eed0003-0000-4000-8000-000000000003', '5eed0001-0000-4000-8000-000000000009', now() - interval '170 days', NULL),
  ('5eed0003-0000-4000-8000-000000000003', '5eed0001-0000-4000-8000-000000000011', now() - interval '80 days',  NULL),
  -- Withdrawn member: only visible in the restricted history.
  ('5eed0003-0000-4000-8000-000000000003', '5eed0001-0000-4000-8000-000000000013', now() - interval '300 days', now() - interval '30 days')
ON CONFLICT DO NOTHING;

-- Projects: every seedbed has Activo, Finalizado and Radicado projects.
INSERT INTO proyectos (id, titulo, resumen, semillero_id, area_investigacion, estado, avance, fecha_inicio, fecha_fin, responsables, integrantes, creado_por)
SELECT ('5eed0004-0000-4000-8000-0000000000' || lpad(n::text, 2, '0'))::uuid, titulo, resumen,
       ('5eed0003-0000-4000-8000-00000000000' || s)::uuid, area, estado, avance,
       CURRENT_DATE + inicio, CASE WHEN estado = 'Finalizado' THEN CURRENT_DATE + inicio + 180 END,
       ARRAY['5eed0001-0000-4000-8000-00000000000' || s], '{}',
       ('5eed0001-0000-4000-8000-00000000000' || s)::uuid
FROM (VALUES
  (1,  1, 'Recorrido aumentado del campus', 'Prototipo para descubrir espacios universitarios con información virtual.', 'Experiencias inmersivas', 'Activo', 45, -120),
  (2,  1, 'Laboratorio de anatomía 3D', 'Modelos interactivos para acompañar la formación en ciencias de la salud.', 'Experiencias inmersivas', 'Activo', 70, -200),
  (3,  1, 'Museo interactivo', 'Experiencia aumentada para explorar piezas y relatos culturales.', 'Experiencias inmersivas', 'Finalizado', 100, -500),
  (4,  1, 'Guía inmersiva de patrimonio', 'Recorrido que conecta lugares con su historia.', 'Experiencias inmersivas', 'Radicado', 0, 30),
  (5,  2, 'Asistente académico inteligente', 'Orientación sobre recursos y actividades universitarias.', 'Aprendizaje automático', 'Activo', 55, -150),
  (6,  2, 'Clasificación de documentos', 'Modelos para organizar información por su contenido.', 'Aprendizaje automático', 'Activo', 30, -90),
  (7,  2, 'Análisis de datos ambientales', 'Patrones en registros abiertos del entorno.', 'Aprendizaje automático', 'Finalizado', 100, -450),
  (8,  2, 'Visión artificial para reciclaje', 'Reconocimiento de materiales a partir de imágenes.', 'Aprendizaje automático', 'Radicado', 0, 45),
  (9,  3, 'Análisis de seguridad en aplicaciones', 'Exploración de vulnerabilidades y prácticas de protección de datos.', 'Seguridad digital', 'Activo', 60, -180),
  (10, 3, 'Detección de mensajes sospechosos', 'Señales para reconocer intentos de suplantación.', 'Seguridad digital', 'Activo', 25, -60),
  (11, 3, 'Guía de protección de datos', 'Material de consulta sobre el cuidado de la información personal.', 'Seguridad digital', 'Finalizado', 100, -400),
  (12, 3, 'Taller de ciudadanía digital', 'Formación en prácticas responsables en internet.', 'Seguridad digital', 'Radicado', 0, 20)
) AS p(n, s, titulo, resumen, area, estado, avance, inicio)
ON CONFLICT DO NOTHING;

-- Activities: past, current and upcoming per seedbed; status follows the date like the backend computes it.
INSERT INTO actividades (id, titulo, descripcion, estado, fecha, semillero_id, responsables)
SELECT ('5eed0005-0000-4000-8000-0000000000' || lpad(n::text, 2, '0'))::uuid, titulo, descripcion,
       CASE WHEN dia < 0 THEN 'finalizada' WHEN dia = 0 THEN 'en curso' ELSE 'próxima' END,
       CURRENT_DATE + dia, ('5eed0003-0000-4000-8000-00000000000' || s)::uuid,
       ARRAY['5eed0001-0000-4000-8000-00000000000' || s]
FROM (VALUES
  (1,  1, 'Laboratorio de experiencias inmersivas', 'Prototipos que combinan objetos físicos y contenido virtual.', 21),
  (2,  1, 'Realidad aumentada en el aula', 'Experiencias interactivas para apoyar el aprendizaje.', 0),
  (3,  1, 'Encuentro de diseño tridimensional', 'Técnicas de modelado para experiencias aumentadas.', -40),
  (4,  1, 'Jornada de tecnologías inmersivas', 'Demostración de propuestas para la comunidad.', -150),
  (5,  2, 'Encuentro de aprendizaje automático', 'Experimentos y resultados de modelos de aprendizaje.', 28),
  (6,  2, 'IA aplicada a la educación', 'Herramientas para acompañar procesos de aprendizaje.', 7),
  (7,  2, 'Laboratorio de datos abiertos', 'Exploración y preparación de conjuntos de datos.', -55),
  (8,  2, 'Jornada de IA responsable', 'Uso responsable de modelos y sus resultados.', -130),
  (9,  3, 'Encuentro de semilleros', 'Ideas, iniciativas y equipos de investigación.', 14),
  (10, 3, 'Participación docentes', 'Intercambio de experiencias y avances de investigación.', 0),
  (11, 3, 'Ponencia universitaria', 'Proyectos y aprendizajes del semillero.', -35),
  (12, 3, 'Congreso de seguridad en aplicaciones', 'Buenas prácticas de desarrollo seguro.', -190)
) AS a(n, s, titulo, descripcion, dia)
ON CONFLICT DO NOTHING;

-- Registration requests in every state. Tracking key of request N: synthetic-token-N
-- (token_hash = base64url(sha256(token)) without padding, like the backend).
INSERT INTO solicitudes_registro (id, semillero_id, nombre, cc, cod, telefono, fecha_nacimiento, lugar_nacimiento,
  direccion_residencia, link_cvlac, link_orcid, maximo_nivel_estudio, titulo_maximo_nivel_estudio, titulo_pregrado,
  vinculacion, categoria_actual_investigador, convocatoria_minciencias, estado, observacion, token_hash, fecha_radicacion, actualizado_en)
SELECT ('5eed0006-0000-4000-8000-0000000000' || lpad(n::text, 2, '0'))::uuid,
       CASE WHEN s = 0 THEN NULL ELSE ('5eed0003-0000-4000-8000-00000000000' || s)::uuid END,
       nombre, '91000000' || lpad(n::text, 2, '0'), '2026000' || lpad(n::text, 2, '0'), '30000000' || lpad(n::text, 2, '0'),
       DATE '2003-01-15' + n * 97, 'Tunja', 'Calle Ficticia ' || n, 'https://example.test/cvlac/s' || n,
       'https://example.test/orcid/s' || n, 'Pregrado', 'Ingeniería de Sistemas', 'Ingeniería de Sistemas',
       'Estudiante', 'Sin categoría', 'Sin participación', estado, observacion,
       rtrim(translate(encode(digest('synthetic-token-' || n, 'sha256'), 'base64'), '+/', '-_'), '='),
       now() - (dias || ' days')::interval, now() - (dias || ' days')::interval + interval '1 day'
FROM (VALUES
  (1, 3, 'Natalia Ejemplo Uno',    'Radicada',                 NULL,                                        2),
  (2, 2, 'Esteban Ejemplo Dos',    'Radicada',                 NULL,                                        4),
  (3, 1, 'Lucía Ejemplo Tres',     'En revisión',              'Revisión de documentos en curso.',          9),
  (4, 0, 'Martín Ejemplo Cuatro',  'Pendiente de información', 'Falta el enlace CvLAC actualizado.',         12),
  (5, 3, 'Gabriela Ejemplo Cinco', 'Aprobada',                 'Cumple los requisitos del semillero.',      25),
  (6, 2, 'Diego Ejemplo Seis',     'Rechazada',                'La vinculación no corresponde a la convocatoria.', 30)
) AS r(n, s, nombre, estado, observacion, dias)
ON CONFLICT DO NOTHING;

-- History following the contract transitions (Radicada -> En revisión | Pendiente de información -> ...).
INSERT INTO seguimiento_solicitud (id, solicitud_id, estado, comentario, responsable_id, fecha)
SELECT ('5eed0007-0000-4000-8000-0000000000' || lpad(h::text, 2, '0'))::uuid,
       ('5eed0006-0000-4000-8000-0000000000' || lpad(n::text, 2, '0'))::uuid, estado, comentario,
       CASE WHEN estado = 'Radicada' THEN NULL ELSE '5eed0001-0000-4000-8000-000000000003'::uuid END,
       now() - (dias || ' days')::interval
FROM (VALUES
  (1,  1, 'Radicada', 'Solicitud radicada', 2),
  (2,  2, 'Radicada', 'Solicitud radicada', 4),
  (3,  3, 'Radicada', 'Solicitud radicada', 9),
  (4,  3, 'En revisión', 'Revisión de documentos en curso.', 8),
  (5,  4, 'Radicada', 'Solicitud radicada', 12),
  (6,  4, 'Pendiente de información', 'Falta el enlace CvLAC actualizado.', 11),
  (7,  5, 'Radicada', 'Solicitud radicada', 25),
  (8,  5, 'En revisión', 'Documentos completos.', 24),
  (9,  5, 'Aprobada', 'Cumple los requisitos del semillero.', 20),
  (10, 6, 'Radicada', 'Solicitud radicada', 30),
  (11, 6, 'En revisión', 'Documentos completos.', 29),
  (12, 6, 'Rechazada', 'La vinculación no corresponde a la convocatoria.', 26)
) AS t(h, n, estado, comentario, dias)
ON CONFLICT DO NOTHING;

-- Agenda events (M6): upcoming and past published, plus one draft.
INSERT INTO eventos (id, titulo, descripcion, tipo, inicia_en, termina_en, organizador_id, lugar, enlace, estado)
SELECT ('5eed0008-0000-4000-8000-0000000000' || lpad(n::text, 2, '0'))::uuid, titulo, descripcion, tipo,
       CURRENT_DATE + dia + time '14:00', CURRENT_DATE + dia + time '17:00',
       '5eed0001-0000-4000-8000-000000000002'::uuid, lugar, '', estado
FROM (VALUES
  (1, 'Encuentro institucional de semilleros', 'Muestra de proyectos de todos los semilleros.', 'evento', 18, 'Auditorio principal', 'publicado'),
  (2, 'Taller de escritura científica', 'Estructura de artículos y uso de referencias.', 'taller', 9, 'Sala 204', 'publicado'),
  (3, 'Convocatoria interna de proyectos', 'Socialización de requisitos y fechas.', 'convocatoria', -20, 'Sala virtual', 'publicado'),
  (4, 'Hackatón de datos abiertos', 'Evento en preparación.', 'evento', 45, 'Laboratorio 3', 'borrador')
) AS e(n, titulo, descripcion, tipo, dia, lugar, estado)
ON CONFLICT DO NOTHING;

COMMIT;
