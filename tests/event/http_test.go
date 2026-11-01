package event_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	httpadapter "github.com/galash-uptc/backend/internal/adapters/inbound/http"
	authapp "github.com/galash-uptc/backend/internal/application/auth"
	eventapp "github.com/galash-uptc/backend/internal/application/event"
	"github.com/galash-uptc/backend/internal/domain"
)

type healthChecker struct{}

func (healthChecker) Ping(context.Context) error { return nil }

// verifier maps each Firebase token to the user with the same ID.
type verifier struct{}

func (verifier) Verify(_ context.Context, token string) (domain.Identity, error) {
	return domain.Identity{UID: token, Email: token + "@example.com"}, nil
}

type users map[string]domain.User

func (u users) FindByID(_ context.Context, id string) (domain.User, error) {
	user, ok := u[id]
	if !ok {
		return domain.User{}, errors.New("user not found")
	}
	return user, nil
}
func (u users) FindByEmail(context.Context, string) (domain.User, error) {
	return domain.User{}, errors.New("not used")
}
func (u users) FindByFirebaseUID(ctx context.Context, uid string) (domain.User, error) {
	return u.FindByID(ctx, uid)
}
func (u users) SetPassword(context.Context, string, string) error                { return nil }
func (u users) Search(context.Context, string, int, bool) ([]domain.User, error) { return nil, nil }
func (u users) UpdateRole(context.Context, string, string) error                 { return nil }

type authStore struct {
	users
	sessions map[string]string
}

func (a *authStore) Upsert(context.Context, domain.User) (string, error) { return "", nil }
func (a *authStore) Create(_ context.Context, userID, hash string) error {
	a.sessions[hash] = userID
	return nil
}
func (a *authStore) FindUserID(_ context.Context, hash string) (string, error) {
	if id, ok := a.sessions[hash]; ok {
		return id, nil
	}
	return "", errors.New("session not found")
}
func (a *authStore) Revoke(context.Context, string) error           { return nil }
func (a *authStore) RevokeAllForUser(context.Context, string) error { return nil }
func (a *authStore) DeleteExpiredSessions(context.Context) error    { return nil }

// repository mirrors the PostgreSQL adapter filters and conflict rule in memory.
type repository struct {
	events map[string]domain.Event
	next   int
}

func (r *repository) List(_ context.Context, typ, status string, from, to *time.Time) ([]domain.Event, error) {
	out := []domain.Event{}
	for i := 1; i <= r.next; i++ {
		e, ok := r.events[fmt.Sprint(i)]
		if !ok || typ != "" && e.Type != typ || status != "" && e.Status != status ||
			from != nil && e.StartAt.Before(*from) || to != nil && e.StartAt.After(*to) {
			continue
		}
		out = append(out, e)
	}
	return out, nil
}
func (r *repository) Create(_ context.Context, e domain.Event) (domain.Event, error) {
	r.next++
	e.ID, e.CreatedAt, e.UpdatedAt = fmt.Sprint(r.next), time.Now(), time.Now()
	r.events[e.ID] = e
	return e, nil
}
func (r *repository) Find(_ context.Context, id string) (domain.Event, error) {
	e, ok := r.events[id]
	if !ok {
		return e, domain.ErrEventNotFound
	}
	return e, nil
}
func (r *repository) Update(_ context.Context, id string, e domain.Event) (domain.Event, error) {
	e.CreatedAt, e.UpdatedAt = r.events[id].CreatedAt, time.Now()
	r.events[id] = e
	return e, nil
}
func (r *repository) SetStatus(_ context.Context, id, status string) (domain.Event, error) {
	e := r.events[id]
	e.Status, e.UpdatedAt = status, time.Now()
	r.events[id] = e
	return e, nil
}
func (r *repository) HasConflict(_ context.Context, e domain.Event, excludeID string) (bool, error) {
	end := func(x domain.Event) time.Time {
		if x.EndAt != nil {
			return *x.EndAt
		}
		return x.StartAt.Truncate(24 * time.Hour).Add(24 * time.Hour)
	}
	start := func(x domain.Event) time.Time {
		return x.StartAt
	}
	for id, o := range r.events {
		venue := strings.TrimSpace(o.Venue)
		if id != excludeID && o.Status != "cancelado" && venue != "" && venue == strings.TrimSpace(e.Venue) &&
			start(o).Before(end(e)) && start(e).Before(end(o)) {
			return true, nil
		}
	}
	return false, nil
}

type server struct {
	handler http.Handler
	auth    *authapp.Service
	repo    *repository
}

