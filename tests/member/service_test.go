package member_test

import (
	"context"
	"errors"
	"testing"

	memberapp "github.com/galash-uptc/backend/internal/application/member"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
)

// Fixture IDs are UUIDs because the service rejects malformed member IDs.
const (
	anaID     = "00000000-0000-4000-8000-000000000001"
	hiddenID  = "00000000-0000-4000-8000-000000000002"
	offID     = "00000000-0000-4000-8000-000000000003"
	adminID   = "00000000-0000-4000-8000-000000000004"
	otherID   = "00000000-0000-4000-8000-000000000005"
	missingID = "00000000-0000-4000-8000-000000000006"
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
			anaID:    {ID: anaID, FirebaseUID: "uid-ana", Name: "Ana", LastName: "Pérez", Email: "ana@example.com", DocumentType: "CC", DocumentNumber: "123", Role: "estudiante", Status: "activo", VisiblePublicly: true},
			hiddenID: {ID: hiddenID, Name: "Oculto", Email: "hidden@example.com", Role: "profesor", Status: "activo", VisiblePublicly: false},
			offID:    {ID: offID, Name: "Inactivo", Email: "off@example.com", Role: "estudiante", Status: "inactivo", VisiblePublicly: true},
		},
		publications: map[string][]domain.Publication{anaID: {{ID: "p1", Title: "Artículo"}}},
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
		"admin role":  {anaID, memberapp.UpdateCommand{Role: "admin"}},
		"cvlac no ok": {anaID, memberapp.UpdateCommand{CvLAC: "javascript:alert(1)"}},
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
	if _, err := memberapp.NewService(repo).Update(context.Background(), anaID, memberapp.UpdateCommand{Area: "IA"}); err != nil || repo.updated == nil {
		t.Fatalf("partial update: err=%v", err)
	}
}

func TestSetStatusAcceptsOnlyActivoOrInactivo(t *testing.T) {
	repo := newRepository()
	service := memberapp.NewService(repo)
	p, err := service.SetStatus(context.Background(), anaID, " Inactivo ")
	if err != nil || p.Status != "inactivo" || repo.status != "inactivo" {
		t.Fatalf("SetStatus(Inactivo): status=%q repo=%q err=%v", p.Status, repo.status, err)
	}
	if _, err := service.SetStatus(context.Background(), anaID, "eliminado"); !errors.Is(err, domain.ErrInvalidMemberStatus) {
		t.Fatalf("SetStatus(eliminado): err=%v, want ErrInvalidMemberStatus", err)
	}
}

func TestGetStripsPrivateFieldsForPublicView(t *testing.T) {
	service := memberapp.NewService(newRepository())
	public, err := service.Get(context.Background(), anaID, false)
	if err != nil {
		t.Fatal(err)
	}
	if public.Email != "" || public.DocumentType != "" || public.DocumentNumber != "" || public.FirebaseUID != "" {
		t.Errorf("public profile exposes private fields: %+v", public)
	}
	private, err := service.Get(context.Background(), anaID, true)
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

type fakeRepository struct {
	profile domain.MemberProfile
}

func (f *fakeRepository) List(context.Context, ports.MemberFilter) ([]domain.MemberProfile, error) {
	return []domain.MemberProfile{f.profile}, nil
}
func (f *fakeRepository) Find(context.Context, string) (domain.MemberProfile, error) {
	return f.profile, nil
}
func (f *fakeRepository) CreateMember(context.Context, ports.CreateMemberCommand) (domain.MemberProfile, error) {
	return f.profile, nil
}
func (f *fakeRepository) Update(context.Context, string, ports.UpdateMemberCommand) (domain.MemberProfile, error) {
	return f.profile, nil
}
func (f *fakeRepository) SetStatus(context.Context, string, string) (domain.MemberProfile, error) {
	return f.profile, nil
}
func (f *fakeRepository) Publications(context.Context, string) ([]domain.Publication, error) {
	return nil, nil
}

func TestGetHidesPrivateProfileFromVisitors(t *testing.T) {
	s := memberapp.NewService(&fakeRepository{profile: domain.MemberProfile{
		ID:              "896ac946-b4d3-4527-887d-be37a655a748",
		Status:          "activo",
		VisiblePublicly: false,
	}})

	_, err := s.Get(context.Background(), "896ac946-b4d3-4527-887d-be37a655a748", false)
	if !errors.Is(err, domain.ErrMemberNotFound) {
		t.Fatalf("Get() error = %v, want ErrMemberNotFound", err)
	}
}

func TestMemberOperationsRejectMalformedIDsAsNotFound(t *testing.T) {
	s := memberapp.NewService(&fakeRepository{})
	ctx := context.Background()
	if _, err := s.Get(ctx, "not-a-uuid", false); !errors.Is(err, domain.ErrMemberNotFound) {
		t.Fatalf("Get() error = %v, want ErrMemberNotFound", err)
	}
	if _, err := s.Update(ctx, "not-a-uuid", memberapp.UpdateCommand{}); !errors.Is(err, domain.ErrInvalidMember) && !errors.Is(err, domain.ErrMemberNotFound) {
		t.Fatalf("Update() error = %v, want a controlled not-found validation error", err)
	}
	if _, err := s.SetStatus(ctx, "not-a-uuid", "activo"); !errors.Is(err, domain.ErrMemberNotFound) {
		t.Fatalf("SetStatus() error = %v, want ErrMemberNotFound", err)
	}
	if _, err := s.Publications(ctx, "not-a-uuid"); !errors.Is(err, domain.ErrMemberNotFound) {
		t.Fatalf("Publications() error = %v, want ErrMemberNotFound", err)
	}
}
