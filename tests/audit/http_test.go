package audit_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpadapter "github.com/galash-uptc/backend/internal/adapters/inbound/http"
	auditapp "github.com/galash-uptc/backend/internal/application/audit"
	authapp "github.com/galash-uptc/backend/internal/application/auth"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
)

type verifier struct{}

func (verifier) Verify(context.Context, string) (domain.Identity, error) {
	return domain.Identity{UID: "firebase-admin", Email: "admin@example.com"}, nil
}

type users struct {
	user     domain.User
	sessions map[string]string
}

func (u *users) Upsert(context.Context, domain.User) (string, error)              { return u.user.ID, nil }
func (u *users) FindByID(context.Context, string) (domain.User, error)            { return u.user, nil }
func (u *users) FindByEmail(context.Context, string) (domain.User, error)         { return u.user, nil }
func (u *users) FindByFirebaseUID(context.Context, string) (domain.User, error)   { return u.user, nil }
func (u *users) SetPassword(context.Context, string, string) error                { return nil }
func (u *users) Search(context.Context, string, int, bool) ([]domain.User, error) { return nil, nil }
func (u *users) UpdateRole(context.Context, string, string) error                 { return nil }
func (u *users) Create(_ context.Context, id, token string) error                 { u.sessions[token] = id; return nil }
func (u *users) FindUserID(_ context.Context, token string) (string, error) {
	id, ok := u.sessions[token]
	if !ok {
		return "", errors.New("missing session")
	}
	return id, nil
}
func (u *users) Revoke(context.Context, string) error           { return nil }
func (u *users) RevokeAllForUser(context.Context, string) error { return nil }
func (u *users) DeleteExpiredSessions(context.Context) error    { return nil }

type auditRepository struct {
	logs     []domain.AuditLog
	filter   ports.AuditFilter
	appended []domain.AuditLog
}

func (r *auditRepository) List(_ context.Context, filter ports.AuditFilter) ([]domain.AuditLog, error) {
	r.filter = filter
	return r.logs, nil
}
func (r *auditRepository) Append(_ context.Context, item domain.AuditLog) error {
	r.appended = append(r.appended, item)
	return nil
}

type health struct{}

func (health) Ping(context.Context) error { return nil }

func server(repo *auditRepository, userRepo *users) http.Handler {
	auth := authapp.NewService(verifier{}, userRepo)
	return httpadapter.NewHandlerWithOfficialLinksAndAudit(auth, health{}, nil, nil, nil, nil, nil, nil, nil, nil, nil, auditapp.NewService(repo)).Routes()
}

func session(t *testing.T, server http.Handler) *http.Cookie {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/auth/sessions", strings.NewReader(`{"token":"firebase-token"}`))
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)
	if res.Code != http.StatusCreated {
		t.Fatalf("login status=%d: %s", res.Code, res.Body.String())
	}
	return res.Result().Cookies()[0]
}

func TestAuditLogsRequiresAdminAndParsesFilters(t *testing.T) {
	repo := &auditRepository{logs: []domain.AuditLog{{ID: "1", Action: "role.updated"}}}
	userRepo := &users{user: domain.User{ID: "admin-id", Role: "Admin", Status: "activo"}, sessions: map[string]string{}}
	server := server(repo, userRepo)
	cookie := session(t, server)
	req := httptest.NewRequest(http.MethodGet, "/audit-logs?actor=actor-id&action=role.updated&entity=users&from=2026-01-01T00:00:00Z&to=2026-01-02T00:00:00Z", nil)
	req.AddCookie(cookie)
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)
	if res.Code != http.StatusOK || !strings.Contains(res.Body.String(), `"auditLogs"`) {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if repo.filter.ActorID != "actor-id" || repo.filter.Action != "role.updated" || repo.filter.Entity != "users" {
		t.Fatalf("unexpected filter: %+v", repo.filter)
	}
}

func TestAuditLogsRejectsInvalidPeriod(t *testing.T) {
	repo := &auditRepository{}
	userRepo := &users{user: domain.User{ID: "admin-id", Role: "admin", Status: "activo"}, sessions: map[string]string{}}
	server := server(repo, userRepo)
	cookie := session(t, server)
	req := httptest.NewRequest(http.MethodGet, "/audit-logs?from=2026-01-02T00:00:00Z&to=2026-01-01T00:00:00Z", nil)
	req.AddCookie(cookie)
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
}

func get(server http.Handler, path string, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	res := httptest.NewRecorder()
	server.ServeHTTP(res, req)
	return res
}

