package httpadapter

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	application "github.com/galash-uptc/backend/internal/application/auth"
	eventapp "github.com/galash-uptc/backend/internal/application/event"
	memberapp "github.com/galash-uptc/backend/internal/application/member"
	registrationapp "github.com/galash-uptc/backend/internal/application/registration"
	researchapp "github.com/galash-uptc/backend/internal/application/research"
	seedbedapp "github.com/galash-uptc/backend/internal/application/seedbed"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
)

type registerRequest struct {
	Nombre   string `json:"nombre"`
	Apellido string `json:"apellido"`
	TipoDNI  string `json:"tipo_dni"`
	DNI      string `json:"dni"`
	Email    string `json:"email"`
	Codigo   string `json:"codigo_estudiantil"`
	Fecha    string `json:"fecha_nacimiento"`
	Telefono string `json:"telefono"`
}

type sessionRequest struct {
	Token string `json:"token"`
}
type recoveryRequest struct {
	Email string `json:"email"`
}
type resetRequest struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

type Handler struct {
	service       *application.Service
	seedbeds      *seedbedapp.Service
	registrations *registrationapp.Service
	research      *researchapp.Service
	members       *memberapp.Service
	events        *eventapp.Service
	db            ports.HealthChecker
}

func NewHandler(service *application.Service, db ports.HealthChecker, seedbeds ...*seedbedapp.Service) *Handler {
	return newHandler(service, db, seedbeds, nil, nil)
}

func NewHandlerWithRegistrations(service *application.Service, db ports.HealthChecker, seedbed *seedbedapp.Service, registration *registrationapp.Service) *Handler {
	return newHandler(service, db, []*seedbedapp.Service{seedbed}, registration, nil)
}
func NewHandlerWithResearch(service *application.Service, db ports.HealthChecker, seedbed *seedbedapp.Service, registration *registrationapp.Service, research *researchapp.Service) *Handler {
	h := newHandler(service, db, []*seedbedapp.Service{seedbed}, registration, nil)
	h.research = research
	return h
}

func NewHandlerWithResearchAndMembers(service *application.Service, db ports.HealthChecker, seedbed *seedbedapp.Service, registration *registrationapp.Service, research *researchapp.Service, members *memberapp.Service) *Handler {
	h := newHandler(service, db, []*seedbedapp.Service{seedbed}, registration, members)
	h.research = research
	return h
}

func NewHandlerWithEvents(service *application.Service, db ports.HealthChecker, seedbed *seedbedapp.Service, registration *registrationapp.Service, research *researchapp.Service, members *memberapp.Service, events *eventapp.Service) *Handler {
	h := NewHandlerWithResearchAndMembers(service, db, seedbed, registration, research, members)
	h.events = events
	return h
}

func NewHandlerWithMembers(service *application.Service, db ports.HealthChecker, seedbed *seedbedapp.Service, registration *registrationapp.Service, members *memberapp.Service) *Handler {
	return newHandler(service, db, []*seedbedapp.Service{seedbed}, registration, members)
}

func newHandler(service *application.Service, db ports.HealthChecker, seedbeds []*seedbedapp.Service, registration *registrationapp.Service, members *memberapp.Service) *Handler {
	var sb *seedbedapp.Service
	if len(seedbeds) > 0 {
		sb = seedbeds[0]
	}
	return &Handler{service: service, seedbeds: sb, registrations: registration, members: members, db: db}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", h.health)
	mux.HandleFunc("POST /auth/register", h.register)
	mux.HandleFunc("POST /auth/sessions", h.createSession)
	mux.HandleFunc("DELETE /auth/sessions/current", h.deleteSession)
	mux.HandleFunc("GET /users/me", h.currentUser)
	mux.HandleFunc("GET /members", h.publicMembers)
	mux.HandleFunc("POST /members", h.createMember)
	mux.HandleFunc("GET /members/{id}", h.getMember)
	mux.HandleFunc("PATCH /members/{id}", h.updateMember)
	mux.HandleFunc("PATCH /members/{id}/status", h.setMemberStatus)
	mux.HandleFunc("GET /members/{id}/publications", h.memberPublications)
	mux.HandleFunc("GET /admin/members", h.adminMembers)
	mux.HandleFunc("PUT /users/roles", h.updateRole)
	mux.HandleFunc("PUT /users/{userId}/roles", h.updateRole)
	mux.HandleFunc("GET /seedbeds", h.listSeedbeds)
	mux.HandleFunc("POST /seedbeds", h.createSeedbed)
	mux.HandleFunc("GET /seedbeds/{id}", h.getSeedbed)
	mux.HandleFunc("GET /seedbeds/{id}/space", h.getSeedbedSpace)
	mux.HandleFunc("PATCH /seedbeds/{id}", h.updateSeedbed)
	mux.HandleFunc("PATCH /seedbeds/{id}/visibility", h.setSeedbedVisibility)
	mux.HandleFunc("GET /seedbeds/{id}/members", h.listSeedbedMembers)
	mux.HandleFunc("POST /seedbeds/{id}/members", h.addSeedbedMember)
	mux.HandleFunc("DELETE /seedbeds/{id}/members/{userId}", h.removeSeedbedMember)
	mux.HandleFunc("POST /registration-requests", h.createRegistration)
	mux.HandleFunc("GET /registration-requests/{id}/status", h.registrationStatus)
	mux.HandleFunc("GET /admin/registration-requests", h.listRegistrationRequests)
	mux.HandleFunc("GET /admin/registration-requests/{id}", h.registrationDetail)
	mux.HandleFunc("PATCH /admin/registration-requests/{id}/status", h.updateRegistrationStatus)
	mux.HandleFunc("GET /seedbeds/{seedbedId}/activities", h.listActivities)
	mux.HandleFunc("POST /seedbeds/{seedbedId}/activities", h.createActivity)
	mux.HandleFunc("GET /activities/{id}", h.getActivity)
	mux.HandleFunc("PATCH /activities/{id}", h.updateActivity)
	mux.HandleFunc("GET /projects", h.listProjects)
	mux.HandleFunc("POST /projects", h.createProject)
	mux.HandleFunc("GET /projects/progress", h.projectProgress)
	mux.HandleFunc("GET /projects/{id}", h.getProject)
	mux.HandleFunc("PATCH /projects/{id}", h.updateProject)
	mux.HandleFunc("PATCH /projects/{id}/progress", h.updateProgress)
	mux.HandleFunc("GET /projects/{id}/history", h.projectHistory)
	mux.HandleFunc("GET /research-proposals", h.listProposals)
	mux.HandleFunc("POST /research-proposals", h.createProposal)
	mux.HandleFunc("GET /research-proposals/{id}", h.getProposal)
	mux.HandleFunc("PATCH /research-proposals/{id}", h.updateProposal)
	mux.HandleFunc("POST /research-proposals/{id}/convert-to-project", h.convertProposal)
	mux.HandleFunc("POST /seedbeds/{seedbedId}/reviews", h.createReview)
	mux.HandleFunc("GET /admin/seedbed-reviews", h.listReviews)
	mux.HandleFunc("PATCH /admin/seedbed-reviews/{id}/moderation", h.moderateReview)
	mux.HandleFunc("GET /events", h.listEvents)
	mux.HandleFunc("POST /events", h.createEvent)
	mux.HandleFunc("GET /events/{id}", h.getEvent)
	mux.HandleFunc("PATCH /events/{id}", h.updateEvent)
	mux.HandleFunc("POST /events/{id}/publish", h.publishEvent)
	mux.HandleFunc("POST /events/{id}/cancel", h.cancelEvent)
	mux.HandleFunc("POST /events/{id}/hide", h.hideEvent)
	return localCORS(mux)
}

func localCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if isLocalOrigin(origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Add("Vary", "Origin")
		}
		if r.Method == http.MethodOptions {
			if !isLocalOrigin(origin) {
				writeError(w, http.StatusForbidden, "origen no permitido")
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

type eventRequest struct {
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	Type            string     `json:"type"`
	StartAt         *time.Time `json:"startAt"`
	EndAt           *time.Time `json:"endAt"`
	Venue           string     `json:"venue"`
	MeetingURL      string     `json:"meetingUrl"`
	ImageURL        string     `json:"imageUrl"`
	RegistrationURL string     `json:"registrationUrl"`
}
type eventPatchRequest struct {
	Title           *string         `json:"title"`
	Description     *string         `json:"description"`
	Type            *string         `json:"type"`
	StartAt         *time.Time      `json:"startAt"`
	EndAt           json.RawMessage `json:"endAt"`
	Venue           *string         `json:"venue"`
	MeetingURL      *string         `json:"meetingUrl"`
	ImageURL        *string         `json:"imageUrl"`
	RegistrationURL *string         `json:"registrationUrl"`
}

func (h *Handler) eventReady(w http.ResponseWriter) bool {
	if h.events == nil {
		writeError(w, http.StatusInternalServerError, "módulo de eventos no configurado")
		return false
	}
	return true
}
func eventError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		writeError(w, http.StatusForbidden, err.Error())
	case errors.Is(err, domain.ErrEventNotFound):
		writeError(w, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrEventConflict):
		writeError(w, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrInvalidEvent), errors.Is(err, domain.ErrInvalidEventTransition):
		writeError(w, http.StatusBadRequest, err.Error())
	default:
		writeError(w, http.StatusInternalServerError, "no fue posible procesar el evento")
	}
}
func eventFromRequest(q eventRequest) domain.Event {
	e := domain.Event{Title: q.Title, Description: q.Description, Type: q.Type, EndAt: q.EndAt, Venue: q.Venue, MeetingURL: q.MeetingURL, ImageURL: q.ImageURL, RegistrationURL: q.RegistrationURL}
	if q.StartAt != nil {
		e.StartAt = *q.StartAt
	}
	return e
}
func (h *Handler) listEvents(w http.ResponseWriter, r *http.Request) {
	if !h.eventReady(w) {
		return
	}
	var actor *domain.User
	if token, tokenErr := sessionToken(r); tokenErr == nil {
		current, currentErr := h.service.CurrentUser(r.Context(), token)
		if currentErr != nil {
			invalidSession(w, r, "sesión inválida o expirada")
			return
		}
		actor = &current
	} else if hasSessionCredential(r) {
		invalidSession(w, r, "sesión inválida o expirada")
		return
	}
	var from, to *time.Time
	if value := r.URL.Query().Get("from"); value != "" {
		t, err := time.Parse(time.RFC3339, value)
		if err != nil {
			writeError(w, 400, "from debe ser una fecha RFC3339")
			return
		}
		from = &t
	}
	if value := r.URL.Query().Get("to"); value != "" {
		t, err := time.Parse(time.RFC3339, value)
		if err != nil {
			writeError(w, 400, "to debe ser una fecha RFC3339")
			return
		}
		to = &t
	}
	status := r.URL.Query().Get("status")
	items, err := h.events.List(r.Context(), r.URL.Query().Get("type"), status, from, to, actor)
	if err != nil {
		eventError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": items})
}
func (h *Handler) createEvent(w http.ResponseWriter, r *http.Request) {
	if !h.eventReady(w) {
		return
	}
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	var q eventRequest
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		writeError(w, 400, "cuerpo JSON inválido")
		return
	}
	e, err := h.events.Create(r.Context(), eventFromRequest(q), actor.ID)
	if err != nil {
		eventError(w, err)
		return
	}
	writeJSON(w, 201, e)
}
func (h *Handler) getEvent(w http.ResponseWriter, r *http.Request) {
	if !h.eventReady(w) {
		return
	}
	token, tokenErr := sessionToken(r)
	actor := ""
	if token != "" {
		u, err := h.service.CurrentUser(r.Context(), token)
		if err != nil {
			invalidSession(w, r, "sesión inválida o expirada")
			return
		}
		actor = u.ID
	} else if tokenErr != nil && hasSessionCredential(r) {
		invalidSession(w, r, "sesión inválida o expirada")
		return
	}
	e, err := h.events.Find(r.Context(), r.PathValue("id"), actor)
	if err != nil {
		eventError(w, err)
		return
	}
	writeJSON(w, 200, e)
}
func (h *Handler) updateEvent(w http.ResponseWriter, r *http.Request) {
	if !h.eventReady(w) {
		return
	}
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	old, err := h.events.FindForUpdate(r.Context(), r.PathValue("id"), actor.ID)
	if err != nil {
		eventError(w, err)
		return
	}
	var q eventPatchRequest
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		writeError(w, 400, "cuerpo JSON inválido")
		return
	}
	updated := old
	if q.Title != nil {
		updated.Title = *q.Title
	}
	if q.Description != nil {
		updated.Description = *q.Description
	}
	if q.Type != nil {
		updated.Type = *q.Type
	}
	if q.StartAt != nil {
		updated.StartAt = *q.StartAt
	}
	if q.EndAt != nil {
		if string(q.EndAt) == "null" {
			updated.EndAt = nil
		} else {
			var endAt time.Time
			if json.Unmarshal(q.EndAt, &endAt) != nil {
				writeError(w, http.StatusBadRequest, "endAt debe ser una fecha RFC3339 o null")
				return
			}
			updated.EndAt = &endAt
		}
	}
	if q.Venue != nil {
		updated.Venue = *q.Venue
	}
	if q.MeetingURL != nil {
		updated.MeetingURL = *q.MeetingURL
	}
	if q.ImageURL != nil {
		updated.ImageURL = *q.ImageURL
	}
	if q.RegistrationURL != nil {
		updated.RegistrationURL = *q.RegistrationURL
	}
	e, err := h.events.Update(r.Context(), r.PathValue("id"), updated, actor.ID)
	if err != nil {
		eventError(w, err)
		return
	}
	writeJSON(w, 200, e)
}
func (h *Handler) changeEventStatus(w http.ResponseWriter, r *http.Request, status string) {
	if !h.eventReady(w) {
		return
	}
	actor, ok := h.actor(w, r)
	if !ok {
		return
	}
	e, err := h.events.SetStatus(r.Context(), r.PathValue("id"), status, actor.ID)
	if err != nil {
		eventError(w, err)
		return
	}
	writeJSON(w, 200, e)
}
func (h *Handler) publishEvent(w http.ResponseWriter, r *http.Request) {
	h.changeEventStatus(w, r, "publicado")
}
func (h *Handler) cancelEvent(w http.ResponseWriter, r *http.Request) {
	h.changeEventStatus(w, r, "cancelado")
}
func (h *Handler) hideEvent(w http.ResponseWriter, r *http.Request) {
	h.changeEventStatus(w, r, "oculto")
}