func newServer(t *testing.T) server {
	t.Helper()
	u := users{
		"admin":   {ID: "admin", Role: "admin", Status: "activo"},
		"teacher": {ID: "teacher", Role: "docente", Status: "activo"},
		"other":   {ID: "other", Role: "profesor", Status: "activo"},
		"student": {ID: "student", Role: "estudiante", Status: "activo"},
	}
	repo := &repository{events: map[string]domain.Event{}}
	auth := authapp.NewService(verifier{}, &authStore{users: u, sessions: map[string]string{}})
	handler := httpadapter.NewHandlerWithEvents(auth, healthChecker{}, nil, nil, nil, nil, eventapp.NewService(repo, u)).Routes()
	return server{handler: handler, auth: auth, repo: repo}
}

// do sends a request as userID; an empty userID sends it without a session.
func (s server) do(t *testing.T, method, path, userID, body string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	if userID != "" {
		login, err := s.auth.Login(context.Background(), userID)
		if err != nil {
			t.Fatalf("login %s: %v", userID, err)
		}
		request.Header.Set("Authorization", "Bearer "+login.Token)
	}
	response := httptest.NewRecorder()
	s.handler.ServeHTTP(response, request)
	return response
}

func expect(t *testing.T, name string, response *httptest.ResponseRecorder, status int) domain.Event {
	t.Helper()
	if response.Code != status {
		t.Errorf("%s: status=%d, want %d: %s", name, response.Code, status, response.Body.String())
	}
	var e domain.Event
	_ = json.Unmarshal(response.Body.Bytes(), &e)
	return e
}

func listIDs(t *testing.T, s server, path, userID string) string {
	t.Helper()
	response := s.do(t, http.MethodGet, path, userID, "")
	expect(t, "list "+path, response, http.StatusOK)
	var body struct{ Events []domain.Event }
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	ids := []string{}
	for _, e := range body.Events {
		ids = append(ids, e.ID)
	}
	return strings.Join(ids, ",")
}

const (
	inPerson = `{"title":"Taller IA","description":"d","type":"taller","startAt":"2030-05-10T14:00:00Z","endAt":"2030-05-10T16:00:00Z","venue":"Aula 101","imageUrl":"https://example.com/i.png"}`
	virtual  = `{"title":"Charla","type":"evento","startAt":"2030-05-10T14:00:00Z","meetingUrl":"https://meet.example.com/x","registrationUrl":"https://example.com/inscripcion"}`
)

// PT-EVT-01: create in-person and virtual events, invalid dates and URLs, draft state.
func TestCreateEvent(t *testing.T) {
	s := newServer(t)
	expect(t, "anonymous", s.do(t, http.MethodPost, "/events", "", inPerson), http.StatusUnauthorized)
	expect(t, "student", s.do(t, http.MethodPost, "/events", "student", inPerson), http.StatusForbidden)

	e := expect(t, "in-person", s.do(t, http.MethodPost, "/events", "teacher", inPerson), http.StatusCreated)
	if e.Status != "borrador" || e.OrganizerID != "teacher" || e.ImageURL != "https://example.com/i.png" || e.Venue != "Aula 101" {
		t.Errorf("in-person event = %+v", e)
	}
	e = expect(t, "virtual", s.do(t, http.MethodPost, "/events", "admin", virtual), http.StatusCreated)
	if e.Status != "borrador" || e.MeetingURL == "" || e.EndAt != nil {
		t.Errorf("virtual event = %+v", e)
	}

	for name, body := range map[string]string{
		"bad date format":  `{"title":"x","type":"evento","startAt":"10/05/2030 14:00"}`,
		"impossible date":  `{"title":"x","type":"evento","startAt":"2030-13-40T25:00:00Z"}`,
		"missing startAt":  `{"title":"x","type":"evento"}`,
		"end before start": `{"title":"x","type":"evento","startAt":"2030-05-10T14:00:00Z","endAt":"2030-05-10T13:00:00Z"}`,
		"missing title":    `{"type":"evento","startAt":"2030-05-10T14:00:00Z"}`,
		"bad image url":    `{"title":"x","type":"evento","startAt":"2030-05-10T14:00:00Z","imageUrl":"ftp://example.com/i.png"}`,
		"bad meeting url":  `{"title":"x","type":"evento","startAt":"2030-05-10T14:00:00Z","meetingUrl":"javascript:alert(1)"}`,
		"invalid json":     `{`,
	} {
		expect(t, name, s.do(t, http.MethodPost, "/events", "teacher", body), http.StatusBadRequest)
	}

	overlap := `{"title":"Otro","type":"evento","startAt":"2030-05-10T15:00:00Z","venue":" Aula 101 "}`
	expect(t, "venue overlap", s.do(t, http.MethodPost, "/events", "other", overlap), http.StatusConflict)
	adjacent := `{"title":"Después","type":"evento","startAt":"2030-05-10T16:00:00Z","endAt":"2030-05-10T18:00:00Z","venue":"Aula 101"}`
	expect(t, "adjacent slot", s.do(t, http.MethodPost, "/events", "other", adjacent), http.StatusCreated)
}

