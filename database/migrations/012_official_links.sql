CREATE TABLE IF NOT EXISTS enlaces_oficiales (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    nombre TEXT NOT NULL,
    descripcion TEXT NOT NULL,
    url TEXT NOT NULL,
    categoria TEXT NOT NULL,
    visible BOOLEAN NOT NULL DEFAULT TRUE,
    orden INTEGER NOT NULL DEFAULT 0 CHECK (orden >= 0),
    estado TEXT NOT NULL DEFAULT 'activo' CHECK (estado IN ('activo','inactivo','oculto')),
    actualizado_por UUID NOT NULL REFERENCES usuarios(id),
    creado_en TIMESTAMPTZ NOT NULL DEFAULT now(),
    actualizado_en TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS enlaces_oficiales_historial (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(), enlace_id UUID NOT NULL REFERENCES enlaces_oficiales(id),
    nombre TEXT NOT NULL, descripcion TEXT NOT NULL, url TEXT NOT NULL, categoria TEXT NOT NULL,
    visible BOOLEAN NOT NULL, orden INTEGER NOT NULL, estado TEXT NOT NULL,
    cambiado_por UUID NOT NULL REFERENCES usuarios(id), cambiado_en TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE OR REPLACE FUNCTION registrar_historial_enlace() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  INSERT INTO enlaces_oficiales_historial(enlace_id,nombre,descripcion,url,categoria,visible,orden,estado,cambiado_por)
  VALUES(NEW.id,NEW.nombre,NEW.descripcion,NEW.url,NEW.categoria,NEW.visible,NEW.orden,NEW.estado,NEW.actualizado_por); RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS enlaces_oficiales_historial_trigger ON enlaces_oficiales;
CREATE TRIGGER enlaces_oficiales_historial_trigger AFTER INSERT OR UPDATE ON enlaces_oficiales FOR EACH ROW EXECUTE FUNCTION registrar_historial_enlace();
