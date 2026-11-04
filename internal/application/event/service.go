package event

import (
	"context"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
)

type Service struct {
	repo     ports.EventRepository
	users    ports.UserReader
	seedbeds ports.SeedbedRepository
}

func NewService(repo ports.EventRepository, users ports.UserReader, seedbeds ports.SeedbedRepository) *Service {
	return &Service{repo: repo, users: users, seedbeds: seedbeds}
}

func isRole(u domain.User, roles ...string) bool {
	for _, role := range roles {
		if strings.EqualFold(u.Role, role) {
			return true
		}
	}
	return false
}

// actorCanManage allows Admin, the organizing Docente, or a manager (director or delegate) of the event's seedbed.
func (s *Service) actorCanManage(ctx context.Context, actor string, e domain.Event) error {
	u, err := s.users.FindByID(ctx, actor)
	if err != nil {
		return domain.ErrForbidden
	}
	if isRole(u, "admin") || isRole(u, "docente", "profesor") && actor == e.OrganizerID {
		return nil
	}
	if e.SeedbedID != "" && s.seedbeds != nil {
		managed, err := s.seedbeds.ManagedSeedbeds(ctx, actor)
		if err != nil {
			return err
		}
		if slices.Contains(managed, e.SeedbedID) {
			return nil
		}
	}
	return domain.ErrForbidden
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
	validStatuses := map[string]bool{"": true, "borrador": true, "publicado": true, "cancelado": true, "oculto": true, "proximo": true, "vencido": true}
	if !validStatuses[status] {
		return nil, domain.ErrInvalidEvent
	}
	admin := actor != nil && isRole(*actor, "admin")
	if !admin && (status == "borrador" || status == "cancelado" || status == "oculto") {
		return nil, domain.ErrForbidden
	}
	repoStatus := status
	if !admin || status == "proximo" || status == "vencido" {
		repoStatus = "publicado"
	}
	items, err := s.repo.List(ctx, eventType, repoStatus, from, to)
	if err != nil || (status != "proximo" && status != "vencido") {
		return items, err
	}
	now := time.Now()
	filtered := items[:0]
	for _, item := range items {
		upcoming := !item.StartAt.Before(now)
		if status == "proximo" && upcoming || status == "vencido" && !upcoming {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}

func (s *Service) FindForUpdate(ctx context.Context, id, actor string) (domain.Event, error) {
	e, err := s.repo.Find(ctx, id)
	if err != nil {
		return e, err
	}
	if err := s.actorCanManage(ctx, actor, e); err != nil {
		return domain.Event{}, err
	}
	return e, nil
}

func (s *Service) Create(ctx context.Context, event domain.Event, actor string) (domain.Event, error) {
	event.SeedbedID = strings.TrimSpace(event.SeedbedID)
	if event.SeedbedID != "" {
		if s.seedbeds == nil {
			return event, domain.ErrInvalidEvent
		}
		if _, err := s.seedbeds.Find(ctx, event.SeedbedID); err != nil {
			return event, domain.ErrInvalidEvent
		}
		// The creator becomes organizer, so a seedbed event must come from Admin or one of its managers.
		if err := s.actorCanManage(ctx, actor, domain.Event{SeedbedID: event.SeedbedID}); err != nil {
			return event, err
		}
	} else if u, err := s.users.FindByID(ctx, actor); err != nil || !isRole(u, "admin", "docente", "profesor") {
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
		if actor == "" || s.actorCanManage(ctx, actor, e) != nil {
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
	if err = s.actorCanManage(ctx, actor, old); err != nil {
		return update, err
	}
	if old.Status == "cancelado" {
		return update, domain.ErrInvalidEventTransition
	}
	update.ID, update.OrganizerID, update.Status, update.SeedbedID = old.ID, old.OrganizerID, old.Status, old.SeedbedID
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
	if err = s.actorCanManage(ctx, actor, e); err != nil {
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