// PT-EVT-02: publish, edit, hide and cancel as owner/admin and as users without permission.
func TestEventLifecycleAndPermissions(t *testing.T) {
	s := newServer(t)
	e := expect(t, "create", s.do(t, http.MethodPost, "/events", "teacher", inPerson), http.StatusCreated)
	path := "/events/" + e.ID

	expect(t, "anonymous publish", s.do(t, http.MethodPost, path+"/publish", "", ""), http.StatusUnauthorized)
	expect(t, "student publish", s.do(t, http.MethodPost, path+"/publish", "student", ""), http.StatusForbidden)
	expect(t, "other teacher publish", s.do(t, http.MethodPost, path+"/publish", "other", ""), http.StatusForbidden)
	expect(t, "other teacher edit draft", s.do(t, http.MethodPatch, path, "other", `{"title":"x"}`), http.StatusForbidden)

	if got := expect(t, "owner publish", s.do(t, http.MethodPost, path+"/publish", "teacher", ""), http.StatusOK); got.Status != "publicado" {
		t.Errorf("publish status = %q", got.Status)
	}
	expect(t, "publish twice", s.do(t, http.MethodPost, path+"/publish", "teacher", ""), http.StatusBadRequest)
	expect(t, "other teacher edit published", s.do(t, http.MethodPatch, path, "other", `{"title":"x"}`), http.StatusForbidden)
	expect(t, "student edit published", s.do(t, http.MethodPatch, path, "student", `{"title":"x"}`), http.StatusForbidden)
	expect(t, "invalid patch", s.do(t, http.MethodPatch, path, "teacher", `{"startAt":"2030-05-10T17:00:00Z"}`), http.StatusBadRequest)

	before := s.repo.events[e.ID].UpdatedAt
	time.Sleep(time.Millisecond)
	edited := expect(t, "owner partial edit", s.do(t, http.MethodPatch, path, "teacher", `{"title":"Taller IA avanzado"}`), http.StatusOK)
	if edited.Title != "Taller IA avanzado" || edited.Venue != "Aula 101" || edited.Status != "publicado" || !edited.UpdatedAt.After(before) {
		t.Errorf("partial edit = %+v", edited)
	}
	cleared := expect(t, "clear endAt", s.do(t, http.MethodPatch, path, "teacher", `{"endAt":null}`), http.StatusOK)
	if cleared.EndAt != nil {
		t.Errorf("clear endAt = %v, want nil", cleared.EndAt)
	}
	earlierSameDay := `{"title":"Antes del evento de día","type":"evento","startAt":"2030-05-10T10:00:00Z","endAt":"2030-05-10T12:00:00Z","venue":"Aula 101"}`
	expect(t, "before all-day event start", s.do(t, http.MethodPost, "/events", "other", earlierSameDay), http.StatusCreated)
	allDayConflict := `{"title":"Mismo día","type":"evento","startAt":"2030-05-10T18:00:00Z","endAt":"2030-05-10T19:00:00Z","venue":"Aula 101"}`
	expect(t, "same venue same day after clearing endAt", s.do(t, http.MethodPost, "/events", "other", allDayConflict), http.StatusConflict)
	differentVenue := `{"title":"Otra sala","type":"evento","startAt":"2030-05-10T18:00:00Z","endAt":"2030-05-10T19:00:00Z","venue":"Aula 202"}`
	expect(t, "different venue same day", s.do(t, http.MethodPost, "/events", "other", differentVenue), http.StatusCreated)

	if got := expect(t, "admin hide", s.do(t, http.MethodPost, path+"/hide", "admin", ""), http.StatusOK); got.Status != "oculto" {
		t.Errorf("hide status = %q", got.Status)
	}
	expect(t, "hidden anonymous detail", s.do(t, http.MethodGet, path, "", ""), http.StatusNotFound)
	expect(t, "owner republish hidden", s.do(t, http.MethodPost, path+"/publish", "teacher", ""), http.StatusOK)
	expect(t, "other teacher cancel", s.do(t, http.MethodPost, path+"/cancel", "other", ""), http.StatusForbidden)
	if got := expect(t, "owner cancel", s.do(t, http.MethodPost, path+"/cancel", "teacher", ""), http.StatusOK); got.Status != "cancelado" {
		t.Errorf("cancel status = %q", got.Status)
	}
	expect(t, "cancel twice", s.do(t, http.MethodPost, path+"/cancel", "admin", ""), http.StatusBadRequest)
	expect(t, "hide cancelled", s.do(t, http.MethodPost, path+"/hide", "admin", ""), http.StatusBadRequest)
	expect(t, "publish cancelled", s.do(t, http.MethodPost, path+"/publish", "admin", ""), http.StatusBadRequest)
	expect(t, "edit cancelled", s.do(t, http.MethodPatch, path, "admin", `{"title":"x"}`), http.StatusBadRequest)
	if got := expect(t, "cancelled history", s.do(t, http.MethodGet, path, "admin", ""), http.StatusOK); got.Title != "Taller IA avanzado" || got.Status != "cancelado" {
		t.Errorf("cancelled event = %+v", got)
	}
	expect(t, "unknown event", s.do(t, http.MethodPost, "/events/999/publish", "admin", ""), http.StatusNotFound)
}