func isLocalOrigin(origin string) bool {
	return origin == "http://localhost" || strings.HasPrefix(origin, "http://localhost:") ||
		origin == "http://127.0.0.1" || strings.HasPrefix(origin, "http://127.0.0.1:") ||
		origin == "http://[::1]" || strings.HasPrefix(origin, "http://[::1]:")
}

type seedbedRequest struct {
	Name                 string                 `json:"name"`
	Description          string                 `json:"description"`
	ResearchLine         string                 `json:"researchLine"`
	Objective            string                 `json:"objective"`
	ResponsibleTeacherID string                 `json:"responsibleTeacherId"`
	ContactEmail         string                 `json:"contactEmail"`
	Topics               *[]domain.SeedbedTopic `json:"topics"`
}
type visibilityRequest struct {
	Status     string `json:"status"`
	Visibility string `json:"visibility"`
}
type seedbedMemberRequest struct {
	UserID string `json:"userId"`
}

func (h *Handler) seedbedReady(w http.ResponseWriter) bool {
	if h.seedbeds == nil {
		writeError(w, 500, "módulo de semilleros no configurado")
		return false
	}
	return true
}
func (h *Handler) actor(w http.ResponseWriter, r *http.Request) (domain.User, bool) {
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
	return u, true
}
func seedbedError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		writeError(w, 403, err.Error())
	case errors.Is(err, domain.ErrSeedbedNotFound):
		writeError(w, 404, err.Error())
	case errors.Is(err, domain.ErrMemberNotFound):
		writeError(w, 404, err.Error())
	case errors.Is(err, domain.ErrInvalidSeedbed), errors.Is(err, domain.ErrInvalidTeacher), errors.Is(err, domain.ErrInvalidSeedbedStatus):
		writeError(w, 400, err.Error())
	case errors.Is(err, domain.ErrDuplicateSeedbed):
		writeError(w, 409, err.Error())
	default:
		writeError(w, 500, "no fue posible procesar el semillero")
	}
}
func (h *Handler) listSeedbeds(w http.ResponseWriter, r *http.Request) {
	if !h.seedbedReady(w) {
		return
	}
	private := false
	if r.URL.Query().Get("includeInactive") == "true" {
		a, ok := h.actor(w, r)
		if !ok {
			return
		}
		if !strings.EqualFold(a.Role, "admin") {
			writeError(w, 403, "se requiere rol admin")
			return
		}
		private = true
	}
	items, err := h.seedbeds.List(r.Context(), r.URL.Query().Get("q"), r.URL.Query().Get("researchLine"), queryInt(r, "page", 1), queryInt(r, "pageSize", 20), private)
	if err != nil {
		seedbedError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"seedbeds": items})
}
func queryInt(r *http.Request, key string, def int) int {
	var n int
	if _, err := fmt.Sscanf(r.URL.Query().Get(key), "%d", &n); err != nil || n < 1 {
		return def
	}
	return n
}
func (h *Handler) createSeedbed(w http.ResponseWriter, r *http.Request) {
	if !h.seedbedReady(w) {
		return
	}
	a, ok := h.actor(w, r)
	if !ok {
		return
	}
	if !strings.EqualFold(a.Role, "admin") && !strings.EqualFold(a.Role, "profesor") && !strings.EqualFold(a.Role, "docente") {
		writeError(w, 403, "se requiere rol docente o admin")
		return
	}
	var q seedbedRequest
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		writeError(w, 400, "cuerpo JSON inválido")
		return
	}
	s, err := h.seedbeds.Create(r.Context(), seedbedapp.CreateCommand{
		Name: q.Name, Description: q.Description, ResearchLine: q.ResearchLine,
		Objective: q.Objective, ResponsibleTeacherID: q.ResponsibleTeacherID,
		ContactEmail: q.ContactEmail, Topics: q.Topics,
	})
	if err != nil {
		seedbedError(w, err)
		return
	}
	writeJSON(w, 201, s)
}
func (h *Handler) getSeedbed(w http.ResponseWriter, r *http.Request) {
	if !h.seedbedReady(w) {
		return
	}
	s, err := h.seedbeds.Find(r.Context(), r.PathValue("id"))
	if err != nil {
		seedbedError(w, err)
		return
	}
	if s.Status != domain.SeedbedPublished || s.Visibility != domain.SeedbedPublished {
		a, ok := h.actor(w, r)
		if !ok {
			return
		}
		if !strings.EqualFold(a.Role, "admin") && a.ID != s.ResponsibleTeacherID {
			writeError(w, 404, "semillero no encontrado")
			return
		}
	}
	writeJSON(w, 200, s)
}

