CREATE TABLE IF NOT EXISTS comunicados (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    titulo TEXT NOT NULL,
    descripcion TEXT NOT NULL,
    adjuntos JSONB NOT NULL DEFAULT '[]',
    estado TEXT NOT NULL DEFAULT 'borrador',
    autor_id UUID NOT NULL REFERENCES usuarios(id),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT now(),
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT now(),
    publicado_en TIMESTAMPTZ
);
CREATE TABLE IF NOT EXISTS comunicados_lecturas (
    comunicado_id UUID NOT NULL REFERENCES comunicados(id) ON DELETE CASCADE,
    usuario_id UUID NOT NULL REFERENCES usuarios(id) ON DELETE CASCADE,
    confirmado_en TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (comunicado_id, usuario_id)
);
CREATE INDEX IF NOT EXISTS comunicados_estado_idx ON comunicados(estado, creado_en DESC);
