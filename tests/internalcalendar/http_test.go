package internalcalendar_test

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
	calendarapp "github.com/galash-uptc/backend/internal/application/internalcalendar"
	"github.com/galash-uptc/backend/internal/domain"
)

type healthChecker struct{}

func (healthChecker) Ping(context.Context) error { return nil }

// verifier maps each Firebase token to the user with the same ID.
type verifier struct{}

func (verifier) Verify(_ context.Context, token string) (domain.Identity, error) {
	return domain.Identity{UID: token, Email: token + "@example.com"}, nil
}

type userMap map[string]domain.User

func (u userMap) FindByID(_ context.Context, id string) (domain.User, error) {
	user, ok := u[id]
	if !ok {
		return domain.User{}, errors.New("user not found")
	}
	return user, nil
}
func (u userMap) FindByEmail(context.Context, string) (domain.User, error) {
	return domain.User{}, errors.New("not used")
}
func (u userMap) FindByFirebaseUID(ctx context.Context, uid string) (domain.User, error) {
	return u.FindByID(ctx, uid)
}
func (userMap) SetPassword(context.Context, string, string) error                { return nil }
func (userMap) Search(context.Context, string, int, bool) ([]domain.User, error) { return nil, nil }
func (userMap) UpdateRole(context.Context, string, string) error                 { return nil }

