package announcement

import (
	"context"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
	"net/url"
	"path/filepath"
	"strings"
)

type Service struct {
	repo  ports.AnnouncementRepository
	users ports.UserReader
}

func NewService(repo ports.AnnouncementRepository, users ports.UserReader) *Service {
	return &Service{repo: repo, users: users}
}
func allowed(u domain.User) bool {
	return strings.EqualFold(u.Role, "admin") || strings.EqualFold(u.Role, "docente")
}
func (s *Service) actor(ctx context.Context, id string) (domain.User, error) {
	u, e := s.users.FindByID(ctx, id)
	if e != nil || !allowed(u) || !strings.EqualFold(u.Status, "activo") {
		return u, domain.ErrForbidden
	}
	return u, nil
}
func validAttachment(a domain.AnnouncementAttachment) bool {
	if strings.TrimSpace(a.Name) == "" || strings.TrimSpace(a.URL) == "" {
		return false
	}
	ext := strings.ToLower(filepath.Ext(a.Name))
	if ext != ".pdf" && ext != ".doc" && ext != ".docx" && ext != ".xls" && ext != ".xlsx" {
		return false
	}
	u, e := url.Parse(a.URL)
	return e == nil && (u.Scheme == "https" || u.Scheme == "http") && u.Host != ""
}
func valid(a domain.Announcement) bool {
	if strings.TrimSpace(a.Title) == "" || strings.TrimSpace(a.Description) == "" || len(a.Attachments) > 10 {
		return false
	}
	for _, x := range a.Attachments {
		if !validAttachment(x) {
			return false
		}
	}
	return true
}
func (s *Service) List(ctx context.Context, actor string) ([]domain.Announcement, error) {
	u, e := s.actor(ctx, actor)
	if e != nil {
		return nil, e
	}
	return s.repo.List(ctx, actor, strings.EqualFold(u.Role, "admin"))
}
func (s *Service) Find(ctx context.Context, id, actor string) (domain.Announcement, error) {
	if _, e := s.actor(ctx, actor); e != nil {
		return domain.Announcement{}, e
	}
	return s.repo.Find(ctx, id, actor)
}
func (s *Service) Create(ctx context.Context, a domain.Announcement, actor string) (domain.Announcement, error) {
	if _, e := s.actor(ctx, actor); e != nil {
		return a, e
	}
	if !valid(a) {
		return a, domain.ErrInvalidAnnouncement
	}
	a.AuthorID = actor
	return s.repo.Create(ctx, a)
}
func (s *Service) Update(ctx context.Context, id string, a domain.Announcement, actor string) (domain.Announcement, error) {
	u, e := s.actor(ctx, actor)
	if e != nil {
		return a, e
	}
	old, e := s.repo.Find(ctx, id, actor)
	if e != nil {
		return a, e
	}
	if !strings.EqualFold(u.Role, "admin") && old.AuthorID != actor {
		return a, domain.ErrForbidden
	}
	if old.Status != "borrador" {
		return a, domain.ErrInvalidAnnouncementTransition
	}
	if a.Title == "" {
		a.Title = old.Title
	}
	if a.Description == "" {
		a.Description = old.Description
	}
	if a.Attachments == nil {
		a.Attachments = old.Attachments
	}
	if !valid(a) {
		return a, domain.ErrInvalidAnnouncement
	}
	return s.repo.Update(ctx, id, a.Title, a.Description, a.Attachments)
}
func (s *Service) Publish(ctx context.Context, id, actor string) (domain.Announcement, error) {
	u, e := s.actor(ctx, actor)
	if e != nil {
		return domain.Announcement{}, e
	}
	old, e := s.repo.Find(ctx, id, actor)
	if e != nil {
		return domain.Announcement{}, e
	}
	if !strings.EqualFold(u.Role, "admin") && old.AuthorID != actor {
		return domain.Announcement{}, domain.ErrForbidden
	}
	if old.Status != "borrador" {
		return old, domain.ErrInvalidAnnouncementTransition
	}
	return s.repo.Publish(ctx, id)
}
func (s *Service) Confirm(ctx context.Context, id, actor string) error {
	if _, e := s.actor(ctx, actor); e != nil {
		return e
	}
	if _, e := s.repo.Find(ctx, id, actor); e != nil {
		return e
	}
	return s.repo.ConfirmRead(ctx, id, actor)
}
func (s *Service) ReadStatus(ctx context.Context, id, actor string) ([]domain.AnnouncementReadStatus, error) {
	u, e := s.actor(ctx, actor)
	if e != nil {
		return nil, e
	}
	a, e := s.repo.Find(ctx, id, actor)
	if e != nil {
		return nil, e
	}
	if !strings.EqualFold(u.Role, "admin") && a.AuthorID != actor {
		return nil, domain.ErrForbidden
	}
	return s.repo.ReadStatus(ctx, id)
}
