-- Ejecutar en instalaciones existentes. La misma migración está incluida en init.sql.
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