func (h *Handler) getSeedbedSpace(w http.ResponseWriter, r *http.Request) {
	if !h.seedbedReady(w) {
		return
	}
	id := r.PathValue("id")
	seedbed, err := h.seedbeds.Find(r.Context(), id)
	if err != nil {
		seedbedError(w, err)
		return
	}
	public := seedbed.Status == domain.SeedbedPublished && seedbed.Visibility == domain.SeedbedPublished
	if !public {
		actor, ok := h.actor(w, r)
		if !ok {
			return
		}
		if err := h.seedbeds.CanManage(r.Context(), actor.ID, id); err != nil {
			writeError(w, http.StatusNotFound, "semillero no encontrado")
			return
		}
	}
	space, err := h.seedbeds.Space(r.Context(), id)
	if err != nil {
		seedbedError(w, err)
		return
	}
	if public {
		space.Responsible.Email = ""
	}
	writeJSON(w, http.StatusOK, space)
}
func (h *Handler) updateSeedbed(w http.ResponseWriter, r *http.Request) {
	if !h.seedbedReady(w) {
		return
	}
	a, ok := h.actor(w, r)
	if !ok {
		return
	}
	if h.seedbeds.CanManage(r.Context(), a.ID, r.PathValue("id")) != nil {
		writeError(w, 403, "no tiene permisos para esta operación")
		return
	}
	var q seedbedRequest
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		writeError(w, 400, "cuerpo JSON inválido")
		return
	}
	s, err := h.seedbeds.Update(r.Context(), r.PathValue("id"), seedbedapp.UpdateCommand{
		Name: q.Name, Description: q.Description, ResearchLine: q.ResearchLine,
		Objective: q.Objective, ResponsibleTeacherID: q.ResponsibleTeacherID,
		ContactEmail: q.ContactEmail, Topics: q.Topics,
	})
	if err != nil {
		seedbedError(w, err)
		return
	}
	writeJSON(w, 200, s)
}
func (h *Handler) setSeedbedVisibility(w http.ResponseWriter, r *http.Request) {
	a, ok := h.actor(w, r)
	if !ok || !h.seedbedReady(w) {
		return
	}
	var q visibilityRequest
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		writeError(w, 400, "cuerpo JSON inválido")
		return
	}
	status := q.Status
	if status == "" {
		status = q.Visibility
	}
	s, err := h.seedbeds.Visibility(r.Context(), a.ID, r.PathValue("id"), status)
	if err != nil {
		seedbedError(w, err)
		return
	}
	writeJSON(w, 200, s)
}
func (h *Handler) listSeedbedMembers(w http.ResponseWriter, r *http.Request) {
	if !h.seedbedReady(w) {
		return
	}
	history := r.URL.Query().Get("history") == "true"
	seedbed, err := h.seedbeds.Find(r.Context(), r.PathValue("id"))
	if err != nil {
		seedbedError(w, err)
		return
	}
	if !history && (seedbed.Status != domain.SeedbedPublished || seedbed.Visibility != domain.SeedbedPublished) {
		actor, authenticated := h.actor(w, r)
		if !authenticated {
			return
		}
		if !strings.EqualFold(actor.Role, "admin") && actor.ID != seedbed.ResponsibleTeacherID {
			writeError(w, http.StatusNotFound, "semillero no encontrado")
			return
		}
	}
	if history {
		a, ok := h.actor(w, r)
		if !ok {
			return
		}
		if !strings.EqualFold(a.Role, "admin") && seedbed.ResponsibleTeacherID != a.ID {
			writeError(w, 403, "historial restringido")
			return
		}
	}
	m, err := h.seedbeds.Members(r.Context(), r.PathValue("id"), history)
	if err != nil {
		seedbedError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"members": m})
}
func (h *Handler) addSeedbedMember(w http.ResponseWriter, r *http.Request) {
	if !h.seedbedReady(w) {
		return
	}
	a, ok := h.actor(w, r)
	if !ok {
		return
	}
	var q seedbedMemberRequest
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		writeError(w, 400, "cuerpo JSON inválido")
		return
	}
	if err := h.seedbeds.AddMember(r.Context(), a.ID, r.PathValue("id"), q.UserID); err != nil {
		seedbedError(w, err)
		return
	}
	w.WriteHeader(204)
}
func (h *Handler) removeSeedbedMember(w http.ResponseWriter, r *http.Request) {
	if !h.seedbedReady(w) {
		return
	}
	a, ok := h.actor(w, r)
	if !ok {
		return
	}
	if err := h.seedbeds.RemoveMember(r.Context(), a.ID, r.PathValue("id"), r.PathValue("userId")); err != nil {
		seedbedError(w, err)
		return
	}
	w.WriteHeader(204)
}

func (h *Handler) registrationReady(w http.ResponseWriter) bool {
	if h.registrations == nil {
		writeError(w, 500, "módulo de solicitudes no configurado")
		return false
	}
	return true
}

func (h *Handler) createRegistration(w http.ResponseWriter, r *http.Request) {
	if !h.registrationReady(w) {
		return
	}
	var request domain.RegistrationRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, 400, "cuerpo JSON inválido")
		return
	}
	userID := ""
	if hasSessionCredential(r) {
		actor, ok := h.actor(w, r)
		if !ok {
			return
		}
		userID = actor.ID
	}
	created, trackingToken, err := h.registrations.Create(r.Context(), request, userID)
	if errors.Is(err, domain.ErrIncompleteRegistration) || errors.Is(err, domain.ErrInvalidRegistrationData) {
		writeError(w, 400, err.Error())
		return
	}
	if errors.Is(err, domain.ErrSeedbedNotFound) {
		writeError(w, 400, err.Error())
		return
	}
	if err != nil {
		log.Printf("registration request create failed: %v", err)
		writeError(w, 500, "no fue posible radicar la solicitud")
		return
	}
	writeJSON(w, 201, map[string]any{"id": created.ID, "estado": created.Status, "fechaRadicacion": created.CreatedAt, "trackingToken": trackingToken})
}

func (h *Handler) registrationStatus(w http.ResponseWriter, r *http.Request) {
	if !h.registrationReady(w) {
		return
	}
	trackingToken := strings.TrimSpace(r.URL.Query().Get("trackingToken"))
	userID := ""
	if trackingToken == "" && hasSessionCredential(r) {
		actor, ok := h.actor(w, r)
		if !ok {
			return
		}
		userID = actor.ID
	}
	item, err := h.registrations.Status(r.Context(), r.PathValue("id"), trackingToken, userID)
	if errors.Is(err, domain.ErrInvalidTrackingToken) {
		writeError(w, 401, err.Error())
		return
	}
	if errors.Is(err, domain.ErrRegistrationNotFound) {
		writeError(w, 404, "solicitud no encontrada")
		return
	}
	if err != nil {
		writeError(w, 500, "no fue posible consultar la solicitud")
		return
	}
	writeJSON(w, 200, map[string]any{"id": item.ID, "estado": item.Status, "fechaActualizacion": item.UpdatedAt, "observacion": item.Observation})
}

type registrationStatusRequest struct {
	Status        string `json:"estado"`
	Observation   string `json:"observacion"`
	ResponsibleID string `json:"responsableId"`
}

func (h *Handler) adminRegistration(w http.ResponseWriter, r *http.Request) (domain.User, bool) {
	a, ok := h.actor(w, r)
	if !ok {
		return domain.User{}, false
	}
	if !strings.EqualFold(a.Role, "admin") {
		writeError(w, 403, "se requiere rol admin")
		return domain.User{}, false
	}
	return a, true
}

func (h *Handler) listRegistrationRequests(w http.ResponseWriter, r *http.Request) {
	if !h.registrationReady(w) {
		return
	}
	if _, ok := h.adminRegistration(w, r); !ok {
		return
	}
	items, err := h.registrations.List(r.Context(), r.URL.Query().Get("estado"), r.URL.Query().Get("desde"), r.URL.Query().Get("hasta"), r.URL.Query().Get("seedbedId"), r.URL.Query().Get("responsableId"))
	if err != nil {
		writeError(w, 500, "no fue posible consultar las solicitudes")
		return
	}
	writeJSON(w, 200, map[string]any{"requests": items})
}

func (h *Handler) registrationDetail(w http.ResponseWriter, r *http.Request) {
	if !h.registrationReady(w) {
		return
	}
	if _, ok := h.adminRegistration(w, r); !ok {
		return
	}
	item, err := h.registrations.Detail(r.Context(), r.PathValue("id"))
	if errors.Is(err, domain.ErrRegistrationNotFound) {
		writeError(w, 404, err.Error())
		return
	}
	if err != nil {
		writeError(w, 500, "no fue posible consultar el detalle")
		return
	}
	writeJSON(w, 200, item)
}

func (h *Handler) updateRegistrationStatus(w http.ResponseWriter, r *http.Request) {
	if !h.registrationReady(w) {
		return
	}
	actor, ok := h.adminRegistration(w, r)
	if !ok {
		return
	}
	var request registrationStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, 400, "cuerpo JSON inválido")
		return
	}
	responsible := request.ResponsibleID
	if responsible == "" {
		responsible = actor.ID
	}
	item, err := h.registrations.UpdateStatus(r.Context(), r.PathValue("id"), request.Status, request.Observation, responsible)
	switch {
	case errors.Is(err, domain.ErrInvalidRegistrationStatus), errors.Is(err, domain.ErrInvalidRegistrationTransition):
		writeError(w, 400, err.Error())
	case errors.Is(err, domain.ErrRegistrationNotFound):
		writeError(w, 404, err.Error())
	case err != nil:
		writeError(w, 500, "no fue posible actualizar la solicitud")
	default:
		writeJSON(w, 200, item)
	}
}

