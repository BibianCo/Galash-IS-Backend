package research_test

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

// seed creates a user and a seedbed; cleanup removes everything they own.
func seed(t *testing.T) (*postgresadapter.ResearchRepository, *pgxpool.Pool, string, string) {
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
	run := fmt.Sprintf("research-%d", time.Now().UnixNano())
	var user, seedbed string
	if err := db.QueryRow(ctx, `INSERT INTO usuarios (firebase_uid,nombre,email,rol) VALUES ($1,'Prueba',$1||'@example.test','profesor') RETURNING id`, run).Scan(&user); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.QueryRow(ctx, `INSERT INTO semilleros (nombre,lider_id) VALUES ($1,$2) RETURNING id`, run, user).Scan(&seedbed); err != nil {
		t.Fatalf("seed seedbed: %v", err)
	}
	t.Cleanup(func() {
		for _, q := range []struct{ sql, id string }{
			{`DELETE FROM propuestas_investigacion WHERE autor_id=$1`, user},
			{`DELETE FROM proyecto_historial WHERE autor_id=$1`, user},
			{`DELETE FROM proyectos WHERE semillero_id=$1`, seedbed},
			{`DELETE FROM actividades WHERE semillero_id=$1`, seedbed},
			{`DELETE FROM semillero_resenas WHERE semillero_id=$1`, seedbed},
			{`DELETE FROM semilleros WHERE id=$1`, seedbed},
			{`DELETE FROM usuarios WHERE id=$1`, user},
		} {
			if _, err := db.Exec(ctx, q.sql, q.id); err != nil {
				t.Errorf("cleanup %q: %v", q, err)
			}
		}
		db.Close()
	})
	return postgresadapter.NewResearchRepository(db), db, user, seedbed
}

