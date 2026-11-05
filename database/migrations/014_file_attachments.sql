CREATE TABLE IF NOT EXISTS archivos_adjuntos (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entidad_tipo TEXT NOT NULL,
    entidad_id UUID NOT NULL,
    nombre TEXT NOT NULL,
    tipo_contenido TEXT NOT NULL,
    tamano BIGINT NOT NULL CHECK (tamano > 0 AND tamano <= 10485760),
    contenido BYTEA NOT NULL,
    subido_por UUID NOT NULL REFERENCES usuarios(id),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS archivos_adjuntos_entidad_idx ON archivos_adjuntos(entidad_tipo, entidad_id);
