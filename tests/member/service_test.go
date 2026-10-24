package member_test

import (
	"context"
	"errors"
	"testing"

	memberapp "github.com/galash-uptc/backend/internal/application/member"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
)

// repository is an in-memory MemberRepository that records the last call.
type repository struct {
	profiles     map[string]domain.MemberProfile
	publications map[string][]domain.Publication
	filter       ports.MemberFilter
	created      *ports.CreateMemberCommand
	updated      *ports.UpdateMemberCommand
	status       string
}

func newRepository() *repository {
	return &repository{
		profiles: map[string]domain.MemberProfile{
			"ana":    {ID: "ana", FirebaseUID: "uid-ana", Name: "Ana", LastName: "Pérez", Email: "ana@example.com", DocumentType: "CC", DocumentNumber: "123", Role: "estudiante", Status: "activo", VisiblePublicly: true},
			"hidden": {ID: "hidden", Name: "Oculto", Email: "hidden@example.com", Role: "profesor", Status: "activo", VisiblePublicly: false},
			"off":    {ID: "off", Name: "Inactivo", Email: "off@example.com", Role: "estudiante", Status: "inactivo", VisiblePublicly: true},
		},
		publications: map[string][]domain.Publication{"ana": {{ID: "p1", Title: "Artículo"}}},
	}
}

func (r *repository) List(_ context.Context, f ports.MemberFilter) ([]domain.MemberProfile, error) {
	r.filter = f
	return []domain.MemberProfile{}, nil
}
func (r *repository) Find(_ context.Context, id string) (domain.MemberProfile, error) {
	p, ok := r.profiles[id]
	if !ok {
		return domain.MemberProfile{}, domain.ErrMemberNotFound
	}
	return p, nil
}
func (r *repository) CreateMember(_ context.Context, c ports.CreateMemberCommand) (domain.MemberProfile, error) {
	r.created = &c
	return domain.MemberProfile{ID: "new", Name: c.Name, Role: c.Role, Status: "activo"}, nil
}
func (r *repository) Update(_ context.Context, id string, c ports.UpdateMemberCommand) (domain.MemberProfile, error) {
	r.updated = &c
	return r.Find(context.Background(), id)
}
func (r *repository) SetStatus(_ context.Context, id, status string) (domain.MemberProfile, error) {
	p, err := r.Find(context.Background(), id)
	if err != nil {
		return p, err
	}
	r.status = status
	p.Status = status
	r.profiles[id] = p
	return p, nil
}
func (r *repository) Publications(_ context.Context, id string) ([]domain.Publication, error) {
	return r.publications[id], nil
}

func validCreate() memberapp.CreateCommand {
	return memberapp.CreateCommand{FirebaseUID: "uid", Name: "Ana", LastName: "Pérez", Email: "ana@example.com", Role: "estudiante"}
}

func TestCreateValidatesRequiredFieldsRoleAndCvLAC(t *testing.T) {
	invalid := map[string]func(*memberapp.CreateCommand){
		"missing uid":       func(c *memberapp.CreateCommand) { c.FirebaseUID = " " },
		"missing name":      func(c *memberapp.CreateCommand) { c.Name = "" },
		"missing last name": func(c *memberapp.CreateCommand) { c.LastName = "" },
		"missing email":     func(c *memberapp.CreateCommand) { c.Email = "" },
		"admin role":        func(c *memberapp.CreateCommand) { c.Role = "admin" },
		"cvlac not url":     func(c *memberapp.CreateCommand) { c.CvLAC = "cvlac" },
		"cvlac ftp":         func(c *memberapp.CreateCommand) { c.CvLAC = "ftp://scienti.minciencias.gov.co/cvlac" },
	}
	for name, mutate := range invalid {
		repo := newRepository()
		c := validCreate()
		mutate(&c)
		if _, err := memberapp.NewService(repo).Create(context.Background(), c); !errors.Is(err, domain.ErrInvalidMember) {
			t.Errorf("%s: err=%v, want ErrInvalidMember", name, err)
		}
		if repo.created != nil {
			t.Errorf("%s: invalid member reached repository", name)
		}
	}
	for _, role := range []string{"estudiante", "profesor", "Docente"} {
		repo := newRepository()
		c := validCreate()
		c.Role, c.CvLAC = role, "https://scienti.minciencias.gov.co/cvlac/visualizador"
		if _, err := memberapp.NewService(repo).Create(context.Background(), c); err != nil || repo.created == nil {
			t.Errorf("role %s: err=%v created=%v", role, err, repo.created != nil)
		}
	}
}