func TestProjectUpdatesKeepAuthorAndDateHistory(t *testing.T) {
	r, _, user, seedbed := seed(t)
	ctx := context.Background()
	p, err := r.CreateProject(ctx, domain.Project{SeedbedID: seedbed, Title: "P", Status: "Radicado", CreatedBy: user})
	if err != nil {
		t.Fatal(err)
	}
	p.Status = "Activo"
	if _, err := r.UpdateProject(ctx, p.ID, p, user, []domain.ProjectHistory{{Field: "estado", Value: "Activo"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := r.UpdateProgress(ctx, p.ID, 60, user); err != nil {
		t.Fatal(err)
	}
	got, err := r.FindProject(ctx, p.ID)
	if err != nil || got.Status != "Activo" || got.Progress != 60 || got.CreatedBy != user {
		t.Fatalf("project=%+v err=%v", got, err)
	}
	history, err := r.ProjectHistory(ctx, p.ID)
	if err != nil || len(history) != 2 {
		t.Fatalf("history=%+v err=%v", history, err)
	}
	for _, h := range history {
		if h.AuthorID != user || h.CreatedAt.IsZero() {
			t.Fatalf("history row without author/date: %+v", h)
		}
	}
	if _, err := r.UpdateProgress(ctx, p.ID, 101, user); err == nil {
		t.Fatal("database accepted progress 101")
	}
	if h, _ := r.ProjectHistory(ctx, p.ID); len(h) != 2 {
		t.Fatalf("failed progress left history rows: %d", len(h))
	}
}

func TestConvertProposalKeepsHistoryAndLinksProject(t *testing.T) {
	r, _, user, seedbed := seed(t)
	ctx := context.Background()
	p, err := r.CreateProposal(ctx, domain.ResearchProposal{SeedbedID: seedbed, Title: "T", Problem: "P", Objectives: "O", Methodology: "M", Status: "Radicada", AuthorID: user, Attachments: []string{"anexo.pdf"}})
	if err != nil {
		t.Fatal(err)
	}
	p.Status, p.Observations = "Aprobada", "cumple"
	if _, err := r.UpdateProposal(ctx, p.ID, p, user); err != nil {
		t.Fatal(err)
	}
	project, err := r.ConvertProposal(ctx, p.ID, user)
	if err != nil || project.Status != "Radicado" || project.SeedbedID != seedbed || project.Title != "T" {
		t.Fatalf("project=%+v err=%v", project, err)
	}
	got, err := r.FindProposal(ctx, p.ID)
	if err != nil || got.Status != "Convertida" || got.ProjectID == nil || *got.ProjectID != project.ID || len(got.Attachments) != 1 {
		t.Fatalf("proposal=%+v err=%v", got, err)
	}
	history, err := r.ProposalHistory(ctx, p.ID)
	if err != nil || len(history) != 3 || history[0].Status != "Radicada" || history[1].Observation != "cumple" || history[2].Status != "Convertida" {
		t.Fatalf("proposal history=%+v err=%v", history, err)
	}
	link, err := r.ProjectHistory(ctx, project.ID)
	if err != nil || len(link) != 1 || link[0].Field != "propuesta" || link[0].Value != p.ID {
		t.Fatalf("project link=%+v err=%v", link, err)
	}
}

func TestProposalWithoutSeedbedIsStored(t *testing.T) {
	r, _, user, _ := seed(t)
	p, err := r.CreateProposal(context.Background(), domain.ResearchProposal{Title: "T", Problem: "P", Objectives: "O", Methodology: "M", Status: "Radicada", AuthorID: user})
	if err != nil || p.SeedbedID != "" || p.ProjectID != nil {
		t.Fatalf("proposal=%+v err=%v", p, err)
	}
}

func TestActivityWithoutOptionalListsIsStored(t *testing.T) {
	r, _, _, seedbed := seed(t)
	a, err := r.CreateActivity(context.Background(), domain.Activity{SeedbedID: seedbed, Title: "Taller", Status: "próxima"})
	if err != nil || a.ResponsibleIDs == nil || a.Evidence == nil {
		t.Fatalf("activity=%+v err=%v", a, err)
	}
}

func TestReviewNotifiesActiveAdmins(t *testing.T) {
	r, db, user, seedbed := seed(t)
	ctx := context.Background()
	var admin string
	if err := db.QueryRow(ctx, `INSERT INTO usuarios (firebase_uid,nombre,email,rol) VALUES ($1,'Admin',$1||'@example.test','admin') RETURNING id`, fmt.Sprintf("research-admin-%d", time.Now().UnixNano())).Scan(&admin); err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(ctx, `DELETE FROM notificaciones WHERE usuario_id=$1`, admin)
		_, _ = db.Exec(ctx, `DELETE FROM usuarios WHERE id=$1`, admin)
	})
	v, err := r.CreateReview(ctx, domain.SeedbedReview{SeedbedID: seedbed, AuthorID: user, Text: "Buen semillero"})
	if err != nil || v.Status != "pendiente" {
		t.Fatalf("review=%+v err=%v", v, err)
	}
	var n int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM notificaciones WHERE usuario_id=$1 AND tipo='seedbed-review' AND mensaje LIKE '%'||$2||'%'`, admin, v.ID).Scan(&n); err != nil || n != 1 {
		t.Fatalf("admin notifications=%d err=%v", n, err)
	}
	pending, err := r.ListReviews(ctx, "publicada")
	if err != nil || pending == nil {
		t.Fatalf("empty list must be [] not nil: %v err=%v", pending, err)
	}
	if public, err := r.PublishedReviews(ctx, seedbed); err != nil || public == nil || len(public) != 0 {
		t.Fatalf("pending review must not be public: %+v err=%v", public, err)
	}
	if _, err := r.ModerateReview(ctx, v.ID, "publicada", ""); err != nil {
		t.Fatalf("moderate: %v", err)
	}
	if public, err := r.PublishedReviews(ctx, seedbed); err != nil || len(public) != 1 || public[0].Text != "Buen semillero" || public[0].AuthorName == "" {
		t.Fatalf("published review=%+v err=%v", public, err)
	}
}

func TestMalformedOrMissingReferencesAreNotFound(t *testing.T) {
	r, _, user, _ := seed(t)
	ctx := context.Background()
	if _, err := r.FindProject(ctx, "not-a-uuid"); !errors.Is(err, domain.ErrResearchNotFound) {
		t.Fatalf("malformed project id: %v", err)
	}
	if _, err := r.ProjectHistory(ctx, "not-a-uuid"); !errors.Is(err, domain.ErrResearchNotFound) {
		t.Fatalf("malformed history id: %v", err)
	}
	missing := "00000000-0000-0000-0000-000000000000"
	if _, err := r.CreateProject(ctx, domain.Project{SeedbedID: missing, Title: "P", Status: "Radicado", CreatedBy: user}); !errors.Is(err, domain.ErrSeedbedNotFound) {
		t.Fatalf("missing seedbed project: %v", err)
	}
	if _, err := r.CreateReview(ctx, domain.SeedbedReview{SeedbedID: missing, AuthorID: user, Text: "x"}); !errors.Is(err, domain.ErrSeedbedNotFound) {
		t.Fatalf("missing seedbed review: %v", err)
	}
	activities, err := r.ListActivities(ctx, missing, "", "")
	if err != nil || activities == nil {
		t.Fatalf("empty activities must be [] not nil: %v err=%v", activities, err)
	}
}

func TestProjectReturnsNamesOfResponsiblesAndHiddenMembers(t *testing.T) {
	r, db, user, seedbed := seed(t)
	ctx := context.Background()
	var hidden string
	uid := fmt.Sprintf("hidden-member-%d", time.Now().UnixNano())
	if err := db.QueryRow(ctx, `INSERT INTO usuarios (firebase_uid,nombre,apellido,email,rol) VALUES ($1,'Ana','Oculta Ejemplo',$1||'@example.test','estudiante') RETURNING id`, uid).Scan(&hidden); err != nil {
		t.Fatalf("seed hidden member: %v", err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO perfil_investigador (usuario_id, visible_publicamente) VALUES ($1, false)`, hidden); err != nil {
		t.Fatalf("hide member: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(ctx, `DELETE FROM perfil_investigador WHERE usuario_id=$1`, hidden)
		_, _ = db.Exec(ctx, `DELETE FROM usuarios WHERE id=$1`, hidden)
	})

	created, err := r.CreateProject(ctx, domain.Project{SeedbedID: seedbed, Title: "P", Status: "Radicado", CreatedBy: user,
		ResponsibleIDs: []string{user}, MemberIDs: []string{hidden, "00000000-0000-4000-8000-000000000000"}})
	if err != nil {
		t.Fatal(err)
	}
	want := func(p domain.Project) {
		t.Helper()
		if len(p.Responsibles) != 1 || p.Responsibles[0] != (domain.ProjectPerson{ID: user, Name: "Prueba"}) {
			t.Fatalf("responsibles=%+v", p.Responsibles)
		}
		// The unknown id is skipped; the hidden profile still shows its name.
		if len(p.Members) != 1 || p.Members[0] != (domain.ProjectPerson{ID: hidden, Name: "Ana Oculta Ejemplo"}) {
			t.Fatalf("members=%+v", p.Members)
		}
	}
	want(created)
	found, err := r.FindProject(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	want(found)
	list, err := r.ListProjects(ctx, seedbed, "", "")
	if err != nil || len(list) != 1 {
		t.Fatalf("list=%+v err=%v", list, err)
	}
	want(list[0])

	empty, err := r.CreateProject(ctx, domain.Project{SeedbedID: seedbed, Title: "Sin equipo", Status: "Radicado", CreatedBy: user})
	if err != nil || empty.Responsibles == nil || len(empty.Responsibles) != 0 || empty.Members == nil {
		t.Fatalf("project without people must return empty lists, got %+v err=%v", empty, err)
	}
}
