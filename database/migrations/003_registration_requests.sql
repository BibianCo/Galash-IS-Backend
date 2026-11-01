-- Módulo 3: permite radicación pública y conserva compatibilidad con el esquema inicial.
ALTER TABLE solicitudes_registro
    ALTER COLUMN usuario_id DROP NOT NULL,
    ALTER COLUMN semillero_id DROP NOT NULL;

-- fecha_envio solo existe en el esquema inicial; las bases creadas después usan fecha_radicacion.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'solicitudes_registro' AND column_name = 'fecha_envio'
    ) THEN
        ALTER TABLE solicitudes_registro ALTER COLUMN fecha_envio SET DEFAULT now();
    END IF;
END $$;

ALTER TABLE seguimiento_solicitud
    ALTER COLUMN responsable_id DROP NOT NULL;

DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'solicitudes_registro' AND column_name = 'convocatoria_min_ciencias'
    ) AND NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'solicitudes_registro' AND column_name = 'convocatoria_minciencias'
    ) THEN
        ALTER TABLE solicitudes_registro RENAME COLUMN convocatoria_min_ciencias TO convocatoria_minciencias;
    END IF;
END $$;

UPDATE solicitudes_registro SET estado = 'Radicada' WHERE estado = 'pendiente';
UPDATE solicitudes_registro SET estado = 'En revisión' WHERE estado = 'Informada';
UPDATE solicitudes_registro SET estado = 'Pendiente de información' WHERE estado = 'Detenida';