// PT-TR-03 / PT-TR-01: only an Admin session can read the audit log.
func TestAuditLogsAccessControl(t *testing.T) {
	repo := &auditRepository{}
	if res := get(server(repo, &users{sessions: map[string]string{}}), "/audit-logs", nil); res.Code != http.StatusUnauthorized {
		t.Fatalf("without session status=%d", res.Code)
	}
	for _, role := range []string{"estudiante", "docente", "visitante"} {
		userRepo := &users{user: domain.User{ID: role + "-id", Role: role, Status: "activo"}, sessions: map[string]string{}}
		s := server(repo, userRepo)
		if res := get(s, "/audit-logs", session(t, s)); res.Code != http.StatusForbidden {
			t.Fatalf("%s status=%d", role, res.Code)
		}
	}
	userRepo := &users{user: domain.User{ID: "admin-id", Role: "admin", Status: "activo"}, sessions: map[string]string{}}
	s := server(repo, userRepo)
	cookie := session(t, s)
	userRepo.user.Status = "inactivo"
	if res := get(s, "/audit-logs", cookie); res.Code != http.StatusUnauthorized {
		t.Fatalf("inactive admin status=%d", res.Code)
	}
}

// PT-TR-03: malformed period values are rejected before querying.
func TestAuditLogsRejectsMalformedDates(t *testing.T) {
	repo := &auditRepository{}
	userRepo := &users{user: domain.User{ID: "admin-id", Role: "admin", Status: "activo"}, sessions: map[string]string{}}
	s := server(repo, userRepo)
	cookie := session(t, s)
	for _, q := range []string{"from=2026-01-01", "to=ayer", "from=2026-13-01T00:00:00Z"} {
		if res := get(s, "/audit-logs?"+q, cookie); res.Code != http.StatusBadRequest {
			t.Fatalf("%s status=%d", q, res.Code)
		}
	}
	if res := get(s, "/audit-logs?from=2026-01-01T00:00:00Z", cookie); res.Code != http.StatusOK || repo.filter.From == nil || repo.filter.To != nil {
		t.Fatalf("open period status=%d filter=%+v", res.Code, repo.filter)
	}
}

// PT-TR-03 / TDB-T10: critical writes are recorded internally with actor,
// action, entity, result and origin; reads and anonymous calls are not.
func TestAuditMiddlewareRecordsCriticalWrites(t *testing.T) {
	repo := &auditRepository{}
	userRepo := &users{user: domain.User{ID: "admin-id", Role: "admin", Status: "activo"}, sessions: map[string]string{}}
	s := server(repo, userRepo)
	cookie := session(t, s)
	send := func(method, path string, cookie *http.Cookie) {
		req := httptest.NewRequest(method, path, strings.NewReader(`{}`))
		req.RemoteAddr = "192.0.2.10:5000"
		if cookie != nil {
			req.AddCookie(cookie)
		}
		s.ServeHTTP(httptest.NewRecorder(), req)
	}
	send(http.MethodGet, "/links", cookie)
	send(http.MethodPost, "/links", nil)
	send(http.MethodPost, "/files", cookie)
	if len(repo.appended) != 0 {
		t.Fatalf("reads, anonymous or non-critical writes were audited: %+v", repo.appended)
	}
	for _, path := range []string{"/links", "/seedbeds", "/events", "/announcements", "/admin/registration-requests/x/status", "/users/x/roles", "/admin/content/pages"} {
		repo.appended = nil
		send(http.MethodPost, path, cookie)
		if len(repo.appended) != 1 {
			t.Fatalf("%s audited %d times", path, len(repo.appended))
		}
		got := repo.appended[0]
		if got.ActorID != "admin-id" || got.Action != "POST "+path || got.Entity != path || got.Origin != "192.0.2.10:5000" || got.Result != "failure" || got.Date.IsZero() {
			t.Fatalf("%s audit entry = %+v", path, got)
		}
	}
}

// PT-TR-03: the API exposes no way to edit or delete audit entries.
func TestAuditLogsHaveNoWriteEndpoints(t *testing.T) {
	repo := &auditRepository{}
	userRepo := &users{user: domain.User{ID: "admin-id", Role: "admin", Status: "activo"}, sessions: map[string]string{}}
	s := server(repo, userRepo)
	cookie := session(t, s)
	for _, method := range []string{http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete} {
		for _, path := range []string{"/audit-logs", "/audit-logs/1"} {
			req := httptest.NewRequest(method, path, nil)
			req.AddCookie(cookie)
			res := httptest.NewRecorder()
			s.ServeHTTP(res, req)
			if res.Code != http.StatusMethodNotAllowed && res.Code != http.StatusNotFound {
				t.Fatalf("%s %s status=%d", method, path, res.Code)
			}
		}
	}
}