func (h *Handler) researchReady(w http.ResponseWriter) bool {
	if h.research == nil {
		writeError(w, 500, "módulo de investigación no configurado")
		return false
	}
	return true
}

type activityRequest struct {
	Title          string     `json:"title"`
	Description    string     `json:"description"`
	Status         string     `json:"status"`
	Date           *time.Time `json:"date"`
	ResponsibleIDs []string   `json:"responsibleIds"`
	Evidence       []string   `json:"evidence"`
}
type projectRequest struct {
	SeedbedID                            string `json:"seedbedId"`
	Title, Summary, ResearchLine, Status string
	ResponsibleIDs, MemberIDs            []string
	StartDate, EndDate                   *time.Time
	Progress                             int
}
type proposalRequest struct {
	SeedbedID, Title, Problem, Objectives, Methodology, Status, Observations string
	ResponsibleIDs, Attachments                                              []string
}

func actorRole(a domain.User, roles ...string) bool {
	for _, x := range roles {
		if strings.EqualFold(a.Role, x) {
			return true
		}
	}
	return false
}
func researchError(w http.ResponseWriter, e error) {
	switch {
	case errors.Is(e, domain.ErrForbidden):
		writeError(w, 403, e.Error())
	case errors.Is(e, domain.ErrResearchNotFound), errors.Is(e, domain.ErrSeedbedNotFound):
		writeError(w, 404, mapResearchNotFound(e))
	case errors.Is(e, domain.ErrInvalidResearch), errors.Is(e, domain.ErrInvalidProjectStatus):
		writeError(w, 400, e.Error())
	default:
		writeError(w, 500, "no fue posible procesar la investigación")
	}
}
func mapResearchNotFound(e error) string {
	if errors.Is(e, domain.ErrSeedbedNotFound) {
		return "semillero no encontrado"
	}
	return e.Error()
}
func (h *Handler) listActivities(w http.ResponseWriter, r *http.Request) {
	if !h.researchReady(w) {
		return
	}
	v, e := h.research.ListActivities(r.Context(), r.PathValue("seedbedId"), r.URL.Query().Get("period"), r.URL.Query().Get("status"))
	if e != nil {
		researchError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"activities": v})
}
func (h *Handler) createActivity(w http.ResponseWriter, r *http.Request) {
	if !h.researchReady(w) {
		return
	}
	a, ok := h.actor(w, r)
	if !ok {
		return
	}
	var q activityRequest
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		writeError(w, 400, "cuerpo JSON inválido")
		return
	}
	v, e := h.research.CreateActivity(r.Context(), domain.Activity{SeedbedID: r.PathValue("seedbedId"), Title: q.Title, Description: q.Description, Status: q.Status, Date: q.Date, ResponsibleIDs: q.ResponsibleIDs, Evidence: q.Evidence}, a.ID)
	if e != nil {
		researchError(w, e)
		return
	}
	writeJSON(w, 201, v)
}
func (h *Handler) getActivity(w http.ResponseWriter, r *http.Request) {
	if !h.researchReady(w) {
		return
	}
	v, e := h.research.FindActivity(r.Context(), r.PathValue("id"))
	if e != nil {
		researchError(w, e)
		return
	}
	writeJSON(w, 200, v)
}

