-- M1: enlace a la imagen de perfil del usuario; solo se guarda la URL, no el archivo.
ALTER TABLE usuarios ADD COLUMN IF NOT EXISTS foto_url TEXT;
