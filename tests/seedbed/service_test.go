package seedbed_test

import (
	"context"
	"errors"
	"testing"

	application "github.com/galash-uptc/backend/internal/application/seedbed"
	"github.com/galash-uptc/backend/internal/domain"
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
func (u users) SetPassword(context.Context, string, string) error           { return nil }
func (u users) Search(context.Context, string, int) ([]domain.User, error) { return nil, nil }
func (u users) UpdateRole(context.Context, string, string) error           { return nil }

type repository struct {
	seedbeds         map[string]domain.Seedbed
	space            domain.SeedbedSpace
	created          domain.Seedbed
	page, size       int
	private, history bool
	visibility       string
	added, removed   string
}

func (r *repository) Create(_ context.Context, s domain.Seedbed) (domain.Seedbed, error) {
	s.ID = "new-seedbed"
	r.created = s
	return s, nil
}
func (r *repository) List(_ context.Context, _, _ string, page, size int, private bool) ([]domain.Seedbed, error) {
	r.page, r.size, r.private = page, size, private
	return []domain.Seedbed{}, nil
}
func (r *repository) Find(_ context.Context, id string) (domain.Seedbed, error) {
	s, ok := r.seedbeds[id]
	if !ok {
		return domain.Seedbed{}, domain.ErrSeedbedNotFound
	}
	return s, nil
}
func (r *repository) Space(context.Context, string) (domain.SeedbedSpace, error) { return r.space, nil }
func (r *repository) Update(_ context.Context, id string, s domain.Seedbed) (domain.Seedbed, error) {
	s.ID = id
	return s, nil
}
func (r *repository) SetVisibility(_ context.Context, id, status string) (domain.Seedbed, error) {
	r.visibility = status
	s := r.seedbeds[id]
	s.Status, s.Visibility = status, status
	return s, nil
}
func (r *repository) ListMembers(_ context.Context, _ string, history bool) ([]domain.SeedbedMember, error) {
	r.history = history
	return []domain.SeedbedMember{}, nil
}
func (r *repository) AddMember(_ context.Context, _, userID string) error {
	r.added = userID
	return nil
}
func (r *repository) RemoveMember(_ context.Context, _, userID string) error {
	r.removed = userID
	return nil
}

func fixtures() (users, *repository) {
	u := users{
		"admin":    {ID: "admin", Role: "admin", Status: "activo"},
		"teacher":  {ID: "teacher", Role: "profesor", Status: "activo"},
		"other":    {ID: "other", Role: "docente", Status: "activo"},
		"student":  {ID: "student", Role: "estudiante", Status: "activo"},
		"visitor":  {ID: "visitor", Role: "visitante", Status: "activo"},
		"inactive": {ID: "inactive", Role: "profesor", Status: "inactivo"},
	}
	r := &repository{seedbeds: map[string]domain.Seedbed{
		"public": {ID: "public", ResponsibleTeacherID: "teacher", Status: domain.SeedbedPublished, Visibility: domain.SeedbedPublished},
		"hidden": {ID: "hidden", ResponsibleTeacherID: "teacher", Status: domain.SeedbedHidden, Visibility: domain.SeedbedHidden},
	}}
	return u, r
}

func validCommand() application.CreateCommand {
	return application.CreateCommand{Name: " Robótica ", Description: "d", ResearchLine: "IA", Objective: "o", ResponsibleTeacherID: "teacher", ContactEmail: "semillero@example.com"}
}

func TestCreateRejectsMissingRequiredFields(t *testing.T) {
	u, r := fixtures()
	service := application.NewService(r, u)
	for name, mutate := range map[string]func(*application.CreateCommand){
		"name":        func(c *application.CreateCommand) { c.Name = " " },
		"responsible": func(c *application.CreateCommand) { c.ResponsibleTeacherID = "" },
		"email":       func(c *application.CreateCommand) { c.ContactEmail = "sin-arroba" },
	} {
		c := validCommand()
		mutate(&c)
		if _, err := service.Create(context.Background(), c); !errors.Is(err, domain.ErrInvalidSeedbed) {
			t.Errorf("%s: err=%v, want ErrInvalidSeedbed", name, err)
		}
	}
}

func TestCreateRequiresActiveTeacherAsResponsible(t *testing.T) {
	u, r := fixtures()
	service := application.NewService(r, u)
	for _, id := range []string{"student", "admin", "inactive", "missing"} {
		c := validCommand()
		c.ResponsibleTeacherID = id
		if _, err := service.Create(context.Background(), c); !errors.Is(err, domain.ErrInvalidTeacher) {
			t.Errorf("%s: err=%v, want ErrInvalidTeacher", id, err)
		}
	}
}

func TestCreateStartsAsDraftWithTrimmedData(t *testing.T) {
	u, r := fixtures()
	s, err := application.NewService(r, u).Create(context.Background(), validCommand())
	if err != nil {
		t.Fatalf("Create(): %v", err)
	}
	if s.ID == "" || s.Name != "Robótica" || s.ResponsibleTeacherID != "teacher" ||
		s.Status != domain.SeedbedDraft || s.Visibility != domain.SeedbedDraft {
		t.Fatalf("unexpected seedbed: %+v", s)
	}
}

func TestVisibilityAllowsResponsibleOrAdminAndKnownStates(t *testing.T) {
	u, r := fixtures()
	service := application.NewService(r, u)
	ctx := context.Background()
	for _, tc := range []struct{ actor, status string }{
		{"teacher", domain.SeedbedPublished}, {"teacher", domain.SeedbedHidden}, {"admin", domain.SeedbedInactive},
	} {
		s, err := service.Visibility(ctx, tc.actor, "public", tc.status)
		if err != nil || s.Status != tc.status {
			t.Errorf("%s->%s: seedbed=%+v err=%v", tc.actor, tc.status, s, err)
		}
	}
	if _, err := service.Visibility(ctx, "other", "public", domain.SeedbedHidden); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("non-responsible: err=%v, want ErrForbidden", err)
	}
	if _, err := service.Visibility(ctx, "teacher", "public", "Eliminado"); !errors.Is(err, domain.ErrInvalidSeedbedStatus) {
		t.Errorf("unknown status: err=%v, want ErrInvalidSeedbedStatus", err)
	}
	if _, err := service.Visibility(ctx, "admin", "missing", domain.SeedbedHidden); !errors.Is(err, domain.ErrSeedbedNotFound) {
		t.Errorf("missing seedbed: err=%v, want ErrSeedbedNotFound", err)
	}
}

