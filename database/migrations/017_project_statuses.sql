-- Projects have only Activo and Finalizado; application history is preserved.
BEGIN;
INSERT INTO proyecto_historial(proyecto_id,autor_id,campo,valor)
SELECT id,NULL,'estado','Activo' FROM proyectos WHERE estado='Radicado';
UPDATE proyectos SET estado='Activo',actualizado_en=now() WHERE estado='Radicado';
ALTER TABLE proyectos ALTER COLUMN estado SET DEFAULT 'Activo';
DO $$ BEGIN
 IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conrelid='proyectos'::regclass AND conname='proyectos_estado_check') THEN
  ALTER TABLE proyectos ADD CONSTRAINT proyectos_estado_check CHECK (estado IN ('Activo','Finalizado'));
 END IF;
END $$;
COMMIT;
