CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE TABLE usuarios (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    firebase_uid TEXT NOT NULL UNIQUE,
    nombre TEXT NOT NULL,
    apellido TEXT,
    tipo_dni TEXT,
    dni TEXT UNIQUE,
    email TEXT NOT NULL,
    password_hash TEXT,
    rol TEXT NOT NULL DEFAULT 'visitante',
    estado TEXT NOT NULL DEFAULT 'activo',
    semillero_id UUID,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sesiones (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    usuario_id UUID NOT NULL REFERENCES usuarios(id),
    token_hash TEXT NOT NULL UNIQUE,
    expira_en TIMESTAMPTZ NOT NULL,
    revocada_en TIMESTAMPTZ
);

CREATE TABLE recuperaciones_password (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    usuario_id UUID NOT NULL REFERENCES usuarios(id),
    token_hash TEXT NOT NULL UNIQUE,
    expira_en TIMESTAMPTZ NOT NULL,
    usado_en TIMESTAMPTZ
);

CREATE TABLE lineas_investigacion (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nombre TEXT NOT NULL UNIQUE,
    descripcion TEXT
);

CREATE TABLE semilleros (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nombre TEXT NOT NULL UNIQUE,
    descripcion TEXT,
    linea_id UUID REFERENCES lineas_investigacion(id),
    lider_id UUID REFERENCES usuarios(id),
    palabras_clave TEXT[]
);

ALTER TABLE semilleros ADD COLUMN IF NOT EXISTS linea_investigacion TEXT;
ALTER TABLE semilleros ADD COLUMN IF NOT EXISTS objetivo TEXT;
ALTER TABLE semilleros ADD COLUMN IF NOT EXISTS email_contacto TEXT;
ALTER TABLE semilleros ADD COLUMN IF NOT EXISTS estado TEXT NOT NULL DEFAULT 'Borrador';
ALTER TABLE semilleros ADD COLUMN IF NOT EXISTS visibilidad TEXT NOT NULL DEFAULT 'Borrador';
ALTER TABLE semilleros ADD COLUMN IF NOT EXISTS creado_en TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE semilleros ADD COLUMN IF NOT EXISTS actualizado_en TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE TABLE IF NOT EXISTS semillero_miembros (
    semillero_id UUID NOT NULL REFERENCES semilleros(id),
    usuario_id UUID NOT NULL REFERENCES usuarios(id),
    ingreso_en TIMESTAMPTZ NOT NULL DEFAULT now(),
    retirado_en TIMESTAMPTZ,
    PRIMARY KEY (semillero_id, usuario_id)
);
CREATE INDEX IF NOT EXISTS semillero_miembros_usuario_idx ON semillero_miembros(usuario_id);

ALTER TABLE usuarios
    ADD CONSTRAINT usuarios_semillero_fk
    FOREIGN KEY (semillero_id) REFERENCES semilleros(id);

CREATE TABLE solicitudes_registro (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    usuario_id UUID REFERENCES usuarios(id),
    semillero_id UUID REFERENCES semilleros(id),
    nombre TEXT NOT NULL,
    cc TEXT NOT NULL,
    cod TEXT NOT NULL,
    telefono TEXT NOT NULL,
    fecha_nacimiento DATE NOT NULL,
    lugar_nacimiento TEXT NOT NULL,
    direccion_residencia TEXT NOT NULL,
    link_cvlac TEXT NOT NULL,
    link_orcid TEXT NOT NULL,
    maximo_nivel_estudio TEXT NOT NULL,
    titulo_maximo_nivel_estudio TEXT NOT NULL,
    titulo_pregrado TEXT NOT NULL,
    vinculacion TEXT NOT NULL,
    categoria_actual_investigador TEXT NOT NULL,
    convocatoria_minciencias TEXT NOT NULL,
    estado TEXT NOT NULL DEFAULT 'Radicada',
    observacion TEXT,
    token_hash TEXT NOT NULL UNIQUE,
    responsable_id UUID REFERENCES usuarios(id),
    fecha_radicacion TIMESTAMPTZ NOT NULL DEFAULT now(),
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE solicitudes_registro ADD COLUMN IF NOT EXISTS nombre TEXT;
ALTER TABLE solicitudes_registro ADD COLUMN IF NOT EXISTS cc TEXT;
ALTER TABLE solicitudes_registro ADD COLUMN IF NOT EXISTS cod TEXT;
ALTER TABLE solicitudes_registro ADD COLUMN IF NOT EXISTS telefono TEXT;
ALTER TABLE solicitudes_registro ADD COLUMN IF NOT EXISTS fecha_nacimiento DATE;
ALTER TABLE solicitudes_registro ADD COLUMN IF NOT EXISTS lugar_nacimiento TEXT;
ALTER TABLE solicitudes_registro ADD COLUMN IF NOT EXISTS direccion_residencia TEXT;
ALTER TABLE solicitudes_registro ADD COLUMN IF NOT EXISTS link_cvlac TEXT;
ALTER TABLE solicitudes_registro ADD COLUMN IF NOT EXISTS link_orcid TEXT;
ALTER TABLE solicitudes_registro ADD COLUMN IF NOT EXISTS maximo_nivel_estudio TEXT;
ALTER TABLE solicitudes_registro ADD COLUMN IF NOT EXISTS titulo_maximo_nivel_estudio TEXT;
ALTER TABLE solicitudes_registro ADD COLUMN IF NOT EXISTS titulo_pregrado TEXT;
ALTER TABLE solicitudes_registro ADD COLUMN IF NOT EXISTS vinculacion TEXT;
ALTER TABLE solicitudes_registro ADD COLUMN IF NOT EXISTS categoria_actual_investigador TEXT;
ALTER TABLE solicitudes_registro ADD COLUMN IF NOT EXISTS convocatoria_minciencias TEXT;
ALTER TABLE solicitudes_registro ADD COLUMN IF NOT EXISTS observacion TEXT;
ALTER TABLE solicitudes_registro ADD COLUMN IF NOT EXISTS token_hash TEXT;
ALTER TABLE solicitudes_registro ADD COLUMN IF NOT EXISTS responsable_id UUID REFERENCES usuarios(id);
ALTER TABLE solicitudes_registro ADD COLUMN IF NOT EXISTS fecha_radicacion TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE solicitudes_registro ADD COLUMN IF NOT EXISTS actualizado_en TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE solicitudes_registro ALTER COLUMN usuario_id DROP NOT NULL;
ALTER TABLE solicitudes_registro ALTER COLUMN semillero_id DROP NOT NULL;
ALTER TABLE solicitudes_registro ALTER COLUMN estado SET DEFAULT 'Radicada';
UPDATE solicitudes_registro SET estado='Radicada' WHERE estado='pendiente';
UPDATE solicitudes_registro SET estado='En revisión' WHERE estado='Informada';
UPDATE solicitudes_registro SET estado='Pendiente de información' WHERE estado='Detenida';

CREATE TABLE perfil_investigador (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    usuario_id UUID NOT NULL UNIQUE REFERENCES usuarios(id),
    formacion TEXT,
    orcid TEXT,
    google_scholar TEXT,
    scopus TEXT,
    proyectos_dirigidos INTEGER NOT NULL DEFAULT 0,
    articulos_publicados INTEGER NOT NULL DEFAULT 0
);
ALTER TABLE perfil_investigador ADD COLUMN IF NOT EXISTS area_investigacion TEXT;
ALTER TABLE perfil_investigador ADD COLUMN IF NOT EXISTS cvlac TEXT;
ALTER TABLE perfil_investigador ADD COLUMN IF NOT EXISTS visible_publicamente BOOLEAN NOT NULL DEFAULT true;

CREATE TABLE IF NOT EXISTS publicaciones (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    usuario_id UUID NOT NULL REFERENCES usuarios(id),
    titulo TEXT NOT NULL,
    descripcion TEXT,
    url TEXT,
    publicado_en TIMESTAMPTZ NOT NULL DEFAULT now(),
    visible BOOLEAN NOT NULL DEFAULT true
);
CREATE INDEX IF NOT EXISTS publicaciones_usuario_idx ON publicaciones(usuario_id, publicado_en DESC);

CREATE TABLE noticias (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    titulo TEXT NOT NULL,
    contenido TEXT NOT NULL,
    fecha_publicacion TIMESTAMPTZ NOT NULL DEFAULT now(),
    imagen_url TEXT,
    autor_id UUID NOT NULL REFERENCES usuarios(id)
);

CREATE TABLE seguimiento_solicitud (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    solicitud_id UUID NOT NULL REFERENCES solicitudes_registro(id),
    estado TEXT NOT NULL,
    comentario TEXT,
    responsable_id UUID NOT NULL REFERENCES usuarios(id),
    fecha TIMESTAMPTZ NOT NULL DEFAULT now()
);
ALTER TABLE seguimiento_solicitud ALTER COLUMN responsable_id DROP NOT NULL;

CREATE TABLE IF NOT EXISTS notificaciones (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    usuario_id UUID NOT NULL REFERENCES usuarios(id),
    tipo TEXT NOT NULL,
    mensaje TEXT NOT NULL,
    leida_en TIMESTAMPTZ,
    creada_en TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE proyectos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    titulo TEXT NOT NULL,
    resumen TEXT,
    impacto TEXT,
    fecha_inicio DATE,
    fecha_fin DATE,
    semillero_id UUID NOT NULL REFERENCES semilleros(id),
    area_investigacion TEXT,
    estado TEXT NOT NULL DEFAULT 'Radicado', avance INTEGER NOT NULL DEFAULT 0 CHECK (avance BETWEEN 0 AND 100),
    responsables TEXT[] NOT NULL DEFAULT '{}', integrantes TEXT[] NOT NULL DEFAULT '{}', creado_por UUID REFERENCES usuarios(id), actualizado_en TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE actividades (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    titulo TEXT NOT NULL,
    descripcion TEXT,
    estado TEXT NOT NULL DEFAULT 'pendiente',
    fecha DATE,
    semillero_id UUID NOT NULL REFERENCES semilleros(id), responsables TEXT[] NOT NULL DEFAULT '{}', evidencias TEXT[] NOT NULL DEFAULT '{}', creado_en TIMESTAMPTZ NOT NULL DEFAULT now(), actualizado_en TIMESTAMPTZ NOT NULL DEFAULT now()
);
ALTER TABLE proyectos ADD COLUMN IF NOT EXISTS estado TEXT NOT NULL DEFAULT 'Radicado';
ALTER TABLE proyectos ADD COLUMN IF NOT EXISTS avance INTEGER NOT NULL DEFAULT 0;
ALTER TABLE proyectos ADD COLUMN IF NOT EXISTS responsables TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE proyectos ADD COLUMN IF NOT EXISTS integrantes TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE proyectos ADD COLUMN IF NOT EXISTS creado_por UUID REFERENCES usuarios(id);
ALTER TABLE proyectos ADD COLUMN IF NOT EXISTS actualizado_en TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE actividades ADD COLUMN IF NOT EXISTS responsables TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE actividades ADD COLUMN IF NOT EXISTS evidencias TEXT[] NOT NULL DEFAULT '{}';
ALTER TABLE actividades ADD COLUMN IF NOT EXISTS creado_en TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE actividades ADD COLUMN IF NOT EXISTS actualizado_en TIMESTAMPTZ NOT NULL DEFAULT now();
CREATE TABLE IF NOT EXISTS proyecto_historial (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), proyecto_id UUID NOT NULL REFERENCES proyectos(id) ON DELETE CASCADE, autor_id UUID REFERENCES usuarios(id), campo TEXT NOT NULL, valor TEXT NOT NULL, creado_en TIMESTAMPTZ NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS propuestas_investigacion (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), semillero_id UUID REFERENCES semilleros(id), titulo TEXT NOT NULL, problema TEXT NOT NULL, objetivos TEXT NOT NULL, metodologia TEXT NOT NULL, estado TEXT NOT NULL DEFAULT 'Radicada', responsables TEXT[] NOT NULL DEFAULT '{}', anexos TEXT[] NOT NULL DEFAULT '{}', autor_id UUID NOT NULL REFERENCES usuarios(id), observaciones TEXT, creado_en TIMESTAMPTZ NOT NULL DEFAULT now(), actualizado_en TIMESTAMPTZ NOT NULL DEFAULT now());
ALTER TABLE propuestas_investigacion ADD COLUMN IF NOT EXISTS proyecto_id UUID REFERENCES proyectos(id);
CREATE TABLE IF NOT EXISTS propuesta_historial (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), propuesta_id UUID NOT NULL REFERENCES propuestas_investigacion(id) ON DELETE CASCADE, autor_id UUID REFERENCES usuarios(id), estado TEXT, observacion TEXT, creado_en TIMESTAMPTZ NOT NULL DEFAULT now());
CREATE TABLE IF NOT EXISTS semillero_resenas (id UUID PRIMARY KEY DEFAULT gen_random_uuid(), semillero_id UUID NOT NULL REFERENCES semilleros(id), autor_id UUID NOT NULL REFERENCES usuarios(id), texto TEXT NOT NULL, estado TEXT NOT NULL DEFAULT 'pendiente', nota_moderacion TEXT, creado_en TIMESTAMPTZ NOT NULL DEFAULT now(), moderado_en TIMESTAMPTZ);

CREATE TABLE eventos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    titulo TEXT NOT NULL,
    descripcion TEXT,
    fecha TIMESTAMPTZ NOT NULL,
    semillero_id UUID NOT NULL REFERENCES semilleros(id)
);

