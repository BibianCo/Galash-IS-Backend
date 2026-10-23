package research_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	application "github.com/galash-uptc/backend/internal/application/research"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
)

type users map[string]domain.User

func (u users) FindByID(_ context.Context, id string) (domain.User, error) {
	user, ok := u[id]
	if !ok {
		return domain.User{}, errors.New("user not found")
	}
	return user, nil
}
func (u users) FindByEmail(context.Context, string) (domain.User, error) {
	return domain.User{}, errors.New("not used")
}
func (u users) FindByFirebaseUID(ctx context.Context, uid string) (domain.User, error) {
	return u.FindByID(ctx, uid)
}
func (u users) SetPassword(context.Context, string, string) error          { return nil }
func (u users) Search(context.Context, string, int) ([]domain.User, error) { return nil, nil }
func (u users) UpdateRole(context.Context, string, string) error           { return nil }

// seedbeds implements only Find; other methods are not used by the research service.
type seedbeds struct{ ports.SeedbedRepository }

func (seedbeds) Find(_ context.Context, id string) (domain.Seedbed, error) {
	if id != "sb" {
		return domain.Seedbed{}, domain.ErrSeedbedNotFound
	}
	return domain.Seedbed{ID: "sb", ResponsibleTeacherID: "teacher"}, nil
}

// repository embeds the port so unused methods panic if reached.
type repository struct {
	ports.ResearchRepository
	project   domain.Project
	proposal  domain.ResearchProposal
	changes   []domain.ProjectHistory
	actor     string
	saved     domain.ResearchProposal
	converted bool
}

func (r *repository) CreateActivity(_ context.Context, a domain.Activity) (domain.Activity, error) {
	a.ID = "act"
	return a, nil
}
func (r *repository) FindActivity(context.Context, string) (domain.Activity, error) {
	return domain.Activity{ID: "act", SeedbedID: "sb", Title: "Taller", Status: "próxima"}, nil
}
func (r *repository) UpdateActivity(_ context.Context, id string, a domain.Activity) (domain.Activity, error) {
	a.ID = id
	return a, nil
}
func (r *repository) CreateProject(_ context.Context, p domain.Project) (domain.Project, error) {
	p.ID = "proj"
	return p, nil
}
func (r *repository) FindProject(context.Context, string) (domain.Project, error) {
	return r.project, nil
}
func (r *repository) UpdateProject(_ context.Context, id string, p domain.Project, actor string, changes []domain.ProjectHistory) (domain.Project, error) {
	r.actor, r.changes = actor, changes
	p.ID = id
	return p, nil
}
func (r *repository) UpdateProgress(_ context.Context, _ string, n int, actor string) (domain.Project, error) {
	r.actor = actor
	p := r.project
	p.Progress = n
	return p, nil
}
func (r *repository) CreateProposal(_ context.Context, p domain.ResearchProposal) (domain.ResearchProposal, error) {
	r.saved = p
	return p, nil
}
func (r *repository) FindProposal(context.Context, string) (domain.ResearchProposal, error) {
	return r.proposal, nil
}
func (r *repository) UpdateProposal(_ context.Context, _ string, p domain.ResearchProposal, actor string) (domain.ResearchProposal, error) {
	r.saved, r.actor = p, actor
	return p, nil
}
func (r *repository) ListProposals(_ context.Context, actor string) ([]domain.ResearchProposal, error) {
	r.actor = actor
	return []domain.ResearchProposal{}, nil
}
func (r *repository) CreateReview(_ context.Context, v domain.SeedbedReview) (domain.SeedbedReview, error) {
	v.ID, v.Status = "rev", "pendiente"
	return v, nil
}
func (r *repository) ListReviews(_ context.Context, status string) ([]domain.SeedbedReview, error) {
	r.actor = status
	return []domain.SeedbedReview{}, nil
}
func (r *repository) ProposalHistory(context.Context, string) ([]domain.ProposalHistory, error) {
	return []domain.ProposalHistory{{Status: "Radicada"}}, nil
}
func (r *repository) ConvertProposal(_ context.Context, _ string, actor string) (domain.Project, error) {
	r.converted, r.actor = true, actor
	return domain.Project{ID: "proj", Status: "Radicado"}, nil
}

func fixtures() (*application.Service, *repository) {
	s, r, _ := fixturesWithUsers()
	return s, r
}

