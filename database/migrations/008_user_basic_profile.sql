-- M1/M3: datos básicos en la cuenta; la solicitud referencia al usuario y no los copia.
ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS codigo_estudiantil TEXT UNIQUE;
ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS fecha_nacimiento DATE;
ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS telefono TEXT;

-- Las solicitudes nuevas leen estos datos de usuarios; las anónimas antiguas los conservan.
ALTER TABLE solicitudes_registro
    ALTER COLUMN nombre DROP NOT NULL,
    ALTER COLUMN cc DROP NOT NULL,
    ALTER COLUMN cod DROP NOT NULL,
    ALTER COLUMN telefono DROP NOT NULL,
    ALTER COLUMN fecha_nacimiento DROP NOT NULL;
