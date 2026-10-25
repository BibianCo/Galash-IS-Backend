package event

import (
	"context"
	"net/url"
	"strings"
	"time"

	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
)

type Service struct {
	repo  ports.EventRepository
	users ports.UserReader
}

func NewService(repo ports.EventRepository, users ports.UserReader) *Service {
	return &Service{repo: repo, users: users}
}

func isRole(u domain.User, roles ...string) bool {
	for _, role := range roles {
		if strings.EqualFold(u.Role, role) {
			return true
		}
	}
	return false
}

func (s *Service) actorCanManage(ctx context.Context, actor, owner string) error {
	u, err := s.users.FindByID(ctx, actor)
	if err != nil || !isRole(u, "admin", "docente", "profesor") {
		return domain.ErrForbidden
	}
	if !isRole(u, "admin") && actor != owner {
		return domain.ErrForbidden
	}
	return nil
}

func validURL(value string) bool {
	if value == "" {
		return true
	}
	u, err := url.Parse(value)
	return err == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host != ""
}

func validEvent(e domain.Event) bool {
	return strings.TrimSpace(e.Title) != "" && strings.TrimSpace(e.Type) != "" &&
		!e.StartAt.IsZero() && (e.EndAt == nil || !e.EndAt.Before(e.StartAt)) &&
		validURL(e.MeetingURL) && validURL(e.ImageURL) && validURL(e.RegistrationURL)
}

func (s *Service) List(ctx context.Context, eventType, status string, from, to *time.Time, actor *domain.User) ([]domain.Event, error) {
	// The public agenda never exposes drafts, hidden or cancelled events.
	if actor == nil || !isRole(*actor, "admin") {
		if status == "" || status == "proximo" || status == "vencido" {
			status = "publicado"
		}
	}
	return s.repo.List(ctx, eventType, status, from, to)
}

func (s *Service) Create(ctx context.Context, event domain.Event, actor string) (domain.Event, error) {
	u, err := s.users.FindByID(ctx, actor)
	if err != nil || !isRole(u, "admin", "docente", "profesor") {
		return event, domain.ErrForbidden
	}
	if !validEvent(event) {
		return event, domain.ErrInvalidEvent
	}
	conflict, err := s.repo.HasConflict(ctx, event, "")
	if err != nil {
		return event, err
	}
	if conflict {
		return event, domain.ErrEventConflict
	}
	event.OrganizerID, event.Status = actor, "borrador"
	return s.repo.Create(ctx, event)
}

func (s *Service) Find(ctx context.Context, id string, actor string) (domain.Event, error) {
	e, err := s.repo.Find(ctx, id)
	if err != nil {
		return e, err
	}
	if e.Status != "publicado" {
		if actor == "" || s.actorCanManage(ctx, actor, e.OrganizerID) != nil {
			return domain.Event{}, domain.ErrEventNotFound
		}
	}
	return e, nil
}

func (s *Service) Update(ctx context.Context, id string, update domain.Event, actor string) (domain.Event, error) {
	old, err := s.repo.Find(ctx, id)
	if err != nil {
		return update, err
	}
	if err = s.actorCanManage(ctx, actor, old.OrganizerID); err != nil {
		return update, err
	}
	if old.Status == "cancelado" {
		return update, domain.ErrInvalidEventTransition
	}
	update.ID, update.OrganizerID, update.Status = old.ID, old.OrganizerID, old.Status
	if !validEvent(update) {
		return update, domain.ErrInvalidEvent
	}
	conflict, err := s.repo.HasConflict(ctx, update, id)
	if err != nil {
		return update, err
	}
	if conflict {
		return update, domain.ErrEventConflict
	}
	return s.repo.Update(ctx, id, update)
}

func (s *Service) SetStatus(ctx context.Context, id, status, actor string) (domain.Event, error) {
	e, err := s.repo.Find(ctx, id)
	if err != nil {
		return e, err
	}
	if err = s.actorCanManage(ctx, actor, e.OrganizerID); err != nil {
		return e, err
	}
	if status == "publicado" && e.Status != "borrador" && e.Status != "oculto" {
		return e, domain.ErrInvalidEventTransition
	}
	if status == "cancelado" && e.Status == "cancelado" {
		return e, domain.ErrInvalidEventTransition
	}
	if status == "oculto" && e.Status == "cancelado" {
		return e, domain.ErrInvalidEventTransition
	}
	return s.repo.SetStatus(ctx, id, status)
}