func fixturesWithUsers() (*application.Service, *repository, users) {
	u := users{
		"admin":   {ID: "admin", Role: "admin"},
		"teacher": {ID: "teacher", Role: "profesor"},
		"other":   {ID: "other", Role: "docente"},
		"student": {ID: "student", Role: "estudiante"},
	}
	r := &repository{
		project:  domain.Project{ID: "proj", SeedbedID: "sb", Title: "P", Status: "Radicado", Progress: 10},
		proposal: domain.ResearchProposal{ID: "prop", SeedbedID: "sb", Title: "T", Problem: "P", Objectives: "O", Methodology: "M", AuthorID: "student", Status: "Radicada", Observations: "revisar"},
	}
	return application.NewService(r, seedbeds{}, u), r, u
}

func TestActivityRequiresResponsibleTeacherOrAdmin(t *testing.T) {
	s, _ := fixtures()
	ctx := context.Background()
	a := domain.Activity{SeedbedID: "sb", Title: "Taller", Status: "próxima"}
	for _, actor := range []string{"teacher", "admin"} {
		if _, err := s.CreateActivity(ctx, a, actor); err != nil {
			t.Fatalf("%s create: %v", actor, err)
		}
		if _, err := s.UpdateActivity(ctx, "act", a, actor); err != nil {
			t.Fatalf("%s update: %v", actor, err)
		}
	}
	for _, actor := range []string{"other", "student", "missing"} {
		if _, err := s.CreateActivity(ctx, a, actor); !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("%s create: want forbidden, got %v", actor, err)
		}
		if _, err := s.UpdateActivity(ctx, "act", a, actor); !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("%s update: want forbidden, got %v", actor, err)
		}
	}
	for _, bad := range []domain.Activity{{SeedbedID: "sb", Title: " ", Status: "próxima"}, {SeedbedID: "sb", Title: "T", Status: "pendiente"}} {
		if _, err := s.CreateActivity(ctx, bad, "teacher"); !errors.Is(err, domain.ErrInvalidResearch) {
			t.Fatalf("want invalid for %+v, got %v", bad, err)
		}
	}
}

func TestProjectProgressMustBeBetween0And100(t *testing.T) {
	s, _ := fixtures()
	ctx := context.Background()
	for _, n := range []int{-1, 101} {
		if _, err := s.CreateProject(ctx, domain.Project{SeedbedID: "sb", Title: "P", Status: "Activo", Progress: n}, "teacher"); !errors.Is(err, domain.ErrInvalidResearch) {
			t.Fatalf("create %d: want invalid, got %v", n, err)
		}
		if _, err := s.Progress(ctx, "proj", n, "teacher"); !errors.Is(err, domain.ErrInvalidResearch) {
			t.Fatalf("progress %d: want invalid, got %v", n, err)
		}
	}
	for _, n := range []int{0, 100} {
		if _, err := s.CreateProject(ctx, domain.Project{SeedbedID: "sb", Title: "P", Status: "Activo", Progress: n}, "teacher"); err != nil {
			t.Fatalf("create %d: %v", n, err)
		}
	}
	if _, err := s.CreateProject(ctx, domain.Project{SeedbedID: "sb", Title: "P", Status: "Pausado"}, "teacher"); !errors.Is(err, domain.ErrInvalidResearch) {
		t.Fatalf("unknown status: want invalid, got %v", err)
	}
}

func TestProjectUpdateRecordsActorAndChangedFields(t *testing.T) {
	s, r := fixtures()
	p := r.project
	p.Status, p.Progress = "Activo", 40
	if _, err := s.UpdateProject(context.Background(), "proj", p, "teacher"); err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, h := range r.changes {
		got[h.Field] = h.Value
	}
	if r.actor != "teacher" || len(got) != 2 || got["estado"] != "Activo" || got["avance"] != "40" {
		t.Fatalf("history actor=%q changes=%v", r.actor, got)
	}
}

func TestProgressOnlyByResponsibleOrAdmin(t *testing.T) {
	s, r := fixtures()
	ctx := context.Background()
	for _, actor := range []string{"other", "student"} {
		if _, err := s.Progress(ctx, "proj", 50, actor); !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("%s: want forbidden, got %v", actor, err)
		}
	}
	p, err := s.Progress(ctx, "proj", 50, "teacher")
	if err != nil || p.Progress != 50 || r.actor != "teacher" {
		t.Fatalf("responsible: progress=%d actor=%q err=%v", p.Progress, r.actor, err)
	}
}

func TestProposalCreateAlwaysStartsRadicada(t *testing.T) {
	s, r := fixtures()
	in := domain.ResearchProposal{Title: "T", Problem: "P", Objectives: "O", Methodology: "M", Status: "Aprobada"}
	if _, err := s.CreateProposal(context.Background(), in, "student"); err != nil {
		t.Fatal(err)
	}
	if r.saved.Status != "Radicada" || r.saved.AuthorID != "student" {
		t.Fatalf("saved status=%q author=%q", r.saved.Status, r.saved.AuthorID)
	}
	in.Methodology = " "
	if _, err := s.CreateProposal(context.Background(), in, "student"); !errors.Is(err, domain.ErrInvalidResearch) {
		t.Fatalf("missing methodology: want invalid, got %v", err)
	}
}