// PATCH handlers decode the body over the stored record so omitted fields keep their value.
func (h *Handler) updateActivity(w http.ResponseWriter, r *http.Request) {
	if !h.researchReady(w) {
		return
	}
	a, ok := h.actor(w, r)
	if !ok {
		return
	}
	q, e := h.research.FindActivity(r.Context(), r.PathValue("id"))
	if e != nil {
		researchError(w, e)
		return
	}
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		writeError(w, 400, "cuerpo JSON inválido")
		return
	}
	v, e := h.research.UpdateActivity(r.Context(), r.PathValue("id"), q, a.ID)
	if e != nil {
		researchError(w, e)
		return
	}
	writeJSON(w, 200, v)
}
func toProject(q projectRequest) domain.Project {
	return domain.Project{SeedbedID: q.SeedbedID, Title: q.Title, Summary: q.Summary, ResearchLine: q.ResearchLine, Status: q.Status, ResponsibleIDs: q.ResponsibleIDs, MemberIDs: q.MemberIDs, StartDate: q.StartDate, EndDate: q.EndDate, Progress: q.Progress}
}
func (h *Handler) listProjects(w http.ResponseWriter, r *http.Request) {
	if !h.researchReady(w) {
		return
	}
	v, e := h.research.ListProjects(r.Context(), r.URL.Query().Get("seedbedId"), r.URL.Query().Get("status"), r.URL.Query().Get("period"))
	if e != nil {
		researchError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"projects": v})
}
func (h *Handler) createProject(w http.ResponseWriter, r *http.Request) {
	if !h.researchReady(w) {
		return
	}
	a, ok := h.actor(w, r)
	if !ok {
		return
	}
	var q projectRequest
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		writeError(w, 400, "cuerpo JSON inválido")
		return
	}
	v, e := h.research.CreateProject(r.Context(), toProject(q), a.ID)
	if e != nil {
		researchError(w, e)
		return
	}
	writeJSON(w, 201, v)
}
func (h *Handler) getProject(w http.ResponseWriter, r *http.Request) {
	if !h.researchReady(w) {
		return
	}
	v, e := h.research.FindProject(r.Context(), r.PathValue("id"))
	if e != nil {
		researchError(w, e)
		return
	}
	writeJSON(w, 200, v)
}
func (h *Handler) updateProject(w http.ResponseWriter, r *http.Request) {
	if !h.researchReady(w) {
		return
	}
	a, ok := h.actor(w, r)
	if !ok {
		return
	}
	q, e := h.research.FindProject(r.Context(), r.PathValue("id"))
	if e != nil {
		researchError(w, e)
		return
	}
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		writeError(w, 400, "cuerpo JSON inválido")
		return
	}
	v, e := h.research.UpdateProject(r.Context(), r.PathValue("id"), q, a.ID)
	if e != nil {
		researchError(w, e)
		return
	}
	writeJSON(w, 200, v)
}
func (h *Handler) updateProgress(w http.ResponseWriter, r *http.Request) {
	if !h.researchReady(w) {
		return
	}
	a, ok := h.actor(w, r)
	if !ok {
		return
	}
	var q struct {
		Progress int `json:"progress"`
	}
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		writeError(w, 400, "cuerpo JSON inválido")
		return
	}
	v, e := h.research.Progress(r.Context(), r.PathValue("id"), q.Progress, a.ID)
	if e != nil {
		researchError(w, e)
		return
	}
	writeJSON(w, 200, v)
}
func (h *Handler) projectProgress(w http.ResponseWriter, r *http.Request) {
	if !h.researchReady(w) {
		return
	}
	a, ok := h.actor(w, r)
	if !ok || !actorRole(a, "estudiante", "docente", "profesor", "admin") {
		if ok {
			writeError(w, 403, "rol no autorizado")
		}
		return
	}
	v, e := h.research.ProgressList(r.Context(), r.URL.Query().Get("seedbedId"), r.URL.Query().Get("period"))
	if e != nil {
		researchError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"projects": v})
}
func (h *Handler) projectHistory(w http.ResponseWriter, r *http.Request) {
	if !h.researchReady(w) {
		return
	}
	a, ok := h.actor(w, r)
	if !ok || !actorRole(a, "estudiante", "docente", "profesor", "admin") {
		if ok {
			writeError(w, 403, "rol no autorizado")
		}
		return
	}
	v, e := h.research.History(r.Context(), r.PathValue("id"))
	if e != nil {
		researchError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"history": v})
}
func (h *Handler) listProposals(w http.ResponseWriter, r *http.Request) {
	if !h.researchReady(w) {
		return
	}
	a, ok := h.actor(w, r)
	if !ok {
		return
	}
	v, e := h.research.Proposals(r.Context(), a.ID)
	if e != nil {
		researchError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"proposals": v})
}
func (h *Handler) createProposal(w http.ResponseWriter, r *http.Request) {
	if !h.researchReady(w) {
		return
	}
	a, ok := h.actor(w, r)
	if !ok {
		return
	}
	var q proposalRequest
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		writeError(w, 400, "cuerpo JSON inválido")
		return
	}
	v, e := h.research.CreateProposal(r.Context(), domain.ResearchProposal{SeedbedID: q.SeedbedID, Title: q.Title, Problem: q.Problem, Objectives: q.Objectives, Methodology: q.Methodology, Status: q.Status, ResponsibleIDs: q.ResponsibleIDs, Attachments: q.Attachments}, a.ID)
	if e != nil {
		researchError(w, e)
		return
	}
	writeJSON(w, 201, v)
}
func (h *Handler) getProposal(w http.ResponseWriter, r *http.Request) {
	if !h.researchReady(w) {
		return
	}
	a, ok := h.actor(w, r)
	if !ok {
		return
	}
	v, e := h.research.Proposal(r.Context(), r.PathValue("id"), a.ID)
	if e != nil {
		researchError(w, e)
		return
	}
	writeJSON(w, 200, v)
}
func (h *Handler) updateProposal(w http.ResponseWriter, r *http.Request) {
	if !h.researchReady(w) {
		return
	}
	a, ok := h.actor(w, r)
	if !ok {
		return
	}
	q, e := h.research.Proposal(r.Context(), r.PathValue("id"), a.ID)
	if e != nil {
		researchError(w, e)
		return
	}
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		writeError(w, 400, "cuerpo JSON inválido")
		return
	}
	v, e := h.research.UpdateProposal(r.Context(), r.PathValue("id"), q, a.ID)
	if e != nil {
		researchError(w, e)
		return
	}
	writeJSON(w, 200, v)
}
func (h *Handler) convertProposal(w http.ResponseWriter, r *http.Request) {
	if !h.researchReady(w) {
		return
	}
	a, ok := h.actor(w, r)
	if !ok {
		return
	}
	v, e := h.research.Convert(r.Context(), r.PathValue("id"), a.ID)
	if e != nil {
		researchError(w, e)
		return
	}
	writeJSON(w, 201, v)
}
func (h *Handler) createReview(w http.ResponseWriter, r *http.Request) {
	if !h.researchReady(w) {
		return
	}
	a, ok := h.actor(w, r)
	if !ok {
		return
	}
	var q struct {
		Text string `json:"text"`
	}
	if json.NewDecoder(r.Body).Decode(&q) != nil || strings.TrimSpace(q.Text) == "" {
		writeError(w, 400, "reseña inválida")
		return
	}
	v, e := h.research.Review(r.Context(), domain.SeedbedReview{SeedbedID: r.PathValue("seedbedId"), AuthorID: a.ID, Text: q.Text})
	if e != nil {
		researchError(w, e)
		return
	}
	writeJSON(w, 201, v)
}
func (h *Handler) listReviews(w http.ResponseWriter, r *http.Request) {
	if !h.researchReady(w) {
		return
	}
	if _, ok := h.adminRegistration(w, r); !ok {
		return
	}
	v, e := h.research.Reviews(r.Context(), r.URL.Query().Get("status"))
	if e != nil {
		researchError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"reviews": v})
}
func (h *Handler) moderateReview(w http.ResponseWriter, r *http.Request) {
	if !h.researchReady(w) {
		return
	}
	if _, ok := h.adminRegistration(w, r); !ok {
		return
	}
	var q struct{ Status, Note string }
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		writeError(w, 400, "cuerpo JSON inválido")
		return
	}
	v, e := h.research.Moderate(r.Context(), r.PathValue("id"), q.Status, q.Note)
	if e != nil {
		researchError(w, e)
		return
	}
	writeJSON(w, 200, v)
}

func hasSessionCredential(r *http.Request) bool {
	if strings.TrimSpace(r.Header.Get("Authorization")) != "" {
		return true
	}
	cookie, err := r.Cookie(sessionCookieName)
	return err == nil && cookie.Value != ""
}

type roleRequest struct {
	Role string `json:"role"`
	Rol  string `json:"rol"`
}

type memberRequest struct {
	FirebaseUID     string   `json:"firebase_uid"`
	Name            string   `json:"name"`
	LastName        string   `json:"last_name"`
	Email           string   `json:"email"`
	DocumentType    string   `json:"document_type"`
	DocumentNumber  string   `json:"document_number"`
	Role            string   `json:"role"`
	Area            string   `json:"area"`
	Formation       string   `json:"formation"`
	ORCID           string   `json:"orcid"`
	CvLAC           string   `json:"cvlac"`
	VisiblePublicly *bool    `json:"visible_publicly"`
	SeedbedIDs      []string `json:"seedbed_ids"`
	Status          string   `json:"status"`
}

