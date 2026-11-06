CREATE TABLE IF NOT EXISTS audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id UUID REFERENCES usuarios(id),
    action TEXT NOT NULL,
    entity TEXT NOT NULL,
    entity_id UUID,
    result TEXT NOT NULL,
    origin TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS audit_logs_actor_idx ON audit_logs(actor_id, created_at DESC);
CREATE INDEX IF NOT EXISTS audit_logs_action_idx ON audit_logs(action, created_at DESC);
CREATE INDEX IF NOT EXISTS audit_logs_entity_idx ON audit_logs(entity, created_at DESC);
CREATE INDEX IF NOT EXISTS audit_logs_date_idx ON audit_logs(created_at DESC);

CREATE OR REPLACE FUNCTION prevent_audit_log_mutation() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'audit_logs es inmutable';
END $$;
DROP TRIGGER IF EXISTS audit_logs_immutable_trigger ON audit_logs;
CREATE TRIGGER audit_logs_immutable_trigger
    BEFORE UPDATE OR DELETE ON audit_logs FOR EACH ROW EXECUTE FUNCTION prevent_audit_log_mutation();
