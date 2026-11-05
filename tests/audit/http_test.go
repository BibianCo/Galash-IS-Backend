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
	logs   []domain.AuditLog
	filter ports.AuditFilter
}

func (r *auditRepository) List(_ context.Context, filter ports.AuditFilter) ([]domain.AuditLog, error) {
	r.filter = filter
	return r.logs, nil
}
func (r *auditRepository) Append(context.Context, domain.AuditLog) error { return nil }

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
