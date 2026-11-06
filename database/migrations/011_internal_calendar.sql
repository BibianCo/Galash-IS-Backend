CREATE TABLE IF NOT EXISTS cronograma_interno (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    semillero_id UUID NOT NULL REFERENCES semilleros(id),
    tipo TEXT NOT NULL CHECK (tipo IN ('reunion','tutoria','hito')),
    titulo TEXT NOT NULL,
    descripcion TEXT,
    inicia_en TIMESTAMPTZ NOT NULL,
    termina_en TIMESTAMPTZ NOT NULL,
    organizador_id UUID NOT NULL REFERENCES usuarios(id),
    estado TEXT NOT NULL DEFAULT 'activo' CHECK (estado IN ('activo','cancelado')),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT now(),
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS cronograma_interno_participantes (
    evento_id UUID NOT NULL REFERENCES cronograma_interno(id) ON DELETE CASCADE,
    usuario_id UUID NOT NULL REFERENCES usuarios(id),
    PRIMARY KEY (evento_id, usuario_id)
);
CREATE TABLE IF NOT EXISTS cronograma_interno_historial (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    evento_id UUID NOT NULL REFERENCES cronograma_interno(id) ON DELETE CASCADE,
    accion TEXT NOT NULL,
    actor_id UUID REFERENCES usuarios(id),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS cronograma_interno_agenda_idx ON cronograma_interno(semillero_id, inicia_en, estado);
