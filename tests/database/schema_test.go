package database_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// fixture creates synthetic rows inside the test transaction; every case rolls back.
const fixture = `
INSERT INTO usuarios (id,firebase_uid,nombre,email,dni) VALUES
  ('00000000-0000-0000-0000-0000000000a1','tdb-uid-1','Prueba','tdb1@example.test','tdb-dni-1');
INSERT INTO semilleros (id,nombre,lider_id) VALUES
  ('00000000-0000-0000-0000-0000000000b1','tdb-semillero','00000000-0000-0000-0000-0000000000a1');
INSERT INTO proyectos (id,titulo,semillero_id) VALUES
  ('00000000-0000-0000-0000-0000000000c1','tdb-proyecto','00000000-0000-0000-0000-0000000000b1');
INSERT INTO proyecto_historial (proyecto_id,campo,valor) VALUES
  ('00000000-0000-0000-0000-0000000000c1','estado','Activo');
INSERT INTO audit_logs (action,entity,result) VALUES ('tdb','tdb','ok');`

func begin(t *testing.T) pgx.Tx {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	tx, err := conn.Begin(ctx)
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	t.Cleanup(func() { _ = tx.Rollback(ctx); _ = conn.Close(ctx) })
	if _, err := tx.Exec(ctx, fixture); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	return tx
}

func sqlState(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

// TDB-T01: the schema created by init.sql has the 31 expected base tables.
func TestSchemaHasExpectedTables(t *testing.T) {
	tx := begin(t)
	var n int
	if err := tx.QueryRow(context.Background(), `SELECT count(*) FROM information_schema.tables WHERE table_schema='public' AND table_type='BASE TABLE'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 31 {
		t.Fatalf("base tables = %d, want 31", n)
	}
}

// TDB-T01/T02/T03/T06: constraints reject invalid writes with the expected SQLSTATE.
func TestSchemaRejectsInvalidWrites(t *testing.T) {
	for name, c := range map[string]struct{ sql, state string }{
		"duplicate dni":              {`INSERT INTO usuarios (firebase_uid,nombre,email,dni) VALUES ('tdb-uid-2','Otra','tdb2@example.test','tdb-dni-1')`, "23505"},
		"duplicate firebase_uid":     {`INSERT INTO usuarios (firebase_uid,nombre,email) VALUES ('tdb-uid-1','Otra','tdb2@example.test')`, "23505"},
		"duplicate seedbed name":     {`INSERT INTO semilleros (nombre) VALUES ('tdb-semillero')`, "23505"},
		"null email":                 {`INSERT INTO usuarios (firebase_uid,nombre,email) VALUES ('tdb-uid-3','Otra',NULL)`, "23502"},
		"session for missing user":   {`INSERT INTO sesiones (usuario_id,token_hash,expira_en) VALUES ('00000000-0000-0000-0000-00000000dead','h',now())`, "23503"},
		"project in missing seedbed": {`INSERT INTO proyectos (titulo,semillero_id) VALUES ('x','00000000-0000-0000-0000-00000000dead')`, "23503"},
		"delete referenced user":     {`DELETE FROM usuarios WHERE id='00000000-0000-0000-0000-0000000000a1'`, "23503"},
		"progress above 100":         {`UPDATE proyectos SET avance=101 WHERE id='00000000-0000-0000-0000-0000000000c1'`, "23514"},
		"progress below 0":           {`UPDATE proyectos SET avance=-1 WHERE id='00000000-0000-0000-0000-0000000000c1'`, "23514"},
		"unknown project state":      {`UPDATE proyectos SET estado='Radicado' WHERE id='00000000-0000-0000-0000-0000000000c1'`, "23514"},
		"update audit log":           {`UPDATE audit_logs SET result='editado' WHERE action='tdb'`, "P0001"},
		"delete audit log":           {`DELETE FROM audit_logs WHERE action='tdb'`, "P0001"},
		"unknown role":               {`UPDATE usuarios SET rol='superadmin' WHERE firebase_uid='tdb-uid-1'`, "23514"},
		"unknown account state":      {`UPDATE usuarios SET estado='suspendido' WHERE firebase_uid='tdb-uid-1'`, "23514"},
		"unknown seedbed state":      {`UPDATE semilleros SET visibilidad='Activo' WHERE nombre='tdb-semillero'`, "23514"},
		"unknown event state":        {`INSERT INTO eventos (titulo,estado) VALUES ('t','archivado')`, "23514"},
		"event ends before start":    {`INSERT INTO eventos (titulo,inicia_en,termina_en) VALUES ('t','2030-01-02','2030-01-01')`, "23514"},
		"project ends before start":  {`UPDATE proyectos SET fecha_inicio='2030-01-02', fecha_fin='2030-01-01' WHERE titulo='tdb-proyecto'`, "23514"},
		"withdrawn before joining":   {`INSERT INTO semillero_miembros (semillero_id,usuario_id,ingreso_en,retirado_en) VALUES ('00000000-0000-0000-0000-0000000000b1','00000000-0000-0000-0000-0000000000a1','2030-01-02','2030-01-01')`, "23514"},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := begin(t).Exec(context.Background(), c.sql)
			if got := sqlState(err); got != c.state {
				t.Fatalf("SQLSTATE = %q (%v), want %s", got, err, c.state)
			}
		})
	}
}

// TDB-T01: deleting a project cascades only to its dependent history.
func TestProjectDeleteCascadesHistory(t *testing.T) {
	tx, ctx := begin(t), context.Background()
	if _, err := tx.Exec(ctx, `DELETE FROM proyectos WHERE id='00000000-0000-0000-0000-0000000000c1'`); err != nil {
		t.Fatalf("delete project: %v", err)
	}
	var n int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM proyecto_historial WHERE proyecto_id='00000000-0000-0000-0000-0000000000c1'`).Scan(&n); err != nil || n != 0 {
		t.Fatalf("history rows = %d (%v), want 0", n, err)
	}
}

// TDB-T15 / D040: the API role (migration 018) is not a superuser, cannot create objects and cannot rewrite the audit trail.
func TestAppRoleHasLeastPrivilege(t *testing.T) {
	tx, ctx := begin(t), context.Background()
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='galash_app')`).Scan(&exists); err != nil || !exists {
		t.Fatalf("role galash_app missing (apply database/migrations/018_app_role.sql): %v", err)
	}
	for check, want := range map[string]bool{
		`SELECT rolsuper OR rolcreatedb OR rolcreaterole FROM pg_roles WHERE rolname='galash_app'`: false,
		`SELECT has_schema_privilege('galash_app','public','CREATE')`:                              false,
		`SELECT has_table_privilege('galash_app','audit_logs','UPDATE')`:                           false,
		`SELECT has_table_privilege('galash_app','audit_logs','DELETE')`:                           false,
		`SELECT has_table_privilege('galash_app','audit_logs','INSERT')`:                           true,
		`SELECT has_table_privilege('galash_app','usuarios','UPDATE')`:                             true,
		`SELECT has_table_privilege('galash_app','sesiones','DELETE')`:                             true,
	} {
		var got bool
		if err := tx.QueryRow(ctx, check).Scan(&got); err != nil || got != want {
			t.Errorf("%s = %v (%v), want %v", check, got, err, want)
		}
	}
}
