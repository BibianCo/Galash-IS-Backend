package httpadapter

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/galash-uptc/backend/internal/domain"
)

type internalCalendarRequest struct {
	SeedbedID      string     `json:"seedbedId"`
	Type           string     `json:"type"`
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	StartAt        *time.Time `json:"startAt"`
	EndAt          *time.Time `json:"endAt"`
	ParticipantIDs []string   `json:"participantIds"`
}
type internalCalendarPatch struct {
	SeedbedID      *string    `json:"seedbedId"`
	Type           *string    `json:"type"`
	Title          *string    `json:"title"`
	Description    *string    `json:"description"`
	StartAt        *time.Time `json:"startAt"`
	EndAt          *time.Time `json:"endAt"`
	ParticipantIDs *[]string  `json:"participantIds"`
}

func (h *Handler) calendarReady(w http.ResponseWriter) bool {
	if h.calendar == nil {
		writeError(w, 500, "módulo de cronograma interno no configurado")
		return false
	}
	return true
}
func calendarError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		writeError(w, 403, err.Error())
	case errors.Is(err, domain.ErrInternalCalendarNotFound):
		writeError(w, 404, err.Error())
	case errors.Is(err, domain.ErrInternalCalendarConflict):
		writeError(w, 409, err.Error())
	case errors.Is(err, domain.ErrInvalidInternalCalendar), errors.Is(err, domain.ErrInvalidInternalCalendarTransition):
		writeError(w, 400, err.Error())
	default:
		writeError(w, 500, "no fue posible procesar el cronograma interno")
	}
}
func (h *Handler) calendarActor(w http.ResponseWriter, r *http.Request) (domain.User, bool) {
	token, err := sessionToken(r)
	if err != nil {
		writeError(w, 401, "sesión requerida")
		return domain.User{}, false
	}
	u, err := h.service.CurrentUser(r.Context(), token)
	if err != nil {
		invalidSession(w, r, "sesión inválida o expirada")
		return domain.User{}, false
	}
	if strings.EqualFold(u.Status, "inactivo") {
		writeError(w, 403, "la cuenta está inactiva")
		return domain.User{}, false
	}
	return u, true
}
func calendarTime(value string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	return &t, nil
}
func (h *Handler) listInternalCalendar(w http.ResponseWriter, r *http.Request) {
	if !h.calendarReady(w) {
		return
	}
	actor, ok := h.calendarActor(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	from, err := calendarTime(q.Get("from"))
	if err != nil {
		writeError(w, 400, "from debe ser una fecha RFC3339")
		return
	}
	to, err := calendarTime(q.Get("to"))
	if err != nil {
		writeError(w, 400, "to debe ser una fecha RFC3339")
		return
	}
	view := q.Get("view")
	if view != "" && view != "week" && view != "weekly" && view != "month" && view != "monthly" {
		writeError(w, 400, "view debe ser week o month")
		return
	}
	items, err := h.calendar.List(r.Context(), actor, q.Get("seedbedId"), from, to)
	if err != nil {
		calendarError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"events": items})
}
func decodeInternalCalendar(r *http.Request) (internalCalendarRequest, error) {
	var q internalCalendarRequest
	err := json.NewDecoder(r.Body).Decode(&q)
	if err != nil {
		return q, err
	}
	return q, nil
}
func (h *Handler) createInternalCalendar(w http.ResponseWriter, r *http.Request) {
	if !h.calendarReady(w) {
		return
	}
	actor, ok := h.calendarActor(w, r)
	if !ok {
		return
	}
	q, err := decodeInternalCalendar(r)
	if err != nil || q.StartAt == nil || q.EndAt == nil {
		writeError(w, 400, "cuerpo JSON inválido: startAt y endAt son obligatorios")
		return
	}
	e := domain.InternalCalendarEvent{SeedbedID: q.SeedbedID, Type: q.Type, Title: q.Title, Description: q.Description, StartAt: *q.StartAt, EndAt: *q.EndAt, ParticipantIDs: q.ParticipantIDs}
	out, err := h.calendar.Create(r.Context(), e, actor)
	if err != nil {
		calendarError(w, err)
		return
	}
	writeJSON(w, 201, out)
}
func (h *Handler) getInternalCalendar(w http.ResponseWriter, r *http.Request) {
	if !h.calendarReady(w) {
		return
	}
	actor, ok := h.calendarActor(w, r)
	if !ok {
		return
	}
	e, err := h.calendar.Find(r.Context(), r.PathValue("id"), actor)
	if err != nil {
		calendarError(w, err)
		return
	}
	writeJSON(w, 200, e)
}
func (h *Handler) updateInternalCalendar(w http.ResponseWriter, r *http.Request) {
	if !h.calendarReady(w) {
		return
	}
	actor, ok := h.calendarActor(w, r)
	if !ok {
		return
	}
	old, err := h.calendar.Find(r.Context(), r.PathValue("id"), actor)
	if err != nil {
		calendarError(w, err)
		return
	}
	var q internalCalendarPatch
	if err = json.NewDecoder(r.Body).Decode(&q); err != nil {
		writeError(w, 400, "cuerpo JSON inválido")
		return
	}
	if q.SeedbedID != nil {
		old.SeedbedID = *q.SeedbedID
	}
	if q.Type != nil {
		old.Type = *q.Type
	}
	if q.Title != nil {
		old.Title = *q.Title
	}
	if q.Description != nil {
		old.Description = *q.Description
	}
	if q.StartAt != nil {
		old.StartAt = *q.StartAt
	}
	if q.EndAt != nil {
		old.EndAt = *q.EndAt
	}
	if q.ParticipantIDs != nil {
		old.ParticipantIDs = *q.ParticipantIDs
	}
	out, err := h.calendar.Update(r.Context(), old.ID, old, actor)
	if err != nil {
		calendarError(w, err)
		return
	}
	writeJSON(w, 200, out)
}
func (h *Handler) deleteInternalCalendar(w http.ResponseWriter, r *http.Request) {
	if !h.calendarReady(w) {
		return
	}
	actor, ok := h.calendarActor(w, r)
	if !ok {
		return
	}
	out, err := h.calendar.Cancel(r.Context(), r.PathValue("id"), actor)
	if err != nil {
		calendarError(w, err)
		return
	}
	writeJSON(w, 200, out)
}
