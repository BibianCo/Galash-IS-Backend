-- Módulo 11: permisos asignables a los cuatro roles fijos del sistema.
-- Los roles no tienen tabla propia: el catálogo es fijo y vive en el dominio.
CREATE TABLE IF NOT EXISTS permisos (
    id TEXT PRIMARY KEY,
    descripcion TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS rol_permisos (
    rol TEXT NOT NULL CHECK (rol IN ('visitante', 'estudiante', 'profesor', 'admin')),
    permiso_id TEXT NOT NULL REFERENCES permisos(id),
    PRIMARY KEY (rol, permiso_id)
);

-- Cada reemplazo de permisos conserva actor, fecha y el conjunto resultante.
CREATE TABLE IF NOT EXISTS rol_permisos_historial (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rol TEXT NOT NULL,
    permisos TEXT[] NOT NULL,
    actor_id UUID REFERENCES usuarios(id),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO permisos (id, descripcion) VALUES
    ('seedbeds.manage', 'Gestionar semilleros y sus integrantes'),
    ('registrations.review', 'Revisar solicitudes de registro'),
    ('research.manage', 'Gestionar actividades, proyectos y propuestas'),
    ('members.manage', 'Gestionar perfiles de integrantes'),
    ('events.manage', 'Gestionar eventos y convocatorias'),
    ('content.manage', 'Gestionar información institucional'),
    ('announcements.manage', 'Gestionar comunicados'),
    ('calendar.manage', 'Gestionar el cronograma interno'),
    ('links.manage', 'Gestionar enlaces oficiales'),
    ('files.upload', 'Subir archivos adjuntos'),
    ('roles.manage', 'Administrar roles y permisos'),
    ('audit.read', 'Consultar la bitácora de auditoría')
ON CONFLICT (id) DO NOTHING;

INSERT INTO rol_permisos (rol, permiso_id)
SELECT 'admin', id FROM permisos
UNION ALL
SELECT 'profesor', unnest(ARRAY['seedbeds.manage', 'research.manage', 'events.manage', 'announcements.manage', 'calendar.manage', 'files.upload'])
UNION ALL
SELECT 'estudiante', 'files.upload'
ON CONFLICT DO NOTHING;
