package internalcalendar

import (
	"context"
	"strings"
	"time"

	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
)

type Service struct {
	repo ports.InternalCalendarRepository
}

func NewService(repo ports.InternalCalendarRepository) *Service { return &Service{repo: repo} }

func roleAllowed(role string, roles ...string) bool {
	for _, allowed := range roles {
		if strings.EqualFold(role, allowed) {
			return true
		}
	}
	return false
}

func valid(e domain.InternalCalendarEvent) bool {
	typ := strings.ToLower(strings.TrimSpace(e.Type))
	return e.SeedbedID != "" && strings.TrimSpace(e.Title) != "" && !e.StartAt.IsZero() && !e.EndAt.IsZero() && e.EndAt.After(e.StartAt) && (typ == "reunion" || typ == "tutoria" || typ == "hito") && len(e.ParticipantIDs) > 0
}

func (s *Service) List(ctx context.Context, actor domain.User, seedbedID string, from, to *time.Time) ([]domain.InternalCalendarEvent, error) {
	if !roleAllowed(actor.Role, "estudiante", "docente", "profesor", "admin") {
		return nil, domain.ErrForbidden
	}
	if from != nil && to != nil && to.Before(*from) {
		return nil, domain.ErrInvalidInternalCalendar
	}
	return s.repo.List(ctx, actor.ID, actor.Role, seedbedID, from, to)
}

func (s *Service) Create(ctx context.Context, e domain.InternalCalendarEvent, actor domain.User) (domain.InternalCalendarEvent, error) {
	if !roleAllowed(actor.Role, "docente", "profesor", "admin") {
		return e, domain.ErrForbidden
	}
	if !valid(e) {
		return e, domain.ErrInvalidInternalCalendar
	}
	e.OrganizerID = actor.ID
	conflict, err := s.repo.HasConflict(ctx, e, "")
	if err != nil {
		return e, err
	}
	if conflict {
		return e, domain.ErrInternalCalendarConflict
	}
	e.Status = "activo"
	return s.repo.Create(ctx, e)
}

func (s *Service) Find(ctx context.Context, id string, actor domain.User) (domain.InternalCalendarEvent, error) {
	if !roleAllowed(actor.Role, "estudiante", "docente", "profesor", "admin") {
		return domain.InternalCalendarEvent{}, domain.ErrForbidden
	}
	return s.repo.Find(ctx, actor.ID, actor.Role, id)
}

func (s *Service) Update(ctx context.Context, id string, e domain.InternalCalendarEvent, actor domain.User) (domain.InternalCalendarEvent, error) {
	if !roleAllowed(actor.Role, "docente", "profesor", "admin") {
		return e, domain.ErrForbidden
	}
	old, err := s.repo.Find(ctx, actor.ID, actor.Role, id)
	if err != nil {
		return e, err
	}
	can, err := s.repo.CanManage(ctx, old, actor.ID, actor.Role)
	if err != nil {
		return e, err
	}
	if !can {
		return e, domain.ErrForbidden
	}
	if old.Status == "cancelado" {
		return e, domain.ErrInvalidInternalCalendarTransition
	}
	e.ID, e.SeedbedID, e.OrganizerID, e.Status = old.ID, old.SeedbedID, old.OrganizerID, old.Status
	if !valid(e) {
		return e, domain.ErrInvalidInternalCalendar
	}
	conflict, err := s.repo.HasConflict(ctx, e, id)
	if err != nil {
		return e, err
	}
	if conflict {
		return e, domain.ErrInternalCalendarConflict
	}
	return s.repo.Update(ctx, e)
}

func (s *Service) Cancel(ctx context.Context, id string, actor domain.User) (domain.InternalCalendarEvent, error) {
	if !roleAllowed(actor.Role, "docente", "profesor", "admin") {
		return domain.InternalCalendarEvent{}, domain.ErrForbidden
	}
	e, err := s.repo.Find(ctx, actor.ID, actor.Role, id)
	if err != nil {
		return e, err
	}
	can, err := s.repo.CanManage(ctx, e, actor.ID, actor.Role)
	if err != nil {
		return e, err
	}
	if !can {
		return e, domain.ErrForbidden
	}
	if e.Status == "cancelado" {
		return e, domain.ErrInvalidInternalCalendarTransition
	}
	return s.repo.Cancel(ctx, id, actor.ID)
}
