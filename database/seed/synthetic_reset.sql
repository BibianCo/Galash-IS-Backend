-- Deletes only the synthetic data loaded by synthetic.sql (ids that start with 5eed).
-- Real rows created through the API that reference synthetic rows are deleted too.
BEGIN;
DELETE FROM seguimiento_solicitud WHERE id::text LIKE '5eed%' OR solicitud_id::text LIKE '5eed%' OR responsable_id::text LIKE '5eed%';
DELETE FROM solicitudes_registro WHERE id::text LIKE '5eed%' OR semillero_id::text LIKE '5eed%';
DELETE FROM eventos WHERE id::text LIKE '5eed%' OR organizador_id::text LIKE '5eed%' OR semillero_id::text LIKE '5eed%';
DELETE FROM actividades WHERE id::text LIKE '5eed%' OR semillero_id::text LIKE '5eed%';
DELETE FROM proyecto_historial WHERE proyecto_id IN (SELECT id FROM proyectos WHERE id::text LIKE '5eed%' OR semillero_id::text LIKE '5eed%');
DELETE FROM proyectos WHERE id::text LIKE '5eed%' OR semillero_id::text LIKE '5eed%';
DELETE FROM semillero_resenas WHERE semillero_id::text LIKE '5eed%' OR autor_id::text LIKE '5eed%';
DELETE FROM semillero_miembros WHERE semillero_id::text LIKE '5eed%' OR usuario_id::text LIKE '5eed%';
UPDATE usuarios SET semillero_id = NULL WHERE semillero_id::text LIKE '5eed%';
DELETE FROM semilleros WHERE id::text LIKE '5eed%';
DELETE FROM perfil_investigador WHERE usuario_id::text LIKE '5eed%';
DELETE FROM sesiones WHERE usuario_id::text LIKE '5eed%';
DELETE FROM usuarios WHERE id::text LIKE '5eed%';
COMMIT;