func (h *Handler) memberReady(w http.ResponseWriter) bool {
	if h.members == nil {
		writeError(w, 500, "módulo de integrantes no configurado")
		return false
	}
	return true
}
func (h *Handler) requireAdmin(w http.ResponseWriter, r *http.Request) (domain.User, bool) {
	a, ok := h.actor(w, r)
	if !ok {
		return domain.User{}, false
	}
	if !strings.EqualFold(a.Role, "admin") {
		writeError(w, 403, "se requiere rol admin")
		return domain.User{}, false
	}
	return a, true
}
func memberFilter(r *http.Request) memberapp.Filter {
	return memberapp.Filter{Query: searchQuery(r), Role: r.URL.Query().Get("type"), SeedbedID: r.URL.Query().Get("seedbedId"), Area: r.URL.Query().Get("area"), Limit: requestedLimit(r)}
}
func (h *Handler) publicMembers(w http.ResponseWriter, r *http.Request) {
	// Keep the lightweight auth-only handler usable by existing callers that do
	// not compose the complete member module (notably isolated auth tests).
	if h.members == nil {
		users, err := h.service.SearchUsers(r.Context(), searchQuery(r), requestedLimit(r))
		if errors.Is(err, domain.ErrIncompleteIdentity) {
			writeError(w, http.StatusBadRequest, "se requiere una búsqueda por nombre o correo")
			return
		}
		if err != nil {
			writeError(w, http.StatusInternalServerError, "no fue posible consultar los integrantes")
			return
		}
		profiles := make([]domain.PublicProfile, 0, len(users))
		for _, user := range users {
			profiles = append(profiles, domain.PublicProfile{ID: user.ID, Name: user.Name, LastName: user.LastName, Role: user.Role})
		}
		writeJSON(w, http.StatusOK, map[string]any{"members": profiles})
		return
	}
	if !h.memberReady(w) {
		return
	}
	items, err := h.members.List(r.Context(), memberFilter(r))
	if err != nil {
		writeError(w, 500, "no fue posible consultar el directorio de integrantes")
		return
	}
	writeJSON(w, 200, map[string]any{"members": items})
}
func (h *Handler) createMember(w http.ResponseWriter, r *http.Request) {
	if !h.memberReady(w) {
		return
	}
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	var q memberRequest
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		writeError(w, 400, "cuerpo JSON inválido")
		return
	}
	p, err := h.members.Create(r.Context(), memberapp.CreateCommand{FirebaseUID: q.FirebaseUID, Name: q.Name, LastName: q.LastName, Email: q.Email, DocumentType: q.DocumentType, DocumentNumber: q.DocumentNumber, Role: q.Role, Area: q.Area, Formation: q.Formation, ORCID: q.ORCID, CvLAC: q.CvLAC, VisiblePublicly: q.VisiblePublicly == nil || *q.VisiblePublicly, SeedbedIDs: q.SeedbedIDs})
	if errors.Is(err, domain.ErrInvalidMember) {
		writeError(w, 400, err.Error())
		return
	}
	if err != nil {
		writeError(w, 500, "no fue posible crear el perfil del integrante")
		return
	}
	writeJSON(w, 201, p)
}
func (h *Handler) getMember(w http.ResponseWriter, r *http.Request) {
	if !h.memberReady(w) {
		return
	}
	id := r.PathValue("id")
	private := false
	if hasSessionCredential(r) {
		a, ok := h.actor(w, r)
		if !ok {
			return
		}
		p, err := h.members.Get(r.Context(), id, true)
		if err != nil {
			writeError(w, 404, "integrante no encontrado")
			return
		}
		private = strings.EqualFold(a.Role, "admin") || a.ID == p.ID
	}
	p, err := h.members.Get(r.Context(), id, private)
	if errors.Is(err, domain.ErrMemberNotFound) {
		writeError(w, 404, err.Error())
		return
	}
	if err != nil {
		writeError(w, 500, "no fue posible consultar el perfil")
		return
	}
	if p.Status != "activo" && !private {
		writeError(w, 404, "perfil no encontrado")
		return
	}
	writeJSON(w, 200, p)
}
func (h *Handler) updateMember(w http.ResponseWriter, r *http.Request) {
	if !h.memberReady(w) {
		return
	}
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	var q memberRequest
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		writeError(w, 400, "cuerpo JSON inválido")
		return
	}
	p, err := h.members.Update(r.Context(), r.PathValue("id"), memberapp.UpdateCommand{Name: q.Name, LastName: q.LastName, Email: q.Email, DocumentType: q.DocumentType, DocumentNumber: q.DocumentNumber, Role: q.Role, Area: q.Area, Formation: q.Formation, ORCID: q.ORCID, CvLAC: q.CvLAC, VisiblePublicly: q.VisiblePublicly, SeedbedIDs: q.SeedbedIDs})
	if errors.Is(err, domain.ErrInvalidMember) {
		writeError(w, 400, err.Error())
		return
	}
	if errors.Is(err, domain.ErrMemberNotFound) {
		writeError(w, 404, err.Error())
		return
	}
	if err != nil {
		writeError(w, 500, "no fue posible actualizar el perfil del integrante")
		return
	}
	writeJSON(w, 200, p)
}
func (h *Handler) setMemberStatus(w http.ResponseWriter, r *http.Request) {
	if !h.memberReady(w) {
		return
	}
	if _, ok := h.requireAdmin(w, r); !ok {
		return
	}
	var q struct {
		Status string `json:"status"`
	}
	if json.NewDecoder(r.Body).Decode(&q) != nil {
		writeError(w, 400, "cuerpo JSON inválido")
		return
	}
	p, err := h.members.SetStatus(r.Context(), r.PathValue("id"), q.Status)
	if errors.Is(err, domain.ErrInvalidMemberStatus) {
		writeError(w, 400, err.Error())
		return
	}
	if errors.Is(err, domain.ErrMemberNotFound) {
		writeError(w, 404, err.Error())
		return
	}
	if err != nil {
		writeError(w, 500, "no fue posible actualizar el estado del perfil")
		return
	}
	writeJSON(w, 200, p)
}
func (h *Handler) memberPublications(w http.ResponseWriter, r *http.Request) {
	if !h.memberReady(w) {
		return
	}
	p, err := h.members.Get(r.Context(), r.PathValue("id"), false)
	if errors.Is(err, domain.ErrMemberNotFound) || err != nil && p.ID == "" {
		writeError(w, 404, "integrante no encontrado")
		return
	}
	if p.Status != "activo" || !p.VisiblePublicly {
		writeError(w, 404, "perfil no encontrado")
		return
	}
	items, err := h.members.Publications(r.Context(), p.ID)
	if err != nil {
		writeError(w, 500, "no fue posible consultar las publicaciones del integrante")
		return
	}
	writeJSON(w, 200, map[string]any{"publications": items})
}

func (h *Handler) adminMembers(w http.ResponseWriter, r *http.Request) {
	token, err := sessionToken(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "sesión requerida")
		return
	}
	users, err := h.service.SearchAdminUsers(r.Context(), token, searchQuery(r), requestedLimit(r))
	if errors.Is(err, domain.ErrInvalidSession) {
		writeError(w, http.StatusUnauthorized, "sesión inválida o expirada")
		return
	}
	if errors.Is(err, domain.ErrForbidden) {
		writeError(w, http.StatusForbidden, "se requiere rol admin")
		return
	}
	if errors.Is(err, domain.ErrIncompleteIdentity) {
		writeError(w, http.StatusBadRequest, "se requiere una búsqueda por nombre o correo")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no fue posible consultar los usuarios")
		return
	}
	profiles := make([]domain.AdminProfile, 0, len(users))
	for _, user := range users {
		profiles = append(profiles, domain.AdminProfile{ID: user.ID, FirebaseUID: user.FirebaseUID, Name: user.Name, LastName: user.LastName, Email: user.Email, DocumentType: user.DocumentType, DocumentNumber: user.DocumentNumber, Role: user.Role, Status: user.Status})
	}
	writeJSON(w, http.StatusOK, map[string]any{"users": profiles})
}

