package announcements_test

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
	announcementapp "github.com/galash-uptc/backend/internal/application/announcement"
	authapp "github.com/galash-uptc/backend/internal/application/auth"
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

// memoryRepo mirrors the PostgreSQL adapter visibility rules in memory.
type memoryRepo struct {
	items map[string]domain.Announcement
	reads map[string]time.Time // key announcementID|userID
	users userMap
	next  int
}

func (r *memoryRepo) visible(a domain.Announcement, userID string) bool {
	return a.Status == "publicado" || a.AuthorID == userID
}
func (r *memoryRepo) withRead(a domain.Announcement, userID string) domain.Announcement {
	_, a.ReadConfirmed = r.reads[a.ID+"|"+userID]
	return a
}
func (r *memoryRepo) List(_ context.Context, userID string, all bool) ([]domain.Announcement, error) {
	out := []domain.Announcement{}
	for _, a := range r.items {
		if r.visible(a, userID) || all && a.Status == "borrador" {
			out = append(out, r.withRead(a, userID))
		}
	}
	return out, nil
}
func (r *memoryRepo) Find(_ context.Context, id, userID string, all bool) (domain.Announcement, error) {
	a, ok := r.items[id]
	if !ok || !(r.visible(a, userID) || all && a.Status == "borrador") {
		return domain.Announcement{}, domain.ErrAnnouncementNotFound
	}
	return r.withRead(a, userID), nil
}
func (r *memoryRepo) Create(_ context.Context, a domain.Announcement) (domain.Announcement, error) {
	r.next++
	a.ID, a.Status, a.CreatedAt, a.UpdatedAt = fmt.Sprint(r.next), "borrador", time.Now(), time.Now()
	r.items[a.ID] = a
	return a, nil
}
func (r *memoryRepo) Update(_ context.Context, id, title, description string, at []domain.AnnouncementAttachment) (domain.Announcement, error) {
	a := r.items[id]
	a.Title, a.Description, a.Attachments, a.UpdatedAt = title, description, at, time.Now()
	r.items[id] = a
	return a, nil
}
func (r *memoryRepo) Publish(_ context.Context, id string) (domain.Announcement, error) {
	a := r.items[id]
	now := time.Now()
	a.Status, a.PublishedAt, a.UpdatedAt = "publicado", &now, now
	r.items[id] = a
	return a, nil
}
func (r *memoryRepo) ConfirmRead(_ context.Context, id, userID string) error {
	key := id + "|" + userID
	if _, done := r.reads[key]; !done && r.items[id].Status == "publicado" {
		r.reads[key] = time.Now()
	}
	return nil
}
func (r *memoryRepo) ReadStatus(_ context.Context, id string) ([]domain.AnnouncementReadStatus, error) {
	out := []domain.AnnouncementReadStatus{}
	for _, u := range r.users {
		if u.Role != "docente" || u.Status != "activo" {
			continue
		}
		s := domain.AnnouncementReadStatus{UserID: u.ID, Status: "Pendiente"}
		if at, ok := r.reads[id+"|"+u.ID]; ok {
			s.Status, s.ConfirmedAt = "Leído", &at
		}
		out = append(out, s)
	}
	return out, nil
}

type server struct {
	handler http.Handler
	auth    *authapp.Service
}

