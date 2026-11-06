package links_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	postgresadapter "github.com/galash-uptc/backend/internal/adapters/outbound/postgres"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type dbFixture struct {
	repo   *postgresadapter.OfficialLinkRepository
	db     *pgxpool.Pool
	admin  string
	prefix string
}

func seedDB(t *testing.T) dbFixture {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	f := dbFixture{repo: postgresadapter.NewOfficialLinkRepository(db), db: db, prefix: fmt.Sprintf("m10-%d", time.Now().UnixNano())}
	if err := db.QueryRow(ctx, `INSERT INTO usuarios (firebase_uid,nombre,apellido,email,rol,estado) VALUES ($1,'Prueba','Admin',$1||'@example.test','admin','activo') RETURNING id`, f.prefix).Scan(&f.admin); err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	t.Cleanup(func() {
		for _, sql := range []string{
			`DELETE FROM enlaces_oficiales_historial WHERE cambiado_por=$1`,
			`DELETE FROM enlaces_oficiales WHERE actualizado_por=$1`,
			`DELETE FROM usuarios WHERE id=$1`,
		} {
			if _, err := db.Exec(ctx, sql, f.admin); err != nil {
				t.Errorf("cleanup %q: %v", sql, err)
			}
		}
		db.Close()
	})
	return f
}

func (f dbFixture) link(name, category string, order int) domain.OfficialLink {
	return domain.OfficialLink{Name: f.prefix + "-" + name, Description: "d", URL: "https://example.com/" + name, Category: category, Visible: true, Order: order, Status: "activo", UpdatedBy: f.admin}
}

func (f dbFixture) mine(list []domain.OfficialLink) []string {
	out := []string{}
	for _, l := range list {
		if len(l.Name) > len(f.prefix) && l.Name[:len(f.prefix)] == f.prefix {
			out = append(out, l.Name[len(f.prefix)+1:])
		}
	}
	return out
}

// TDB-T01 / TDB-T08 / TDB-T09: persistence, public visibility, ordering and history.
func TestRepositoryVisibilityOrderAndHistory(t *testing.T) {
	f := seedDB(t)
	ctx := context.Background()
	b, err := f.repo.Create(ctx, f.link("b", "cvlac", 2))
	if err != nil || b.ID == "" || b.UpdatedBy != f.admin || b.CreatedAt.IsZero() {
		t.Fatalf("create = %+v, %v", b, err)
	}
	a, _ := f.repo.Create(ctx, f.link("a", "cvlac", 1))
	hidden := f.link("hidden", "cvlac", 0)
	hidden.Visible = false
	if _, err := f.repo.Create(ctx, hidden); err != nil {
		t.Fatalf("create hidden: %v", err)
	}
	other, _ := f.repo.Create(ctx, f.link("other", "gruplac", 0))
	if _, err := f.repo.SetStatus(ctx, other.ID, f.admin, "inactivo"); err != nil {
		t.Fatalf("set status: %v", err)
	}

	public, err := f.repo.ListVisible(ctx, "", false)
	if got := f.mine(public); err != nil || fmt.Sprint(got) != "[a b]" {
		t.Fatalf("public list = %v, %v", got, err)
	}
	all, _ := f.repo.ListVisible(ctx, "", true)
	if got := f.mine(all); len(got) != 4 {
		t.Fatalf("admin list = %v", got)
	}
	byCategory, _ := f.repo.ListVisible(ctx, "gruplac", true)
	if got := f.mine(byCategory); fmt.Sprint(got) != "[other]" {
		t.Fatalf("category list = %v", got)
	}

	a.Order = 5
	updated, err := f.repo.Update(ctx, a.ID, f.admin, a)
	if err != nil || updated.Order != 5 || updated.UpdatedBy != f.admin || updated.UpdatedAt.Before(a.UpdatedAt) {
		t.Fatalf("update = %+v, %v", updated, err)
	}
	if _, err := f.repo.SetStatus(ctx, a.ID, f.admin, "oculto"); err != nil {
		t.Fatalf("hide: %v", err)
	}
	still, err := f.repo.Find(ctx, a.ID)
	if err != nil || still.Status != "oculto" || still.Visible {
		t.Fatalf("hidden link must remain stored: %+v, %v", still, err)
	}
	var history int
	if err := f.db.QueryRow(ctx, `SELECT count(*) FROM enlaces_oficiales_historial WHERE enlace_id=$1 AND cambiado_por=$2`, a.ID, f.admin).Scan(&history); err != nil || history != 3 {
		t.Fatalf("history rows = %d, %v; want create+update+status", history, err)
	}
}

// TDB-T02 / TDB-T03: missing or malformed IDs and CHECK constraints.
func TestRepositoryNotFoundAndConstraints(t *testing.T) {
	f := seedDB(t)
	ctx := context.Background()
	for _, id := range []string{"00000000-0000-0000-0000-000000000000", "no-es-uuid"} {
		if _, err := f.repo.Find(ctx, id); !errors.Is(err, domain.ErrOfficialLinkNotFound) {
			t.Fatalf("find %s: %v", id, err)
		}
		if _, err := f.repo.SetStatus(ctx, id, f.admin, "activo"); !errors.Is(err, domain.ErrOfficialLinkNotFound) {
			t.Fatalf("status %s: %v", id, err)
		}
	}
	bad := f.link("bad", "cvlac", -1)
	if _, err := f.repo.Create(ctx, bad); err == nil {
		t.Fatal("negative order accepted")
	}
	l, _ := f.repo.Create(ctx, f.link("ok", "cvlac", 0))
	if _, err := f.repo.SetStatus(ctx, l.ID, f.admin, "borrado"); err == nil {
		t.Fatal("invalid status accepted by CHECK")
	}
	if _, err := f.db.Exec(ctx, `UPDATE enlaces_oficiales SET actualizado_por=gen_random_uuid() WHERE id=$1`, l.ID); err == nil {
		t.Fatal("unknown author accepted by FK")
	}
}
