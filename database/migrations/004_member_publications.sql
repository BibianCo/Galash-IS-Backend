-- Módulo 5: publicaciones visibles asociadas a perfiles del directorio.
CREATE TABLE IF NOT EXISTS publicaciones (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    usuario_id UUID NOT NULL REFERENCES usuarios(id),
    titulo TEXT NOT NULL,
    descripcion TEXT,
    url TEXT,
    publicado_en TIMESTAMPTZ NOT NULL DEFAULT now(),
    visible BOOLEAN NOT NULL DEFAULT true
);

CREATE INDEX IF NOT EXISTS publicaciones_usuario_idx
    ON publicaciones(usuario_id, publicado_en DESC);