-- M6: agenda pública de eventos, talleres y convocatorias.
ALTER TABLE eventos ADD COLUMN IF NOT EXISTS tipo TEXT NOT NULL DEFAULT 'evento';
ALTER TABLE eventos ADD COLUMN IF NOT EXISTS inicia_en TIMESTAMPTZ;
ALTER TABLE eventos ADD COLUMN IF NOT EXISTS termina_en TIMESTAMPTZ;
ALTER TABLE eventos ADD COLUMN IF NOT EXISTS organizador_id UUID REFERENCES usuarios(id);
ALTER TABLE eventos ADD COLUMN IF NOT EXISTS lugar TEXT;
ALTER TABLE eventos ADD COLUMN IF NOT EXISTS enlace TEXT;
ALTER TABLE eventos ADD COLUMN IF NOT EXISTS imagen_url TEXT;
ALTER TABLE eventos ADD COLUMN IF NOT EXISTS enlace_inscripcion TEXT;
ALTER TABLE eventos ADD COLUMN IF NOT EXISTS estado TEXT NOT NULL DEFAULT 'borrador';
ALTER TABLE eventos ADD COLUMN IF NOT EXISTS creado_en TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE eventos ADD COLUMN IF NOT EXISTS actualizado_en TIMESTAMPTZ NOT NULL DEFAULT now();
ALTER TABLE eventos ALTER COLUMN fecha DROP NOT NULL;
ALTER TABLE eventos ALTER COLUMN semillero_id DROP NOT NULL;
UPDATE eventos SET inicia_en=fecha WHERE inicia_en IS NULL;
CREATE INDEX IF NOT EXISTS eventos_agenda_idx ON eventos(estado, tipo, inicia_en);

CREATE TABLE cronograma_reuniones (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    titulo TEXT NOT NULL,
    fecha DATE NOT NULL,
    hora TIME NOT NULL,
    semillero_id UUID NOT NULL REFERENCES semilleros(id),
    organizador_id UUID NOT NULL REFERENCES usuarios(id),
    notificacion_enviada BOOLEAN NOT NULL DEFAULT false
);

CREATE TABLE colaboradores_externos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nombre TEXT NOT NULL,
    email TEXT,
    institucion TEXT,
    intereses TEXT,
    fecha_registro TIMESTAMPTZ NOT NULL DEFAULT now(),
    semillero_id UUID NOT NULL REFERENCES semilleros(id)
);

CREATE INDEX usuarios_email_idx ON usuarios(email);
CREATE INDEX solicitudes_usuario_idx ON solicitudes_registro(usuario_id);
CREATE INDEX solicitudes_estado_idx ON solicitudes_registro(estado);
CREATE INDEX solicitudes_fecha_idx ON solicitudes_registro(fecha_radicacion DESC);
CREATE INDEX noticias_fecha_idx ON noticias(fecha_publicacion DESC);
