package announcements_test

import (
	"context"
	"github.com/galash-uptc/backend/internal/application/announcement"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
	"testing"
)

type users struct{ user domain.User }

func (u users) FindByID(context.Context, string) (domain.User, error)  { return u.user, nil }
func (users) FindByEmail(context.Context, string) (domain.User, error) { return domain.User{}, nil }
func (users) FindByFirebaseUID(context.Context, string) (domain.User, error) {
	return domain.User{}, nil
}
func (users) SetPassword(context.Context, string, string) error                { return nil }
func (users) Search(context.Context, string, int, bool) ([]domain.User, error) { return nil, nil }
func (users) UpdateRole(context.Context, string, string) error                 { return nil }

type repo struct {
	created          domain.Announcement
	includeAllDrafts bool
	findAllDrafts    bool
}

func (r *repo) List(_ context.Context, _ string, includeAllDrafts bool) ([]domain.Announcement, error) {
	r.includeAllDrafts = includeAllDrafts
	return nil, nil
}
func (r *repo) Find(_ context.Context, _ string, _ string, includeAllDrafts bool) (domain.Announcement, error) {
	r.findAllDrafts = includeAllDrafts
	return r.created, nil
}
func (r *repo) Create(_ context.Context, a domain.Announcement) (domain.Announcement, error) {
	r.created = a
	return a, nil
}
func (r *repo) Update(context.Context, string, string, string, []domain.AnnouncementAttachment) (domain.Announcement, error) {
	return domain.Announcement{}, nil
}
func (r *repo) Publish(context.Context, string) (domain.Announcement, error) {
	return domain.Announcement{}, nil
}
func (r *repo) ConfirmRead(context.Context, string, string) error { return nil }
func (r *repo) ReadStatus(context.Context, string) ([]domain.AnnouncementReadStatus, error) {
	return nil, nil
}

var _ ports.UserReader = users{}
var _ ports.AnnouncementRepository = (*repo)(nil)

func TestCreateRejectsUnsupportedAttachment(t *testing.T) {
	s := announcement.NewService(&repo{}, users{domain.User{ID: "u1", Role: "Docente", Status: "activo"}})
	_, err := s.Create(context.Background(), domain.Announcement{Title: "Directriz", Description: "Texto", Attachments: []domain.AnnouncementAttachment{{Name: "secreto.exe", URL: "https://files.example/a"}}}, "u1")
	if err != domain.ErrInvalidAnnouncement {
		t.Fatalf("error=%v", err)
	}
}
func TestCreateAcceptsAllowedAttachmentAndSetsAuthor(t *testing.T) {
	r := &repo{}
	s := announcement.NewService(r, users{domain.User{ID: "u1", Role: "Admin", Status: "activo"}})
	a, err := s.Create(context.Background(), domain.Announcement{Title: "Directriz", Description: "Texto", Attachments: []domain.AnnouncementAttachment{{Name: "guia.pdf", URL: "https://files.example/guia.pdf"}}}, "u1")
	if err != nil || a.AuthorID != "u1" || r.created.AuthorID != "u1" {
		t.Fatalf("announcement=%+v error=%v", a, err)
	}
}

func TestListIncludesAllDraftsOnlyForAdmin(t *testing.T) {
	adminRepo := &repo{}
	adminService := announcement.NewService(adminRepo, users{domain.User{ID: "admin", Role: "Admin", Status: "activo"}})
	if _, err := adminService.List(context.Background(), "admin"); err != nil || !adminRepo.includeAllDrafts {
		t.Fatalf("admin list did not request all drafts: error=%v includeAllDrafts=%v", err, adminRepo.includeAllDrafts)
	}
	teacherRepo := &repo{}
	teacherService := announcement.NewService(teacherRepo, users{domain.User{ID: "teacher", Role: "Docente", Status: "activo"}})
	if _, err := teacherService.List(context.Background(), "teacher"); err != nil || teacherRepo.includeAllDrafts {
		t.Fatalf("teacher list requested other drafts: error=%v includeAllDrafts=%v", err, teacherRepo.includeAllDrafts)
	}
}

func TestAdminFindRequestsOtherAuthorsDrafts(t *testing.T) {
	r := &repo{created: domain.Announcement{ID: "draft", Status: "borrador", AuthorID: "other"}}
	s := announcement.NewService(r, users{domain.User{ID: "admin", Role: "Admin", Status: "activo"}})
	if _, err := s.Find(context.Background(), "draft", "admin"); err != nil || !r.findAllDrafts {
		t.Fatalf("admin cannot request another author's draft: error=%v findAllDrafts=%v", err, r.findAllDrafts)
	}
}

func TestDocenteFindDoesNotRequestOtherAuthorsDrafts(t *testing.T) {
	r := &repo{created: domain.Announcement{ID: "draft", Status: "borrador", AuthorID: "other"}}
	s := announcement.NewService(r, users{domain.User{ID: "teacher", Role: "Docente", Status: "activo"}})
	if _, err := s.Find(context.Background(), "draft", "teacher"); err != nil || r.findAllDrafts {
		t.Fatalf("teacher requested another author's draft: error=%v findAllDrafts=%v", err, r.findAllDrafts)
	}
}

// PT-TR-01: an inactive Docente with a still-resolvable identity is rejected by every operation.
func TestInactiveTeacherIsForbidden(t *testing.T) {
	s := announcement.NewService(&repo{}, users{domain.User{ID: "u1", Role: "Docente", Status: "inactivo"}})
	ctx := context.Background()
	if _, err := s.List(ctx, "u1"); err != domain.ErrForbidden {
		t.Errorf("list err=%v", err)
	}
	if _, err := s.Create(ctx, domain.Announcement{Title: "x", Description: "x"}, "u1"); err != domain.ErrForbidden {
		t.Errorf("create err=%v", err)
	}
	if err := s.Confirm(ctx, "a1", "u1"); err != domain.ErrForbidden {
		t.Errorf("confirm err=%v", err)
	}
}