type authStore struct {
	userMap
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

// memoryRepo mirrors the PostgreSQL adapter rules: Admin sees everything, others
// only events of seedbeds they belong to or lead; the leader manages them.
type memoryRepo struct {
	items   map[string]domain.InternalCalendarEvent
	members map[string][]string // seedbedID -> active members
	leaders map[string]string   // seedbedID -> leader
	next    int
}

func (r *memoryRepo) allowed(actorID, role, seedbed string) bool {
	if strings.EqualFold(role, "admin") || r.leaders[seedbed] == actorID {
		return true
	}
	for _, m := range r.members[seedbed] {
		if m == actorID {
			return true
		}
	}
	return false
}
func (r *memoryRepo) List(_ context.Context, actorID, role, seedbedID string, from, to *time.Time) ([]domain.InternalCalendarEvent, error) {
	out := []domain.InternalCalendarEvent{}
	for _, e := range r.items {
		if r.allowed(actorID, role, e.SeedbedID) && (seedbedID == "" || e.SeedbedID == seedbedID) && (from == nil || !e.StartAt.Before(*from)) && (to == nil || !e.StartAt.After(*to)) {
			out = append(out, e)
		}
	}
	return out, nil
}
func (r *memoryRepo) Find(_ context.Context, actorID, role, id string) (domain.InternalCalendarEvent, error) {
	e, ok := r.items[id]
	if !ok || !r.allowed(actorID, role, e.SeedbedID) {
		return domain.InternalCalendarEvent{}, domain.ErrInternalCalendarNotFound
	}
	return e, nil
}
func (r *memoryRepo) Create(_ context.Context, e domain.InternalCalendarEvent) (domain.InternalCalendarEvent, error) {
	r.next++
	e.ID, e.CreatedAt, e.UpdatedAt = fmt.Sprint(r.next), time.Now(), time.Now()
	r.items[e.ID] = e
	return e, nil
}
func (r *memoryRepo) Update(_ context.Context, e domain.InternalCalendarEvent) (domain.InternalCalendarEvent, error) {
	e.UpdatedAt = time.Now()
	r.items[e.ID] = e
	return e, nil
}
func (r *memoryRepo) Cancel(_ context.Context, id, _ string) (domain.InternalCalendarEvent, error) {
	e := r.items[id]
	e.Status = "cancelado"
	r.items[id] = e
	return e, nil
}
func (r *memoryRepo) CanManage(_ context.Context, e domain.InternalCalendarEvent, actorID, role string) (bool, error) {
	return strings.EqualFold(role, "admin") || r.leaders[e.SeedbedID] == actorID, nil
}
func (r *memoryRepo) HasConflict(_ context.Context, e domain.InternalCalendarEvent, exclude string) (bool, error) {
	for _, o := range r.items {
		if o.ID == exclude || o.Status != "activo" || !o.StartAt.Before(e.EndAt) || !o.EndAt.After(e.StartAt) {
			continue
		}
		for _, a := range o.ParticipantIDs {
			for _, b := range e.ParticipantIDs {
				if a == b {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

type server struct {
	handler http.Handler
	auth    *authapp.Service
}

func newServer(t *testing.T) server {
	t.Helper()
	u := userMap{
		"admin":    {ID: "admin", Role: "admin", Status: "activo"},
		"director": {ID: "director", Role: "docente", Status: "activo"},
		"student":  {ID: "student", Role: "estudiante", Status: "activo"},
		"student2": {ID: "student2", Role: "estudiante", Status: "activo"},
		"outsider": {ID: "outsider", Role: "estudiante", Status: "activo"},
	}
	repo := &memoryRepo{
		items:   map[string]domain.InternalCalendarEvent{},
		members: map[string][]string{"sb1": {"student", "student2"}},
		leaders: map[string]string{"sb1": "director"},
	}
	auth := authapp.NewService(verifier{}, &authStore{userMap: u, sessions: map[string]string{}})
	h := httpadapter.NewHandlerWithCalendar(auth, healthChecker{}, nil, nil, nil, nil, nil, nil, nil, calendarapp.NewService(repo)).Routes()
	return server{handler: h, auth: auth}
}

// do sends a request as userID; an empty userID sends it without a session.
func (s server) do(t *testing.T, method, path, userID, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if userID != "" {
		login, err := s.auth.Login(context.Background(), userID)
		if err != nil {
			t.Fatalf("login %s: %v", userID, err)
		}
		req.Header.Set("Authorization", "Bearer "+login.Token)
	}
	res := httptest.NewRecorder()
	s.handler.ServeHTTP(res, req)
	return res
}

func expect(t *testing.T, name string, res *httptest.ResponseRecorder, status int) domain.InternalCalendarEvent {
	t.Helper()
	if res.Code != status {
		t.Errorf("%s: status=%d, want %d: %s", name, res.Code, status, res.Body.String())
	}
	var e domain.InternalCalendarEvent
	_ = json.Unmarshal(res.Body.Bytes(), &e)
	return e
}

// listIDs expects a 200 list response and returns its event IDs.
func listIDs(t *testing.T, name string, res *httptest.ResponseRecorder) []string {
	t.Helper()
	expect(t, name, res, http.StatusOK)
	var body struct {
		Events []domain.InternalCalendarEvent `json:"events"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode list: %v: %s", err, res.Body.String())
	}
	ids := []string{}
	for _, e := range body.Events {
		ids = append(ids, e.ID)
	}
	return ids
}

const path = "/internal-calendar/events"

func body(typ, start, end string, participants ...string) string {
	p, _ := json.Marshal(participants)
	return fmt.Sprintf(`{"seedbedId":"sb1","type":%q,"title":"Tutoría","description":"Revisión","startAt":%q,"endAt":%q,"participantIds":%s}`, typ, start, end, p)
}

var validBody = body("tutoria", "2026-11-10T14:00:00Z", "2026-11-10T15:00:00Z", "student")

// PT-CRO-01 / PT-TR-01: session required; students read their seedbed but cannot write.
func TestCalendarAccessControl(t *testing.T) {
	s := newServer(t)
	created := expect(t, "director create", s.do(t, http.MethodPost, path, "director", validBody), http.StatusCreated)
	item := path + "/" + created.ID
	for _, r := range [][2]string{{http.MethodGet, path}, {http.MethodPost, path}, {http.MethodGet, item}, {http.MethodPatch, item}, {http.MethodDelete, item}} {
		expect(t, "no session "+r[0]+" "+r[1], s.do(t, r[0], r[1], "", validBody), http.StatusUnauthorized)
	}
	if ids := listIDs(t, "student list", s.do(t, http.MethodGet, path, "student", "")); len(ids) != 1 || ids[0] != created.ID {
		t.Errorf("student list = %v, want [%s]", ids, created.ID)
	}
	expect(t, "student detail", s.do(t, http.MethodGet, item, "student", ""), http.StatusOK)
	expect(t, "student create", s.do(t, http.MethodPost, path, "student", validBody), http.StatusForbidden)
	expect(t, "student patch", s.do(t, http.MethodPatch, item, "student", `{"title":"x"}`), http.StatusForbidden)
	expect(t, "student delete", s.do(t, http.MethodDelete, item, "student", ""), http.StatusForbidden)
	if ids := listIDs(t, "outsider list", s.do(t, http.MethodGet, path, "outsider", "")); len(ids) != 0 {
		t.Errorf("student of another seedbed lists %v", ids)
	}
	expect(t, "outsider detail", s.do(t, http.MethodGet, item, "outsider", ""), http.StatusNotFound)
	expect(t, "admin detail", s.do(t, http.MethodGet, item, "admin", ""), http.StatusOK)
}

// PT-CRO-01: payload, type, date range, participants and query validation.
func TestCalendarCreateValidation(t *testing.T) {
	s := newServer(t)
	for name, b := range map[string]string{
		"malformed JSON":  `{`,
		"missing startAt": `{"seedbedId":"sb1","type":"hito","title":"Entrega","endAt":"2026-11-10T15:00:00Z","participantIds":["student"]}`,
		"invalid type":    body("otro", "2026-11-10T14:00:00Z", "2026-11-10T15:00:00Z", "student"),
		"end before":      body("hito", "2026-11-10T15:00:00Z", "2026-11-10T14:00:00Z", "student"),
		"end equals":      body("hito", "2026-11-10T15:00:00Z", "2026-11-10T15:00:00Z", "student"),
		"no participants": body("reunion", "2026-11-10T14:00:00Z", "2026-11-10T15:00:00Z"),
		"blank title":     strings.Replace(validBody, `"Tutoría"`, `"  "`, 1),
		"missing seedbed": strings.Replace(validBody, `"sb1"`, `""`, 1),
	} {
		expect(t, name, s.do(t, http.MethodPost, path, "director", b), http.StatusBadRequest)
	}
	for _, q := range []string{"?from=ayer", "?to=2026-13-01", "?view=day", "?from=2026-11-10T00:00:00Z&to=2026-11-01T00:00:00Z"} {
		expect(t, "list "+q, s.do(t, http.MethodGet, path+q, "director", ""), http.StatusBadRequest)
	}
	e := expect(t, "valid create", s.do(t, http.MethodPost, path, "director", validBody), http.StatusCreated)
	if e.OrganizerID != "director" || e.Status != "activo" || e.Type != "tutoria" || e.SeedbedID != "sb1" {
		t.Errorf("created = %+v", e)
	}
	in := listIDs(t, "range hit", s.do(t, http.MethodGet, path+"?view=week&seedbedId=sb1&from=2026-11-09T00:00:00Z&to=2026-11-15T23:59:59Z", "director", ""))
	out := listIDs(t, "range miss", s.do(t, http.MethodGet, path+"?view=month&from=2026-12-01T00:00:00Z&to=2026-12-31T23:59:59Z", "director", ""))
	if len(in) != 1 || len(out) != 0 {
		t.Errorf("range filter in=%v out=%v", in, out)
	}
}

// PT-CRO-01 / TDB-T08: partial update, schedule conflict, logical cancellation and its final state.
func TestCalendarPatchConflictAndCancel(t *testing.T) {
	s := newServer(t)
	a := expect(t, "create a", s.do(t, http.MethodPost, path, "director", validBody), http.StatusCreated)
	expect(t, "overlap same participant", s.do(t, http.MethodPost, path, "director", body("reunion", "2026-11-10T14:30:00Z", "2026-11-10T15:30:00Z", "student")), http.StatusConflict)
	b := expect(t, "adjacent slot", s.do(t, http.MethodPost, path, "director", body("reunion", "2026-11-10T15:00:00Z", "2026-11-10T16:00:00Z", "student")), http.StatusCreated)
	expect(t, "other participant overlap", s.do(t, http.MethodPost, path, "director", body("hito", "2026-11-10T14:30:00Z", "2026-11-10T15:30:00Z", "student2")), http.StatusCreated)

	itemA := path + "/" + a.ID
	p := expect(t, "partial patch", s.do(t, http.MethodPatch, itemA, "director", `{"title":"Tutoría reprogramada"}`), http.StatusOK)
	if p.Title != "Tutoría reprogramada" || p.Type != "tutoria" || !p.StartAt.Equal(a.StartAt) || len(p.ParticipantIDs) != 1 || p.OrganizerID != "director" {
		t.Errorf("patched = %+v", p)
	}
	expect(t, "patch into conflict", s.do(t, http.MethodPatch, itemA, "director", `{"endAt":"2026-11-10T15:30:00Z"}`), http.StatusConflict)
	expect(t, "patch invalid range", s.do(t, http.MethodPatch, itemA, "director", `{"endAt":"2026-11-10T13:00:00Z"}`), http.StatusBadRequest)
	expect(t, "patch empty participants", s.do(t, http.MethodPatch, itemA, "director", `{"participantIds":[]}`), http.StatusBadRequest)
	expect(t, "patch malformed", s.do(t, http.MethodPatch, itemA, "director", `{`), http.StatusBadRequest)
	expect(t, "patch unknown", s.do(t, http.MethodPatch, path+"/999", "director", `{"title":"x"}`), http.StatusNotFound)

	c := expect(t, "cancel b", s.do(t, http.MethodDelete, path+"/"+b.ID, "admin", ""), http.StatusOK)
	if c.Status != "cancelado" {
		t.Errorf("cancel status = %q", c.Status)
	}
	expect(t, "cancel twice", s.do(t, http.MethodDelete, path+"/"+b.ID, "admin", ""), http.StatusBadRequest)
	expect(t, "patch cancelled", s.do(t, http.MethodPatch, path+"/"+b.ID, "director", `{"title":"x"}`), http.StatusBadRequest)
	if got := expect(t, "detail cancelled", s.do(t, http.MethodGet, path+"/"+b.ID, "student", ""), http.StatusOK); got.Status != "cancelado" {
		t.Errorf("cancelled event not kept: %+v", got)
	}
	expect(t, "slot freed after cancel", s.do(t, http.MethodPatch, itemA, "director", `{"endAt":"2026-11-10T15:30:00Z"}`), http.StatusOK)
	expect(t, "delete unknown", s.do(t, http.MethodDelete, path+"/999", "director", ""), http.StatusNotFound)
}
