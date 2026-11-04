-- Delegados del semillero: relación sobre la membresía, no un rol (máx. 2 por semillero, validado en la aplicación).
ALTER TABLE semillero_miembros ADD COLUMN IF NOT EXISTS delegado BOOLEAN NOT NULL DEFAULT false;