func TestAddMemberAcceptsOnlyActiveStudentsOrTeachers(t *testing.T) {
	u, r := fixtures()
	service := application.NewService(r, u)
	ctx := context.Background()
	for _, id := range []string{"student", "other"} {
		if err := service.AddMember(ctx, "teacher", "public", id); err != nil || r.added != id {
			t.Errorf("%s: added=%q err=%v", id, r.added, err)
		}
	}
	for _, id := range []string{"visitor", "admin", "inactive", "missing"} {
		if err := service.AddMember(ctx, "teacher", "public", id); err == nil {
			t.Errorf("%s: linked, want rejection", id)
		}
	}
	if err := service.AddMember(ctx, "other", "public", "student"); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("non-responsible: err=%v, want ErrForbidden", err)
	}
}

func TestRemoveMemberRequiresResponsibleOrAdmin(t *testing.T) {
	u, r := fixtures()
	service := application.NewService(r, u)
	ctx := context.Background()
	if err := service.RemoveMember(ctx, "student", "public", "student"); !errors.Is(err, domain.ErrForbidden) || r.removed != "" {
		t.Fatalf("student self-removal: removed=%q err=%v", r.removed, err)
	}
	if err := service.RemoveMember(ctx, "admin", "public", "student"); err != nil || r.removed != "student" {
		t.Fatalf("admin removal: removed=%q err=%v", r.removed, err)
	}
}

func TestListBoundsPagination(t *testing.T) {
	u, r := fixtures()
	service := application.NewService(r, u)
	_, _ = service.List(context.Background(), "", "", 0, 500, false)
	if r.page != 1 || r.size != 100 {
		t.Fatalf("page=%d size=%d, want 1 and 100", r.page, r.size)
	}
	_, _ = service.List(context.Background(), "", "", 2, 0, false)
	if r.page != 2 || r.size != 20 {
		t.Fatalf("page=%d size=%d, want 2 and 20", r.page, r.size)
	}
}
