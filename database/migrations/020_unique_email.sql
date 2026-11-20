-- D033: one account per e-mail, case-insensitive. Safe to re-run.
-- Aborts without changes and lists the duplicated addresses (count only, not the rows) if any exist,
-- so duplicates are merged deliberately (e.g. an imported SGI member claimed by its Firebase account).
-- The same block is appended to database/init.sql for new databases.
DO $$
DECLARE
  dupes text;
BEGIN
  SELECT string_agg(format('%s cuentas', n), ', ') INTO dupes
  FROM (SELECT count(*) AS n FROM usuarios GROUP BY lower(email) HAVING count(*) > 1) d;
  IF dupes IS NOT NULL THEN
    RAISE EXCEPTION 'Hay correos repetidos (sin distinguir mayúsculas): %. Consulta: SELECT lower(email), count(*) FROM usuarios GROUP BY 1 HAVING count(*) > 1;', dupes;
  END IF;
  CREATE UNIQUE INDEX IF NOT EXISTS usuarios_email_lower_key ON usuarios (lower(email));
END $$;
