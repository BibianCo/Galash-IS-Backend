-- Proyectos del grupo importados de GrupLAC no pertenecen a un semillero.
-- TODO: permitir asignar a qué semillero pertenece cada proyecto del grupo.
ALTER TABLE proyectos ALTER COLUMN semillero_id DROP NOT NULL;
