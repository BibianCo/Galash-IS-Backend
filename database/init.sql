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
    rol TEXT NOT NULL DEFAULT 'estudiante',
    estado TEXT NOT NULL DEFAULT 'activo',
    semillero_id UUID,
    creado_en TIMESTAMPTZ NOT NULL DEFAULT now()
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

ALTER TABLE usuarios
    ADD CONSTRAINT usuarios_semillero_fk
    FOREIGN KEY (semillero_id) REFERENCES semilleros(id);

CREATE TABLE solicitudes_registro (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    usuario_id UUID NOT NULL REFERENCES usuarios(id),
    semillero_id UUID NOT NULL REFERENCES semilleros(id),
    estado TEXT NOT NULL DEFAULT 'pendiente',
    motivo TEXT,
    fecha_envio TIMESTAMPTZ NOT NULL DEFAULT now()
);

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

CREATE TABLE proyectos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    titulo TEXT NOT NULL,
    resumen TEXT,
    impacto TEXT,
    fecha_inicio DATE,
    fecha_fin DATE,
    semillero_id UUID NOT NULL REFERENCES semilleros(id),
    area_investigacion TEXT
);

CREATE TABLE actividades (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    titulo TEXT NOT NULL,
    descripcion TEXT,
    estado TEXT NOT NULL DEFAULT 'pendiente',
    fecha DATE,
    semillero_id UUID NOT NULL REFERENCES semilleros(id)
);

CREATE TABLE eventos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    titulo TEXT NOT NULL,
    descripcion TEXT,
    fecha TIMESTAMPTZ NOT NULL,
    semillero_id UUID NOT NULL REFERENCES semilleros(id)
);

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
CREATE INDEX noticias_fecha_idx ON noticias(fecha_publicacion DESC);
