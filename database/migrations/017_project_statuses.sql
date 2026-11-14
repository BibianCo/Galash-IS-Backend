-- Proyectos: únicos estados válidos Activo y Finalizado. Radicado pasa a Activo
-- conservando avance y datos, y el cambio queda en el historial del proyecto.
BEGIN;
INSERT INTO proyecto_historial (proyecto_id, autor_id, campo, valor)
SELECT id, NULL, 'estado', 'Activo' FROM proyectos WHERE estado = 'Radicado';
UPDATE proyectos SET estado = 'Activo', actualizado_en = now() WHERE estado = 'Radicado';
ALTER TABLE proyectos ALTER COLUMN estado SET DEFAULT 'Activo';
COMMIT;
