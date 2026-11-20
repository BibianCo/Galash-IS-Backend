package research_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/galash-uptc/backend/internal/domain"
)

// TDB-T03: a project cannot end before it starts.
func TestProjectRejectsEndBeforeStart(t *testing.T) {
	s, _ := fixtures()
	start, end := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	p := domain.Project{SeedbedID: "sb", Title: "P", Status: "Activo", StartDate: &start, EndDate: &end}
	if _, err := s.CreateProject(context.Background(), p, "teacher"); !errors.Is(err, domain.ErrInvalidResearch) {
		t.Fatalf("create end before start: err=%v, want ErrInvalidResearch", err)
	}
	if _, err := s.UpdateProject(context.Background(), "proj", p, "teacher"); !errors.Is(err, domain.ErrInvalidResearch) {
		t.Fatalf("update end before start: err=%v, want ErrInvalidResearch", err)
	}
}

// TDB-T11: two editors load the same project; the second save must not silently revert the first one.
func TestStaleProjectSaveDoesNotSilentlyRevertOtherEdit(t *testing.T) {
	repo, db, _, seedbed := seed(t)
	ctx := context.Background()
	var id string
	if err := db.QueryRow(ctx, `INSERT INTO proyectos (titulo,resumen,semillero_id) VALUES ('Original','Resumen',$1) RETURNING id`, seedbed).Scan(&id); err != nil {
		t.Fatal(err)
	}
	loaded, err := repo.FindProject(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	first, second := loaded, loaded
	first.Title = "Título de A"
	second.Summary = "Resumen de B"
	if _, err := repo.UpdateProject(ctx, id, first, "", nil); err != nil {
		t.Fatal(err)
	}
	_, staleErr := repo.UpdateProject(ctx, id, second, "", nil)
	final, _ := repo.FindProject(ctx, id)
	if staleErr == nil || final.Title != first.Title || final.Summary == second.Summary {
		t.Fatalf("stale save err=%v, final=%+v: want rejection keeping A's title", staleErr, final)
	}
}