func TestProposalAuthorCannotApproveOrReview(t *testing.T) {
	s, r := fixtures()
	ctx := context.Background()
	edit := domain.ResearchProposal{Title: "Nuevo", Problem: "P", Objectives: "O", Methodology: "M", Status: "Aprobada", Observations: "ok"}
	if _, err := s.UpdateProposal(ctx, "prop", edit, "student"); err != nil {
		t.Fatal(err)
	}
	if r.saved.Status != "Radicada" || r.saved.Observations != "revisar" || r.saved.Title != "Nuevo" {
		t.Fatalf("author update saved %+v", r.saved)
	}
	if _, err := s.UpdateProposal(ctx, "prop", edit, "teacher"); err != nil {
		t.Fatal(err)
	}
	if r.saved.Status != "Aprobada" || r.saved.Observations != "ok" || r.actor != "teacher" {
		t.Fatalf("reviewer update saved %+v actor=%q", r.saved, r.actor)
	}
	edit.Status = "Convertida"
	if _, err := s.UpdateProposal(ctx, "prop", edit, "admin"); !errors.Is(err, domain.ErrInvalidResearch) {
		t.Fatalf("manual Convertida: want invalid, got %v", err)
	}
}

func TestProposalDetailIncludesHistory(t *testing.T) {
	s, _ := fixtures()
	p, err := s.Proposal(context.Background(), "prop", "student")
	if err != nil || len(p.History) != 1 {
		t.Fatalf("history=%v err=%v", p.History, err)
	}
	if _, err := s.Proposal(context.Background(), "prop", "missing"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("stranger: want forbidden, got %v", err)
	}
}

func TestConvertRequiresApprovedProposalAndReviewer(t *testing.T) {
	s, r := fixtures()
	ctx := context.Background()
	if _, err := s.Convert(ctx, "prop", "teacher"); !errors.Is(err, domain.ErrInvalidResearch) {
		t.Fatalf("not approved: want invalid, got %v", err)
	}
	r.proposal.Status = "Aprobada"
	if _, err := s.Convert(ctx, "prop", "student"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("student: want forbidden, got %v", err)
	}
	p, err := s.Convert(ctx, "prop", "teacher")
	if err != nil || !r.converted || p.Status != "Radicado" || r.actor != "teacher" {
		t.Fatalf("convert project=%+v converted=%v err=%v", p, r.converted, err)
	}
}

func TestResearchJSONUsesCamelCase(t *testing.T) {
	for _, v := range []any{domain.Activity{}, domain.Project{}, domain.ProjectHistory{}, domain.ResearchProposal{}, domain.ProposalHistory{}, domain.SeedbedReview{}} {
		raw, _ := json.Marshal(v)
		if !strings.Contains(string(raw), `"id":`) || strings.Contains(string(raw), `"ID"`) || strings.Contains(string(raw), `"SeedbedID"`) {
			t.Fatalf("%T JSON not camelCase: %s", v, raw)
		}
	}
}

func TestProposalListScopesByRole(t *testing.T) {
	s, r := fixtures()
	ctx := context.Background()
	for actor, scope := range map[string]string{"admin": "", "teacher": "", "student": "student"} {
		r.actor = "unset"
		if _, err := s.Proposals(ctx, actor); err != nil || r.actor != scope {
			t.Fatalf("%s: scope=%q err=%v", actor, r.actor, err)
		}
	}
	if _, err := s.Proposals(ctx, "missing"); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("unknown user: want forbidden, got %v", err)
	}
}

func TestReviewRequiresExistingSeedbedAndQueueDefaultsToPending(t *testing.T) {
	s, r := fixtures()
	ctx := context.Background()
	if _, err := s.Review(ctx, domain.SeedbedReview{SeedbedID: "nope", AuthorID: "student", Text: "x"}); !errors.Is(err, domain.ErrSeedbedNotFound) {
		t.Fatalf("unknown seedbed: want not found, got %v", err)
	}
	if v, err := s.Review(ctx, domain.SeedbedReview{SeedbedID: "sb", AuthorID: "student", Text: "x"}); err != nil || v.Status != "pendiente" {
		t.Fatalf("review=%+v err=%v", v, err)
	}
	if _, err := s.Reviews(ctx, ""); err != nil || r.actor != "pendiente" {
		t.Fatalf("default status=%q err=%v", r.actor, err)
	}
	if _, err := s.Reviews(ctx, "publicada"); err != nil || r.actor != "publicada" {
		t.Fatalf("explicit status=%q err=%v", r.actor, err)
	}
}
