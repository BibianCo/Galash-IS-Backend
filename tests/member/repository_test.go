package member_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	postgresadapter "github.com/galash-uptc/backend/internal/adapters/outbound/postgres"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
	"github.com/jackc/pgx/v5/pgxpool"
)

type fixture struct {
	ctx  context.Context
	db   *pgxpool.Pool
	repo *postgresadapter.UserRepository
	tag  string
}

// newFixture connects to TEST_DATABASE_URL and removes every row tagged by the test on cleanup.
func newFixture(t *testing.T) fixture {
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
	f := fixture{ctx: ctx, db: db, repo: postgresadapter.NewUserRepository(db), tag: fmt.Sprintf("member-test-%d", time.Now().UnixNano())}
	t.Cleanup(func() {
		users := `SELECT id FROM usuarios WHERE firebase_uid LIKE $1`
		for _, q := range []string{
			`DELETE FROM publicaciones WHERE usuario_id IN (` + users + `)`,
			`DELETE FROM semillero_miembros WHERE usuario_id IN (` + users + `)`,
			`DELETE FROM perfil_investigador WHERE usuario_id IN (` + users + `)`,
			`DELETE FROM usuarios WHERE firebase_uid LIKE $1`,
			`DELETE FROM semilleros WHERE nombre LIKE $1`,
		} {
			_, _ = db.Exec(ctx, q, f.tag+"%")
		}
		db.Close()
	})
	return f
}

func (f fixture) seedbed(t *testing.T, name string) string {
	t.Helper()
	var id string
	if err := f.db.QueryRow(f.ctx, `INSERT INTO semilleros(nombre) VALUES($1) RETURNING id`, f.tag+"-"+name).Scan(&id); err != nil {
		t.Fatalf("create seedbed: %v", err)
	}
	return id
}

func (f fixture) member(t *testing.T, name string, seedbeds ...string) domain.MemberProfile {
	t.Helper()
	p, err := f.repo.CreateMember(f.ctx, ports.CreateMemberCommand{
		FirebaseUID: f.tag + "-" + name, Name: name, LastName: "Prueba", Email: f.tag + "-" + name + "@example.test",
		Role: "estudiante", CvLAC: "https://scienti.minciencias.gov.co/cvlac", VisiblePublicly: true, SeedbedIDs: seedbeds,
	})
	if err != nil {
		t.Fatalf("CreateMember(%s): %v", name, err)
	}
	return p
}

func TestCreateUpdateAndDeactivateKeepProfileData(t *testing.T) {
	f := newFixture(t)
	s1 := f.seedbed(t, "s1")
	p := f.member(t, "ana", s1)
	if p.Status != "activo" || p.CvLAC == "" || len(p.SeedbedIDs) != 1 || p.SeedbedIDs[0] != s1 {
		t.Fatalf("created profile=%+v", p)
	}
	hidden := false
	p, err := f.repo.Update(f.ctx, p.ID, ports.UpdateMemberCommand{Area: "IA", VisiblePublicly: &hidden})
	if err != nil {
		t.Fatalf("Update(): %v", err)
	}
	if p.Area != "IA" || p.VisiblePublicly || p.Name != "ana" || p.CvLAC == "" || len(p.SeedbedIDs) != 1 {
		t.Fatalf("partial update lost or ignored fields: %+v", p)
	}
	p, err = f.repo.SetStatus(f.ctx, p.ID, "inactivo")
	if err != nil {
		t.Fatalf("SetStatus(): %v", err)
	}
	if p.Status != "inactivo" || len(p.SeedbedIDs) != 1 {
		t.Fatalf("deactivation changed related data: %+v", p)
	}
}

func TestListExcludesInactiveAndHiddenEvenWhenSearchMatches(t *testing.T) {
	f := newFixture(t)
	f.member(t, "visible")
	off := f.member(t, "off")
	if _, err := f.repo.SetStatus(f.ctx, off.ID, "inactivo"); err != nil {
		t.Fatal(err)
	}
	hidden := false
	h := f.member(t, "hidden")
	if _, err := f.repo.Update(f.ctx, h.ID, ports.UpdateMemberCommand{VisiblePublicly: &hidden}); err != nil {
		t.Fatal(err)
	}
	// The tag is only in the email, so the search must still apply the status filters.
	items, err := f.repo.List(f.ctx, ports.MemberFilter{Query: f.tag, Limit: 20})
	if err != nil {
		t.Fatalf("List(): %v", err)
	}
	if len(items) != 1 || items[0].Name != "visible" {
		t.Fatalf("List() returned %d profiles, want only visible: %+v", len(items), items)
	}
	if items[0].Email != "" || items[0].FirebaseUID != "" {
		t.Errorf("directory exposes private fields: %+v", items[0])
	}
}

