package member

import (
	"context"
	"net/url"
	"strings"

	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
)

type Service struct{ repo ports.MemberRepository }

func NewService(repo ports.MemberRepository) *Service { return &Service{repo: repo} }

type Filter = ports.MemberFilter
type CreateCommand = ports.CreateMemberCommand
type UpdateCommand = ports.UpdateMemberCommand

func validateRole(role string) bool {
	return strings.EqualFold(role, "estudiante") || strings.EqualFold(role, "profesor") || strings.EqualFold(role, "docente")
}
func validURL(value string) bool {
	if value == "" {
		return true
	}
	u, err := url.ParseRequestURI(value)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func (s *Service) List(ctx context.Context, f Filter) ([]domain.MemberProfile, error) {
	if f.Limit < 1 {
		f.Limit = 20
	}
	if f.Limit > 100 {
		f.Limit = 100
	}
	return s.repo.List(ctx, f)
}
func (s *Service) Get(ctx context.Context, id string, private bool) (domain.MemberProfile, error) {
	if strings.TrimSpace(id) == "" {
		return domain.MemberProfile{}, domain.ErrMemberNotFound
	}
	p, err := s.repo.Find(ctx, id)
	if err != nil {
		return p, err
	}
	if !private {
		p.Email, p.DocumentType, p.DocumentNumber, p.FirebaseUID = "", "", "", ""
	}
	return p, nil
}
func (s *Service) Create(ctx context.Context, c CreateCommand) (domain.MemberProfile, error) {
	if strings.TrimSpace(c.FirebaseUID) == "" || strings.TrimSpace(c.Name) == "" || strings.TrimSpace(c.LastName) == "" || strings.TrimSpace(c.Email) == "" || !validateRole(c.Role) || !validURL(c.CvLAC) {
		return domain.MemberProfile{}, domain.ErrInvalidMember
	}
	return s.repo.CreateMember(ctx, c)
}
func (s *Service) Update(ctx context.Context, id string, c UpdateCommand) (domain.MemberProfile, error) {
	if id == "" || (c.Role != "" && !validateRole(c.Role)) || !validURL(c.CvLAC) {
		return domain.MemberProfile{}, domain.ErrInvalidMember
	}
	return s.repo.Update(ctx, id, c)
}
func (s *Service) SetStatus(ctx context.Context, id, status string) (domain.MemberProfile, error) {
	status = strings.ToLower(strings.TrimSpace(status))
	if status != "activo" && status != "inactivo" {
		return domain.MemberProfile{}, domain.ErrInvalidMemberStatus
	}
	return s.repo.SetStatus(ctx, id, status)
}
func (s *Service) Publications(ctx context.Context, id string) ([]domain.Publication, error) {
	return s.repo.Publications(ctx, id)
}