func newServer(t *testing.T) server {
	t.Helper()
	u := userMap{
		"admin":    {ID: "admin", Role: "admin", Status: "activo"},
		"teacher":  {ID: "teacher", Role: "docente", Status: "activo"},
		"teacher2": {ID: "teacher2", Role: "docente", Status: "activo"},
		"student":  {ID: "student", Role: "estudiante", Status: "activo"},
		"inactive": {ID: "inactive", Role: "docente", Status: "inactivo"},
	}
	repo := &memoryRepo{items: map[string]domain.Announcement{}, reads: map[string]time.Time{}, users: u}
	auth := authapp.NewService(verifier{}, &authStore{userMap: u, sessions: map[string]string{}})
	h := httpadapter.NewHandlerWithAnnouncements(auth, healthChecker{}, nil, nil, nil, nil, nil, nil, announcementapp.NewService(repo, u)).Routes()
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

func expect(t *testing.T, name string, res *httptest.ResponseRecorder, status int) domain.Announcement {
	t.Helper()
	if res.Code != status {
		t.Errorf("%s: status=%d, want %d: %s", name, res.Code, status, res.Body.String())
	}
	var a domain.Announcement
	_ = json.Unmarshal(res.Body.Bytes(), &a)
	return a
}

const validBody = `{"title":"Directriz","description":"Entregar informe","attachments":[{"name":"guia.pdf","url":"https://files.example/guia.pdf"},{"name":"notas.xlsx","url":"https://files.example/notas.xlsx"},{"name":"acta.docx","url":"https://files.example/acta.docx"}]}`

// PT-COM-01 / PT-TR-01: only Docente/Admin sessions reach the module (inactive accounts cannot log in; see service test).
func TestAnnouncementAccessControl(t *testing.T) {
	s := newServer(t)
	expect(t, "no session", s.do(t, http.MethodGet, "/announcements", "", ""), http.StatusUnauthorized)
	expect(t, "no session create", s.do(t, http.MethodPost, "/announcements", "", validBody), http.StatusUnauthorized)
	for _, user := range []string{"student"} {
		expect(t, user+" list", s.do(t, http.MethodGet, "/announcements", user, ""), http.StatusForbidden)
		expect(t, user+" create", s.do(t, http.MethodPost, "/announcements", user, validBody), http.StatusForbidden)
	}
	expect(t, "admin list", s.do(t, http.MethodGet, "/announcements", "admin", ""), http.StatusOK)
}

// PT-COM-01: form validation for title, description and PDF/Word/Excel attachments.
func TestAnnouncementCreateValidation(t *testing.T) {
	s := newServer(t)
	a := expect(t, "valid", s.do(t, http.MethodPost, "/announcements", "teacher", validBody), http.StatusCreated)
	if a.Status != "borrador" || a.AuthorID != "teacher" || len(a.Attachments) != 3 {
		t.Errorf("created = %+v", a)
	}
	cases := map[string]string{
		"invalid json":   `{`,
		"empty title":    `{"title":" ","description":"x"}`,
		"empty desc":     `{"title":"x","description":""}`,
		"exe attachment": `{"title":"x","description":"x","attachments":[{"name":"a.exe","url":"https://f/a"}]}`,
		"no extension":   `{"title":"x","description":"x","attachments":[{"name":"guia","url":"https://f/a"}]}`,
		"js url":         `{"title":"x","description":"x","attachments":[{"name":"a.pdf","url":"javascript:alert(1)"}]}`,
		"relative url":   `{"title":"x","description":"x","attachments":[{"name":"a.pdf","url":"/a.pdf"}]}`,
		"11 attachments": `{"title":"x","description":"x","attachments":[` + strings.TrimSuffix(strings.Repeat(`{"name":"a.pdf","url":"https://f/a"},`, 11), ",") + `]}`,
	}
	for name, body := range cases {
		expect(t, name, s.do(t, http.MethodPost, "/announcements", "teacher", body), http.StatusBadRequest)
	}
	// author is taken from the session, never from the payload
	spoof := expect(t, "spoof author", s.do(t, http.MethodPost, "/announcements", "teacher", `{"title":"x","description":"x","authorId":"admin"}`), http.StatusCreated)
	if spoof.AuthorID != "teacher" {
		t.Errorf("authorId = %q, want teacher", spoof.AuthorID)
	}
}

// PT-COM-01: draft edit, publication lifecycle and ownership rules.
func TestAnnouncementEditAndPublishLifecycle(t *testing.T) {
	s := newServer(t)
	a := expect(t, "create", s.do(t, http.MethodPost, "/announcements", "teacher", validBody), http.StatusCreated)
	path := "/announcements/" + a.ID

	expect(t, "other teacher sees draft", s.do(t, http.MethodGet, path, "teacher2", ""), http.StatusNotFound)
	expect(t, "other teacher edits draft", s.do(t, http.MethodPatch, path, "teacher2", `{"title":"hack"}`), http.StatusNotFound)
	expect(t, "other teacher publishes", s.do(t, http.MethodPost, path+"/publish", "teacher2", ""), http.StatusNotFound)

	u := expect(t, "partial edit", s.do(t, http.MethodPatch, path, "teacher", `{"title":"Directriz v2"}`), http.StatusOK)
	if u.Title != "Directriz v2" || u.Description != "Entregar informe" || len(u.Attachments) != 3 {
		t.Errorf("partial edit lost fields: %+v", u)
	}
	expect(t, "edit invalid attachment", s.do(t, http.MethodPatch, path, "teacher", `{"attachments":[{"name":"x.zip","url":"https://f/x"}]}`), http.StatusBadRequest)

	p := expect(t, "publish", s.do(t, http.MethodPost, path+"/publish", "teacher", ""), http.StatusOK)
	if p.Status != "publicado" || p.PublishedAt == nil {
		t.Errorf("published = %+v", p)
	}
	expect(t, "republish", s.do(t, http.MethodPost, path+"/publish", "teacher", ""), http.StatusBadRequest)
	expect(t, "edit published", s.do(t, http.MethodPatch, path, "teacher", `{"title":"tarde"}`), http.StatusBadRequest)
	expect(t, "other teacher reads published", s.do(t, http.MethodGet, path, "teacher2", ""), http.StatusOK)
	expect(t, "missing", s.do(t, http.MethodGet, "/announcements/999", "teacher", ""), http.StatusNotFound)
}

// PT-COM-01: an Admin lists every draft and may edit/publish any of them (service grants admin ownership override).
func TestAdminManagesOtherAuthorsDraft(t *testing.T) {
	s := newServer(t)
	a := expect(t, "create", s.do(t, http.MethodPost, "/announcements", "teacher", validBody), http.StatusCreated)
	path := "/announcements/" + a.ID
	expect(t, "admin opens listed draft", s.do(t, http.MethodGet, path, "admin", ""), http.StatusOK)
	expect(t, "admin edits draft", s.do(t, http.MethodPatch, path, "admin", `{"title":"revisado"}`), http.StatusOK)
	expect(t, "admin publishes draft", s.do(t, http.MethodPost, path+"/publish", "admin", ""), http.StatusOK)
}

// PT-COM-02: read confirmation is idempotent and feeds the author's Leído/Pendiente panel.
func TestAnnouncementReadConfirmationAndStatus(t *testing.T) {
	s := newServer(t)
	a := expect(t, "create", s.do(t, http.MethodPost, "/announcements", "teacher", validBody), http.StatusCreated)
	path := "/announcements/" + a.ID

	expect(t, "confirm draft by non-author", s.do(t, http.MethodPost, path+"/read-confirmations", "teacher2", ""), http.StatusNotFound)
	expect(t, "publish", s.do(t, http.MethodPost, path+"/publish", "teacher", ""), http.StatusOK)

	expect(t, "confirm", s.do(t, http.MethodPost, path+"/read-confirmations", "teacher2", ""), http.StatusOK)
	expect(t, "confirm again", s.do(t, http.MethodPost, path+"/read-confirmations", "teacher2", ""), http.StatusOK)
	got := expect(t, "detail after confirm", s.do(t, http.MethodGet, path, "teacher2", ""), http.StatusOK)
	if !got.ReadConfirmed {
		t.Errorf("readConfirmed=false after confirmation")
	}

	expect(t, "status by non-author", s.do(t, http.MethodGet, path+"/read-status", "teacher2", ""), http.StatusForbidden)
	expect(t, "status by student", s.do(t, http.MethodGet, path+"/read-status", "student", ""), http.StatusForbidden)
	expect(t, "status by admin", s.do(t, http.MethodGet, path+"/read-status", "admin", ""), http.StatusOK)
	res := s.do(t, http.MethodGet, path+"/read-status", "teacher", "")
	expect(t, "status by author", res, http.StatusOK)
	var body struct {
		ReadStatus []domain.AnnouncementReadStatus `json:"readStatus"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	status := map[string]domain.AnnouncementReadStatus{}
	for _, r := range body.ReadStatus {
		status[r.UserID] = r
	}
	if len(status) != 2 || status["teacher2"].Status != "Leído" || status["teacher2"].ConfirmedAt == nil || status["teacher"].Status != "Pendiente" {
		t.Errorf("read status = %+v", body.ReadStatus)
	}
}
