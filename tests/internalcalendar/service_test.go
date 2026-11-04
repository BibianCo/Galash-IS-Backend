package internalcalendar_test

import (
	"context"
	"testing"
	"time"

	calendarapp "github.com/galash-uptc/backend/internal/application/internalcalendar"
	"github.com/galash-uptc/backend/internal/domain"
)

func TestCreateRejectsInvalidTypeAndRange(t *testing.T) {
	s := calendarapp.NewService(&fakeRepository{})
	actor := domain.User{ID: "teacher", Role: "docente"}
	e := domain.InternalCalendarEvent{SeedbedID: "seedbed", Type: "otro", Title: "Tutoría", StartAt: time.Now(), EndAt: time.Now().Add(-time.Hour), ParticipantIDs: []string{"student"}}
	if _, err := s.Create(context.Background(), e, actor); err != domain.ErrInvalidInternalCalendar {
		t.Fatalf("error = %v, want invalid calendar", err)
	}
}

// Contract M9: writes depend on being seedbed director/delegate or Admin, not on the Docente role.
func TestCreateRequiresSeedbedManager(t *testing.T) {
	e := domain.InternalCalendarEvent{SeedbedID: "seedbed", Type: "hito", Title: "Entrega", StartAt: time.Now(), EndAt: time.Now().Add(time.Hour), ParticipantIDs: []string{"student"}}
	if _, err := calendarapp.NewService(&fakeRepository{}).Create(context.Background(), e, domain.User{ID: "teacher", Role: "docente"}); err != domain.ErrForbidden {
		t.Fatalf("non-manager docente error = %v, want forbidden", err)
	}
	if _, err := calendarapp.NewService(&fakeRepository{manager: true}).Create(context.Background(), e, domain.User{ID: "delegate", Role: "estudiante"}); err != nil {
		t.Fatalf("delegate student error = %v, want nil", err)
	}
}

type fakeRepository struct{ manager bool }

func (*fakeRepository) List(context.Context, string, string, string, *time.Time, *time.Time) ([]domain.InternalCalendarEvent, error) {
	return nil, nil
}
func (*fakeRepository) Find(context.Context, string, string, string) (domain.InternalCalendarEvent, error) {
	return domain.InternalCalendarEvent{}, nil
}
func (*fakeRepository) Create(context.Context, domain.InternalCalendarEvent) (domain.InternalCalendarEvent, error) {
	return domain.InternalCalendarEvent{}, nil
}
func (*fakeRepository) Update(context.Context, domain.InternalCalendarEvent) (domain.InternalCalendarEvent, error) {
	return domain.InternalCalendarEvent{}, nil
}
func (*fakeRepository) Cancel(context.Context, string, string) (domain.InternalCalendarEvent, error) {
	return domain.InternalCalendarEvent{}, nil
}
func (r *fakeRepository) CanManage(context.Context, domain.InternalCalendarEvent, string, string) (bool, error) {
	return r.manager, nil
}
func (*fakeRepository) HasConflict(context.Context, domain.InternalCalendarEvent, string) (bool, error) {
	return false, nil
}
