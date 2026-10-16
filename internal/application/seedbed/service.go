package seedbed

import (
	"context"
	"strings"
	"time"

	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
)

type Service struct {
	repo  ports.SeedbedRepository
	users ports.UserReader
}

func NewService(repo ports.SeedbedRepository, users ports.UserReader) *Service {
	return &Service{repo: repo, users: users}
}

type CreateCommand struct{ Name, Description, ResearchLine, Objective, ResponsibleTeacherID, ContactEmail string }
type UpdateCommand = CreateCommand

func validate(c CreateCommand) error {
	if strings.TrimSpace(c.Name) == "" || strings.TrimSpace(c.Description) == "" || strings.TrimSpace(c.ResearchLine) == "" || strings.TrimSpace(c.Objective) == "" || strings.TrimSpace(c.ResponsibleTeacherID) == "" || strings.TrimSpace(c.ContactEmail) == "" {
		return domain.ErrInvalidSeedbed
	}
	if !strings.Contains(c.ContactEmail, "@") {
		return domain.ErrInvalidSeedbed
	}
	return nil
}
func (s *Service) teacher(ctx context.Context, id string) error {
	u, err := s.users.FindByID(ctx, strings.TrimSpace(id))
	if err != nil || u.Status != "activo" || !strings.EqualFold(u.Role, "profesor") && !strings.EqualFold(u.Role, "docente") {
		return domain.ErrInvalidTeacher
	}
	return nil
}
func (s *Service) Create(ctx context.Context, c CreateCommand) (domain.Seedbed, error) {
	if err := validate(c); err != nil {
		return domain.Seedbed{}, err
	}
	if err := s.teacher(ctx, c.ResponsibleTeacherID); err != nil {
		return domain.Seedbed{}, err
	}
	now := time.Now().UTC()
	return s.repo.Create(ctx, domain.Seedbed{Name: strings.TrimSpace(c.Name), Description: strings.TrimSpace(c.Description), ResearchLine: strings.TrimSpace(c.ResearchLine), Objective: strings.TrimSpace(c.Objective), ResponsibleTeacherID: strings.TrimSpace(c.ResponsibleTeacherID), ContactEmail: strings.TrimSpace(c.ContactEmail), Status: domain.SeedbedDraft, Visibility: domain.SeedbedDraft, CreatedAt: now, UpdatedAt: now})
}
func (s *Service) Update(ctx context.Context, id string, c UpdateCommand) (domain.Seedbed, error) {
	if err := validate(c); err != nil {
		return domain.Seedbed{}, err
	}
	if err := s.teacher(ctx, c.ResponsibleTeacherID); err != nil {
		return domain.Seedbed{}, err
	}
	return s.repo.Update(ctx, id, domain.Seedbed{Name: c.Name, Description: c.Description, ResearchLine: c.ResearchLine, Objective: c.Objective, ResponsibleTeacherID: c.ResponsibleTeacherID, ContactEmail: c.ContactEmail, UpdatedAt: time.Now().UTC()})
}
func (s *Service) CanManage(ctx context.Context, actorID, id string) error {
	sb, err := s.repo.Find(ctx, id)
	if err != nil {
		return domain.ErrSeedbedNotFound
	}
	if actor, err := s.users.FindByID(ctx, actorID); err == nil && strings.EqualFold(actor.Role, "admin") {
		return nil
	}
	if sb.ResponsibleTeacherID != actorID {
		return domain.ErrForbidden
	}
	return nil
}
func (s *Service) List(ctx context.Context, q, line string, page, size int, private bool) ([]domain.Seedbed, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 100 {
		size = 100
	}
	return s.repo.List(ctx, q, line, page, size, private)
}
func (s *Service) Find(ctx context.Context, id string) (domain.Seedbed, error) {
	return s.repo.Find(ctx, id)
}
func (s *Service) Visibility(ctx context.Context, actor, id, status string) (domain.Seedbed, error) {
	if err := s.CanManage(ctx, actor, id); err != nil {
		return domain.Seedbed{}, err
	}
	if status != domain.SeedbedPublished && status != domain.SeedbedHidden && status != domain.SeedbedInactive && status != domain.SeedbedDraft {
		return domain.Seedbed{}, domain.ErrInvalidSeedbedStatus
	}
	return s.repo.SetVisibility(ctx, id, status)
}
func (s *Service) Members(ctx context.Context, id string, history bool) ([]domain.SeedbedMember, error) {
	return s.repo.ListMembers(ctx, id, history)
}
func (s *Service) AddMember(ctx context.Context, actor, id, userID string) error {
	if err := s.CanManage(ctx, actor, id); err != nil {
		return err
	}
	u, err := s.users.FindByID(ctx, userID)
	if err != nil || u.Status != "activo" || (u.Role != "estudiante" && u.Role != "profesor" && u.Role != "docente") {
		return domain.ErrForbidden
	}
	return s.repo.AddMember(ctx, id, userID)
}
func (s *Service) RemoveMember(ctx context.Context, actor, id, userID string) error {
	if err := s.CanManage(ctx, actor, id); err != nil {
		return err
	}
	return s.repo.RemoveMember(ctx, id, userID)
}