func TestListFiltersBySeedbedTypeAndArea(t *testing.T) {
	f := newFixture(t)
	s1 := f.seedbed(t, "s1")
	in := f.member(t, "in", s1)
	f.member(t, "out")
	if _, err := f.repo.Update(f.ctx, in.ID, ports.UpdateMemberCommand{Area: "Inteligencia artificial"}); err != nil {
		t.Fatal(err)
	}
	for name, filter := range map[string]ports.MemberFilter{
		"seedbed": {SeedbedID: s1, Limit: 20},
		"area":    {Query: f.tag, Area: "inteligencia", Limit: 20},
		"type":    {Query: f.tag + "-in", Role: "Estudiante", Limit: 20},
	} {
		items, err := f.repo.List(f.ctx, filter)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if len(items) != 1 || items[0].ID != in.ID {
			t.Errorf("%s: got %+v, want only %s", name, items, in.ID)
		}
	}
}

func TestMissingOrMalformedMemberIDIsNotFound(t *testing.T) {
	f := newFixture(t)
	missing := "00000000-0000-0000-0000-000000000000"
	checks := map[string]func() error{
		"Find malformed":    func() error { _, err := f.repo.Find(f.ctx, "not-a-uuid"); return err },
		"Find missing":      func() error { _, err := f.repo.Find(f.ctx, missing); return err },
		"SetStatus missing": func() error { _, err := f.repo.SetStatus(f.ctx, missing, "inactivo"); return err },
		"Update missing":    func() error { _, err := f.repo.Update(f.ctx, missing, ports.UpdateMemberCommand{Area: "IA"}); return err },
	}
	for name, check := range checks {
		if err := check(); !errors.Is(err, domain.ErrMemberNotFound) {
			t.Errorf("%s: err=%v, want ErrMemberNotFound", name, err)
		}
	}
}

func TestUpdateSeedbedsKeepsMembershipHistory(t *testing.T) {
	f := newFixture(t)
	s1, s2 := f.seedbed(t, "s1"), f.seedbed(t, "s2")
	p := f.member(t, "ana", s1)
	p, err := f.repo.Update(f.ctx, p.ID, ports.UpdateMemberCommand{SeedbedIDs: []string{s2}})
	if err != nil {
		t.Fatalf("Update(): %v", err)
	}
	if len(p.SeedbedIDs) != 1 || p.SeedbedIDs[0] != s2 {
		t.Fatalf("current seedbeds=%v, want [%s]", p.SeedbedIDs, s2)
	}
	var left int
	if err := f.db.QueryRow(f.ctx, `SELECT count(*) FROM semillero_miembros WHERE usuario_id=$1 AND semillero_id=$2 AND retirado_en IS NOT NULL`, p.ID, s1).Scan(&left); err != nil {
		t.Fatal(err)
	}
	if left != 1 {
		t.Fatalf("withdrawn membership rows for previous seedbed=%d, want 1 kept with retirado_en", left)
	}
}

func TestPublicationsReturnOnlyVisibleNewestFirst(t *testing.T) {
	f := newFixture(t)
	p := f.member(t, "ana")
	for _, q := range []string{
		`INSERT INTO publicaciones(usuario_id,titulo,publicado_en) VALUES($1,'Antigua','2025-01-01Z')`,
		`INSERT INTO publicaciones(usuario_id,titulo,publicado_en) VALUES($1,'Reciente','2026-01-01Z')`,
		`INSERT INTO publicaciones(usuario_id,titulo,visible) VALUES($1,'Oculta',false)`,
	} {
		if _, err := f.db.Exec(f.ctx, q, p.ID); err != nil {
			t.Fatal(err)
		}
	}
	items, err := f.repo.Publications(f.ctx, p.ID)
	if err != nil {
		t.Fatalf("Publications(): %v", err)
	}
	if len(items) != 2 || items[0].Title != "Reciente" || items[1].Title != "Antigua" {
		t.Fatalf("publications=%+v, want [Reciente Antigua]", items)
	}
	other := f.member(t, "sin")
	if items, err := f.repo.Publications(f.ctx, other.ID); err != nil || items == nil || len(items) != 0 {
		t.Fatalf("empty publications=%v err=%v, want []", items, err)
	}
}
