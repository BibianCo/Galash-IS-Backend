-- Applies the seedbed membership schema to databases created before module 2.
-- Membership rows are historical: removal sets retirado_en instead of deleting.
CREATE TABLE IF NOT EXISTS semillero_miembros (
    semillero_id UUID NOT NULL REFERENCES semilleros(id),
    usuario_id UUID NOT NULL REFERENCES usuarios(id),
    ingreso_en TIMESTAMPTZ NOT NULL DEFAULT now(),
    retirado_en TIMESTAMPTZ,
    PRIMARY KEY (semillero_id, usuario_id)
);

CREATE INDEX IF NOT EXISTS semillero_miembros_usuario_idx
    ON semillero_miembros(usuario_id);