func TestUpdateValidatesRoleAndCvLAC(t *testing.T) {
	cases := map[string]struct {
		id string
		c  memberapp.UpdateCommand
	}{
		"empty id":    {"", memberapp.UpdateCommand{}},
		"admin role":  {"ana", memberapp.UpdateCommand{Role: "admin"}},
		"cvlac no ok": {"ana", memberapp.UpdateCommand{CvLAC: "javascript:alert(1)"}},
	}
	for name, tc := range cases {
		repo := newRepository()
		if _, err := memberapp.NewService(repo).Update(context.Background(), tc.id, tc.c); !errors.Is(err, domain.ErrInvalidMember) {
			t.Errorf("%s: err=%v, want ErrInvalidMember", name, err)
		}
		if repo.updated != nil {
			t.Errorf("%s: invalid update reached repository", name)
		}
	}
	repo := newRepository()
	if _, err := memberapp.NewService(repo).Update(context.Background(), "ana", memberapp.UpdateCommand{Area: "IA"}); err != nil || repo.updated == nil {
		t.Fatalf("partial update: err=%v", err)
	}
}

func TestSetStatusAcceptsOnlyActivoOrInactivo(t *testing.T) {
	repo := newRepository()
	service := memberapp.NewService(repo)
	p, err := service.SetStatus(context.Background(), "ana", " Inactivo ")
	if err != nil || p.Status != "inactivo" || repo.status != "inactivo" {
		t.Fatalf("SetStatus(Inactivo): status=%q repo=%q err=%v", p.Status, repo.status, err)
	}
	if _, err := service.SetStatus(context.Background(), "ana", "eliminado"); !errors.Is(err, domain.ErrInvalidMemberStatus) {
		t.Fatalf("SetStatus(eliminado): err=%v, want ErrInvalidMemberStatus", err)
	}
}

func TestGetStripsPrivateFieldsForPublicView(t *testing.T) {
	service := memberapp.NewService(newRepository())
	public, err := service.Get(context.Background(), "ana", false)
	if err != nil {
		t.Fatal(err)
	}
	if public.Email != "" || public.DocumentType != "" || public.DocumentNumber != "" || public.FirebaseUID != "" {
		t.Errorf("public profile exposes private fields: %+v", public)
	}
	private, err := service.Get(context.Background(), "ana", true)
	if err != nil || private.Email != "ana@example.com" || private.DocumentNumber != "123" {
		t.Errorf("private profile lost fields: %+v err=%v", private, err)
	}
	if _, err := service.Get(context.Background(), " ", false); !errors.Is(err, domain.ErrMemberNotFound) {
		t.Errorf("blank id: err=%v, want ErrMemberNotFound", err)
	}
}

func TestListClampsLimit(t *testing.T) {
	for requested, want := range map[int]int{0: 20, -5: 20, 50: 50, 500: 100} {
		repo := newRepository()
		if _, err := memberapp.NewService(repo).List(context.Background(), memberapp.Filter{Limit: requested}); err != nil {
			t.Fatal(err)
		}
		if repo.filter.Limit != want {
			t.Errorf("limit %d: repository got %d, want %d", requested, repo.filter.Limit, want)
		}
	}
}
