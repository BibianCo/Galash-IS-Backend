CREATE TABLE IF NOT EXISTS contenido_paginas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug TEXT NOT NULL UNIQUE,
    titulo TEXT NOT NULL,
    contenido TEXT NOT NULL,
    estado TEXT NOT NULL DEFAULT 'borrador',
    autor_id UUID NOT NULL REFERENCES usuarios(id),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT now(),
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT now(),
    publicado_en TIMESTAMPTZ
);
CREATE INDEX IF NOT EXISTS contenido_paginas_estado_idx ON contenido_paginas(estado, slug);
