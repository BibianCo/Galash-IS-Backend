-- D040: application role with least privilege. The API connects as galash_app, never as the
-- superuser created by POSTGRES_USER. Run once per database as the owner/superuser:
--   psql "$ADMIN_DATABASE_URL" -v app_password="$GALASH_APP_PASSWORD" -f database/migrations/018_app_role.sql
-- then set DATABASE_URL=postgres://galash_app:<password>@host:port/db for the API.
-- Re-running is safe: it only (re)applies grants and the password.
\set ON_ERROR_STOP on
SELECT format('CREATE ROLE galash_app LOGIN PASSWORD %L NOSUPERUSER NOCREATEDB NOCREATEROLE', :'app_password')
WHERE NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'galash_app') \gexec
SELECT format('ALTER ROLE galash_app LOGIN PASSWORD %L NOSUPERUSER NOCREATEDB NOCREATEROLE', :'app_password') \gexec

SELECT format('GRANT CONNECT ON DATABASE %I TO galash_app', current_database()) \gexec
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
GRANT USAGE ON SCHEMA public TO galash_app;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO galash_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO galash_app;
-- The audit trail is append-only for the application.
REVOKE UPDATE, DELETE ON audit_logs FROM galash_app;
-- Tables created later by the owner get the same grants.
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO galash_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO galash_app;
