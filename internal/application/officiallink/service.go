package officiallink

import (
	"context"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
	"net/url"
	"strings"
)

type Service struct {
	repo  ports.OfficialLinkRepository
	users ports.UserReader
}

func NewService(repo ports.OfficialLinkRepository, users ports.UserReader) *Service {
	return &Service{repo: repo, users: users}
}

var categories = map[string]string{
	"redes sociales": "redes-sociales", "redes_sociales": "redes-sociales", "redes-sociales": "redes-sociales",
	"cvlac": "cvlac", "gruplac": "gruplac", "minciencias": "minciencias", "repositorios": "repositorios",
}

func NormalizeCategory(value string) string {
	return categories[strings.ToLower(strings.TrimSpace(value))]
}
func validURL(value string) bool {
	u, err := url.ParseRequestURI(strings.TrimSpace(value))
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func (s *Service) authorize(ctx context.Context, actor string) error {
	u, err := s.users.FindByID(ctx, actor)
	if err != nil || !strings.EqualFold(u.Role, "admin") || !strings.EqualFold(u.Status, "activo") {
		return domain.ErrForbidden
	}
	return nil
}
func validate(l domain.OfficialLink) error {
	if strings.TrimSpace(l.Name) == "" || strings.TrimSpace(l.Description) == "" || !validURL(l.URL) || NormalizeCategory(l.Category) == "" || l.Order < 0 {
		return domain.ErrInvalidOfficialLink
	}
	l.Category = NormalizeCategory(l.Category)
	return nil
}
func (s *Service) List(ctx context.Context, category string, includeAll bool) ([]domain.OfficialLink, error) {
	if category != "" && NormalizeCategory(category) == "" {
		return nil, domain.ErrInvalidOfficialLinkCategory
	}
	return s.repo.ListVisible(ctx, NormalizeCategory(category), includeAll)
}
func (s *Service) Find(ctx context.Context, id, actor string) (domain.OfficialLink, error) {
	if err := s.authorize(ctx, actor); err != nil {
		return domain.OfficialLink{}, err
	}
	return s.repo.Find(ctx, id)
}
func (s *Service) Create(ctx context.Context, link domain.OfficialLink, actor string) (domain.OfficialLink, error) {
	if err := s.authorize(ctx, actor); err != nil {
		return link, err
	}
	link.Category = NormalizeCategory(link.Category)
	link.Status = "activo"
	link.UpdatedBy = actor
	if link.Visible == false { /* false is a valid explicit visibility choice */
	}
	if err := validate(link); err != nil {
		return link, err
	}
	return s.repo.Create(ctx, link)
}
func (s *Service) Update(ctx context.Context, id, actor string, link domain.OfficialLink) (domain.OfficialLink, error) {
	if err := s.authorize(ctx, actor); err != nil {
		return link, err
	}
	link.Category = NormalizeCategory(link.Category)
	link.UpdatedBy = actor
	if err := validate(link); err != nil {
		return link, err
	}
	return s.repo.Update(ctx, id, actor, link)
}
func (s *Service) SetStatus(ctx context.Context, id, actor, status string) (domain.OfficialLink, error) {
	if err := s.authorize(ctx, actor); err != nil {
		return domain.OfficialLink{}, err
	}
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "activo", "inactivo", "oculto":
	default:
		return domain.OfficialLink{}, domain.ErrInvalidOfficialLinkStatus
	}
	return s.repo.SetStatus(ctx, id, actor, strings.ToLower(strings.TrimSpace(status)))
}