func (h *Handler) updateRole(w http.ResponseWriter, r *http.Request) {
	token, err := sessionToken(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "sesión requerida")
		return
	}
	targetUID := strings.TrimSpace(r.Header.Get("X-User-UID"))
	pathUID := strings.TrimSpace(r.PathValue("userId"))
	if targetUID != "" && pathUID != "" && targetUID != pathUID {
		writeError(w, http.StatusBadRequest, "el UID del header no coincide con el recurso")
		return
	}
	if targetUID == "" {
		targetUID = pathUID
	}
	var request roleRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo JSON inválido")
		return
	}
	role := request.Role
	if role == "" {
		role = request.Rol
	}
	err = h.service.UpdateUserRole(r.Context(), token, targetUID, role)
	switch {
	case errors.Is(err, domain.ErrInvalidSession):
		writeError(w, http.StatusUnauthorized, "sesión inválida o expirada")
	case errors.Is(err, domain.ErrForbidden):
		writeError(w, http.StatusForbidden, "se requiere rol admin")
	case errors.Is(err, domain.ErrSelfRoleChange):
		writeError(w, http.StatusConflict, "no puede cambiar su propio rol")
	case errors.Is(err, domain.ErrInvalidRole), errors.Is(err, domain.ErrIncompleteIdentity):
		writeError(w, http.StatusBadRequest, err.Error())
	case err != nil:
		writeError(w, http.StatusNotFound, "usuario no encontrado")
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

func requestedLimit(r *http.Request) int {
	limit := 5
	if value := r.URL.Query().Get("limit"); value != "" {
		if _, err := fmt.Sscanf(value, "%d", &limit); err != nil {
			limit = 5
		}
	}
	return limit
}

func searchQuery(r *http.Request) string {
	for _, key := range []string{"q", "nombre", "name", "correo", "email"} {
		if value := r.URL.Query().Get(key); value != "" {
			return value
		}
	}
	return ""
}

func (h *Handler) createSession(w http.ResponseWriter, r *http.Request) {
	var request sessionRequest
	// The Authorization header is authoritative. Do not decode the body first:
	// clients using the header are allowed to send no body at all.
	if authorization := strings.TrimSpace(r.Header.Get("Authorization")); authorization != "" {
		var err error
		request.Token, err = bearerToken(authorization)
		if err != nil {
			writeError(w, http.StatusUnauthorized, err.Error())
			return
		}
	} else {
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			if err == io.EOF {
				writeError(w, http.StatusUnauthorized, "se requiere un token de sesión")
				return
			}
			writeError(w, http.StatusBadRequest, "cuerpo JSON inválido")
			return
		}
	}
	result, err := h.service.Login(r.Context(), request.Token)
	if errors.Is(err, domain.ErrUserNotRegistered) {
		writeJSON(w, http.StatusForbidden, map[string]string{
			"code":   "REGISTRATION_REQUIRED",
			"error":  "registro incompleto",
			"action": "complete_registration",
		})
		return
	}
	if errors.Is(err, domain.ErrInvalidCredentials) || errors.Is(err, domain.ErrInactiveUser) {
		writeError(w, http.StatusUnauthorized, "credenciales inválidas")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no fue posible iniciar sesión")
		return
	}
	// Rotate: the browser's previous session stops working as soon as a new one is issued.
	if previous, err := r.Cookie(sessionCookieName); err == nil && previous.Value != "" && previous.Value != result.Token {
		_ = h.service.Logout(r.Context(), previous.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: result.Token, Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: 86400})
	writeJSON(w, http.StatusCreated, map[string]any{"user": result.User})
}

func (h *Handler) deleteSession(w http.ResponseWriter, r *http.Request) {
	tokens := sessionTokens(r)
	if len(tokens) == 0 {
		writeError(w, http.StatusUnauthorized, "sesión requerida")
		return
	}
	// Revoke every credential the request presents. A stale Authorization header
	// must not make logout revoke the wrong token and leave the cookie session alive.
	for _, token := range tokens {
		if err := h.service.Logout(r.Context(), token); err != nil {
			writeError(w, http.StatusUnauthorized, "sesión inválida")
			return
		}
	}
	clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) currentUser(w http.ResponseWriter, r *http.Request) {
	token, err := sessionToken(r)
	if err != nil {
		writeError(w, http.StatusUnauthorized, "sesión requerida")
		return
	}
	user, err := h.service.CurrentUser(r.Context(), token)
	if err != nil {
		invalidSession(w, r, "sesión inválida o expirada")
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *Handler) register(w http.ResponseWriter, r *http.Request) {
	token, err := bearerToken(r.Header.Get("Authorization"))
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	var request registerRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "cuerpo JSON inválido")
		return
	}
	result, err := h.service.Register(r.Context(), token, application.RegisterCommand{
		Name: request.Nombre, LastName: request.Apellido,
		DocumentType: request.TipoDNI, DocumentNumber: request.DNI,
		Email: request.Email, StudentCode: request.Codigo,
		BirthDate: request.Fecha, Phone: request.Telefono,
	})
	if errors.Is(err, domain.ErrIncompleteIdentity) || errors.Is(err, domain.ErrInvalidSignup) {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if errors.Is(err, domain.ErrDuplicateIdentity) {
		writeError(w, http.StatusConflict, err.Error())
		return
	}
	if errors.Is(err, domain.ErrInvalidIdentity) {
		writeError(w, http.StatusUnauthorized, "token de Firebase inválido")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "no fue posible guardar el usuario")
		return
	}
	writeJSON(w, http.StatusCreated, result)
}

func (h *Handler) health(w http.ResponseWriter, r *http.Request) {
	if err := h.db.Ping(r.Context()); err != nil {
		writeError(w, http.StatusServiceUnavailable, "base de datos no disponible")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func bearerToken(header string) (string, error) {
	parts := strings.Fields(header)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
		return "", errors.New("se requiere Authorization: Bearer <token>")
	}
	return parts[1], nil
}

const sessionCookieName = "galash_session"

func sessionToken(r *http.Request) (string, error) {
	// Keep the authentication source consistent with POST /auth/sessions:
	// when a client sends Authorization, it must not be shadowed by an older
	// session cookie attached by the browser.
	if authorization := strings.TrimSpace(r.Header.Get("Authorization")); authorization != "" {
		return bearerToken(authorization)
	}
	if cookie, err := r.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
		return cookie.Value, nil
	}
	return "", errors.New("se requiere una sesión")
}

// sessionTokens returns every well-formed session credential on the request.
func sessionTokens(r *http.Request) []string {
	var tokens []string
	if cookie, err := r.Cookie(sessionCookieName); err == nil && cookie.Value != "" {
		tokens = append(tokens, cookie.Value)
	}
	if authorization := strings.TrimSpace(r.Header.Get("Authorization")); authorization != "" {
		if token, err := bearerToken(authorization); err == nil {
			tokens = append(tokens, token)
		}
	}
	return tokens
}

// invalidSession answers 401 and expires the browser cookie when the rejected
// credential came from it, so stale cookies are not replayed on later requests.
func invalidSession(w http.ResponseWriter, r *http.Request, message string) {
	if strings.TrimSpace(r.Header.Get("Authorization")) == "" {
		clearSessionCookie(w)
	}
	writeError(w, http.StatusUnauthorized, message)
}

func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", HttpOnly: true, Secure: true, SameSite: http.SameSiteLaxMode, MaxAge: -1})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