// PT-EVT-03: agenda filters, detail with registration link, upcoming versus past.
func TestAgendaFiltersAndDetail(t *testing.T) {
	s := newServer(t)
	future := expect(t, "future", s.do(t, http.MethodPost, "/events", "admin", virtual), http.StatusCreated)
	past := expect(t, "past", s.do(t, http.MethodPost, "/events", "admin", `{"title":"Pasado","type":"taller","startAt":"2020-01-01T10:00:00Z"}`), http.StatusCreated)
	draft := expect(t, "draft", s.do(t, http.MethodPost, "/events", "teacher", inPerson), http.StatusCreated)
	hidden := expect(t, "hidden", s.do(t, http.MethodPost, "/events", "admin", `{"title":"Oculto","type":"evento","startAt":"2031-01-01T10:00:00Z"}`), http.StatusCreated)
	for _, id := range []string{future.ID, past.ID, hidden.ID} {
		expect(t, "publish "+id, s.do(t, http.MethodPost, "/events/"+id+"/publish", "admin", ""), http.StatusOK)
	}
	expect(t, "hide", s.do(t, http.MethodPost, "/events/"+hidden.ID+"/hide", "admin", ""), http.StatusOK)

	published := future.ID + "," + past.ID
	for path, want := range map[string]string{
		"/events":                              published,
		"/events?type=taller":                  past.ID,
		"/events?status=proximo":               future.ID,
		"/events?status=vencido":               past.ID,
		"/events?from=2025-01-01T00:00:00Z":    future.ID,
		"/events?to=2025-01-01T00:00:00Z":      past.ID,
		"/events?status=publicado&type=evento": future.ID,
	} {
		if got := listIDs(t, s, path, ""); got != want {
			t.Errorf("anonymous %s = %q, want %q", path, got, want)
		}
	}
	if got := listIDs(t, s, "/events", "teacher"); got != published {
		t.Errorf("teacher agenda = %q, want %q", got, published)
	}
	if got := listIDs(t, s, "/events", "admin"); got != strings.Join([]string{future.ID, past.ID, draft.ID, hidden.ID}, ",") {
		t.Errorf("admin agenda = %q", got)
	}
	if got := listIDs(t, s, "/events?status=borrador", "admin"); got != draft.ID {
		t.Errorf("admin drafts = %q", got)
	}

	expect(t, "anonymous drafts", s.do(t, http.MethodGet, "/events?status=borrador", "", ""), http.StatusForbidden)
	expect(t, "teacher hidden", s.do(t, http.MethodGet, "/events?status=oculto", "teacher", ""), http.StatusForbidden)
	expect(t, "bad status", s.do(t, http.MethodGet, "/events?status=x", "", ""), http.StatusBadRequest)
	expect(t, "bad from", s.do(t, http.MethodGet, "/events?from=2030-01-01", "", ""), http.StatusBadRequest)

	got := expect(t, "public detail", s.do(t, http.MethodGet, "/events/"+future.ID, "", ""), http.StatusOK)
	if got.RegistrationURL != "https://example.com/inscripcion" || got.MeetingURL == "" {
		t.Errorf("detail = %+v", got)
	}
	expect(t, "draft anonymous", s.do(t, http.MethodGet, "/events/"+draft.ID, "", ""), http.StatusNotFound)
	expect(t, "draft other teacher", s.do(t, http.MethodGet, "/events/"+draft.ID, "other", ""), http.StatusNotFound)
	expect(t, "draft owner", s.do(t, http.MethodGet, "/events/"+draft.ID, "teacher", ""), http.StatusOK)
	expect(t, "draft admin", s.do(t, http.MethodGet, "/events/"+draft.ID, "admin", ""), http.StatusOK)
}
