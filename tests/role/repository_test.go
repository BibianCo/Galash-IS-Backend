package role_test

import (
	"context"
	"os"
	"testing"
	"time"

	postgresadapter "github.com/galash-uptc/backend/internal/adapters/outbound/postgres"
	"github.com/jackc/pgx/v5/pgxpool"
)

// newRepository connects to TEST_DATABASE_URL and restores the visitante
// permissions and history the test changes.
func newRepository(t *testing.T) (*postgresadapter.RoleRepository, *pgxpool.Pool) {
	t.Helper()
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	repo := postgresadapter.NewRoleRepository(db)
	original, err := repo.RolePermissions(ctx, "visitante")
	if err != nil {
		t.Fatalf("read visitante permissions: %v", err)
	}
	start := time.Now()
	t.Cleanup(func() {
		ids := make([]string, 0, len(original))
		for _, p := range original {
			ids = append(ids, p.ID)
		}
		_ = repo.SetRolePermissions(ctx, "visitante", ids, "")
		_, _ = db.Exec(ctx, `DELETE FROM rol_permisos_historial WHERE rol = 'visitante' AND creado_en >= $1`, start)
		db.Close()
	})
	return repo, db
}

func TestPermissionCatalogIsSeeded(t *testing.T) {
	repo, _ := newRepository(t)
	items, err := repo.Permissions(context.Background())
	if err != nil {
		t.Fatalf("Permissions: %v", err)
	}
	seen := map[string]bool{}
	for _, p := range items {
		if p.Description == "" {
			t.Errorf("permission %s has no description", p.ID)
		}
		seen[p.ID] = true
	}
	for _, id := range []string{"roles.manage", "audit.read", "seedbeds.manage"} {
		if !seen[id] {
			t.Errorf("catalog missing %s: %+v", id, items)
		}
	}
	admin, err := repo.RolePermissions(context.Background(), "admin")
	if err != nil || len(admin) != len(items) {
		t.Fatalf("admin permissions=%d err=%v, want all %d", len(admin), err, len(items))
	}
}

func TestSetRolePermissionsReplacesSetAndRecordsHistory(t *testing.T) {
	repo, db := newRepository(t)
	ctx := context.Background()
	if err := repo.SetRolePermissions(ctx, "visitante", []string{"files.upload", "audit.read"}, ""); err != nil {
		t.Fatalf("first set: %v", err)
	}
	if err := repo.SetRolePermissions(ctx, "visitante", []string{"files.upload"}, ""); err != nil {
		t.Fatalf("second set: %v", err)
	}
	items, err := repo.RolePermissions(ctx, "visitante")
	if err != nil || len(items) != 1 || items[0].ID != "files.upload" {
		t.Fatalf("visitante permissions=%+v err=%v, want only files.upload", items, err)
	}
	var history int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM rol_permisos_historial WHERE rol = 'visitante' AND creado_en >= now() - interval '1 minute'`).Scan(&history); err != nil || history < 2 {
		t.Fatalf("history rows=%d err=%v, want at least 2", history, err)
	}
}

func TestSetRolePermissionsRejectsUnknownPermissionAtomically(t *testing.T) {
	repo, _ := newRepository(t)
	ctx := context.Background()
	before, _ := repo.RolePermissions(ctx, "visitante")
	if err := repo.SetRolePermissions(ctx, "visitante", []string{"files.upload", "no.existe"}, ""); err == nil {
		t.Fatal("unknown permission accepted")
	}
	after, _ := repo.RolePermissions(ctx, "visitante")
	if len(after) != len(before) {
		t.Fatalf("failed replace changed permissions: before=%+v after=%+v", before, after)
	}
}
