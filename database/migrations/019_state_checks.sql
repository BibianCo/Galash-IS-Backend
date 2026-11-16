-- D034 / D038: catalog CHECKs for state and role columns, and date-order CHECKs.
-- Safe to re-run. It first reports rows outside the catalog and aborts without changes if any exist,
-- so a database with legacy values is cleaned up deliberately, never silently rewritten.
-- The same block is appended to database/init.sql for new databases.
DO $$
DECLARE
  rules text[][] := ARRAY[
    ['usuarios', 'usuarios_rol_check', 'rol IN (''visitante'',''estudiante'',''profesor'',''docente'',''admin'')'],
    ['usuarios', 'usuarios_estado_check', 'estado IN (''activo'',''inactivo'')'],
    ['solicitudes_registro', 'solicitudes_registro_estado_check', 'estado IN (''Radicada'',''En revisión'',''Aprobada'',''Rechazada'',''Pendiente de información'')'],
    ['seguimiento_solicitud', 'seguimiento_solicitud_estado_check', 'estado IN (''Radicada'',''En revisión'',''Aprobada'',''Rechazada'',''Pendiente de información'')'],
    ['semilleros', 'semilleros_estado_check', 'estado IN (''Borrador'',''Publicado'',''Oculto'',''Inactivo'')'],
    ['semilleros', 'semilleros_visibilidad_check', 'visibilidad IN (''Borrador'',''Publicado'',''Oculto'',''Inactivo'')'],
    ['eventos', 'eventos_estado_check', 'estado IN (''borrador'',''publicado'',''cancelado'',''oculto'')'],
    ['comunicados', 'comunicados_estado_check', 'estado IN (''borrador'',''publicado'')'],
    ['contenido_paginas', 'contenido_paginas_estado_check', 'estado IN (''borrador'',''publicado'')'],
    ['semillero_resenas', 'semillero_resenas_estado_check', 'estado IN (''pendiente'',''publicada'',''oculta'',''rechazada'')'],
    ['actividades', 'actividades_estado_check', 'estado IN (''pendiente'',''próxima'',''en curso'',''finalizada'')'],
    ['enlaces_oficiales_historial', 'enlaces_oficiales_historial_estado_check', 'estado IN (''activo'',''inactivo'',''oculto'')'],
    ['proyectos', 'proyectos_fechas_check', 'fecha_fin IS NULL OR fecha_inicio IS NULL OR fecha_fin >= fecha_inicio'],
    ['eventos', 'eventos_fechas_check', 'termina_en IS NULL OR inicia_en IS NULL OR termina_en >= inicia_en'],
    ['cronograma_interno', 'cronograma_interno_fechas_check', 'termina_en > inicia_en'],
    ['semillero_miembros', 'semillero_miembros_fechas_check', 'retirado_en IS NULL OR ingreso_en IS NULL OR retirado_en >= ingreso_en']
  ];
  r text[];
  bad bigint;
  report text := '';
BEGIN
  FOREACH r SLICE 1 IN ARRAY rules LOOP
    EXECUTE format('SELECT count(*) FROM %I WHERE NOT (%s)', r[1], r[3]) INTO bad;
    IF bad > 0 THEN
      report := report || format(E'\n  %s: %s fila(s) incumplen %s', r[1], bad, r[3]);
    END IF;
  END LOOP;
  IF report <> '' THEN
    RAISE EXCEPTION 'Datos fuera de catálogo; corrígelos antes de aplicar 019:%', report;
  END IF;
  FOREACH r SLICE 1 IN ARRAY rules LOOP
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = r[2]) THEN
      EXECUTE format('ALTER TABLE %I ADD CONSTRAINT %I CHECK (%s)', r[1], r[2], r[3]);
    END IF;
  END LOOP;
END $$;
