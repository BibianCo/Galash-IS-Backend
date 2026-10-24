package role_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpadapter "github.com/galash-uptc/backend/internal/adapters/inbound/http"
	authapp "github.com/galash-uptc/backend/internal/application/auth"
	roleapp "github.com/galash-uptc/backend/internal/application/role"
	"github.com/galash-uptc/backend/internal/domain"
)

type healthChecker struct{}

func (healthChecker) Ping(context.Context) error { return nil }

// verifier maps each Firebase token to the user with the same ID.
type verifier struct{}

func (verifier) Verify(_ context.Context, token string) (domain.Identity, error) {
	return domain.Identity{UID: token, Email: token + "@example.com"}, nil
}

type authStore struct {
	users    map[string]domain.User
	sessions map[string]string
}

func (a *authStore) FindByID(_ context.Context, id string) (domain.User, error) {
	user, ok := a.users[id]
	if !ok {
		return domain.User{}, errors.New("user not found")
	}
	return user, nil
}
func (a *authStore) FindByEmail(context.Context, string) (domain.User, error) {
	return domain.User{}, errors.New("not used")
}
func (a *authStore) FindByFirebaseUID(ctx context.Context, uid string) (domain.User, error) {
	return a.FindByID(ctx, uid)
}
func (a *authStore) SetPassword(context.Context, string, string) error          { return nil }
func (a *authStore) Search(context.Context, string, int) ([]domain.User, error) { return nil, nil }
func (a *authStore) UpdateRole(context.Context, string, string) error           { return nil }
func (a *authStore) Upsert(context.Context, domain.User) (string, error)        { return "", nil }
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

type server struct {
	handler http.Handler
	auth    *authapp.Service
	repo    *memoryRoles
}

func newServer(t *testing.T) server {
	t.Helper()
	store := &authStore{sessions: map[string]string{}, users: map[string]domain.User{
		"admin":   {ID: "admin", FirebaseUID: "admin", Role: "admin", Status: "activo"},
		"teacher": {ID: "teacher", FirebaseUID: "teacher", Role: "profesor", Status: "activo"},
		"ana":     {ID: "ana", FirebaseUID: "ana", Role: "estudiante", Status: "activo"},
		"guest":   {ID: "guest", FirebaseUID: "guest", Role: "visitante", Status: "activo"},
	}}
	auth := authapp.NewService(verifier{}, store)
	repo := newMemoryRoles()
	handler := httpadapter.NewHandler(auth, healthChecker{}).WithRoles(roleapp.NewService(repo, store)).Routes()
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

func expect(t *testing.T, name string, response *httptest.ResponseRecorder, status int) {
	t.Helper()
	if response.Code != status {
		t.Errorf("%s: status=%d, want %d: %s", name, response.Code, status, response.Body.String())
	}
}

func TestRoleEndpointsRequireAdmin(t *testing.T) {
	s := newServer(t)
	requests := []struct{ method, path, body string }{
		{http.MethodGet, "/roles", ""},
		{http.MethodGet, "/permissions", ""},
		{http.MethodGet, "/roles/estudiante/permissions", ""},
		{http.MethodPut, "/roles/estudiante/permissions", `{"permissions":["audit.read"]}`},
		{http.MethodGet, "/users/ana/roles", ""},
	}
	for _, q := range requests {
		name := q.method + " " + q.path
		expect(t, "anonymous "+name, s.do(t, q.method, q.path, "", q.body), http.StatusUnauthorized)
		expect(t, "teacher "+name, s.do(t, q.method, q.path, "teacher", q.body), http.StatusForbidden)
		expect(t, "admin "+name, s.do(t, q.method, q.path, "admin", q.body), http.StatusOK)
	}
}

func TestRoleCatalogAndPermissionsPayloads(t *testing.T) {
	s := newServer(t)
	var roles struct{ Roles []domain.Role }
	if err := json.NewDecoder(s.do(t, http.MethodGet, "/roles", "admin", "").Body).Decode(&roles); err != nil || len(roles.Roles) != 4 {
		t.Fatalf("roles=%+v err=%v", roles, err)
	}
	var perms struct{ Permissions []domain.Permission }
	if err := json.NewDecoder(s.do(t, http.MethodGet, "/permissions", "admin", "").Body).Decode(&perms); err != nil || len(perms.Permissions) != 3 {
		t.Fatalf("permissions=%+v err=%v", perms, err)
	}
	var role struct {
		Role        domain.Role
		Permissions []domain.Permission
	}
	if err := json.NewDecoder(s.do(t, http.MethodGet, "/roles/admin/permissions", "admin", "").Body).Decode(&role); err != nil || role.Role.Name != "Admin" || len(role.Permissions) != 3 {
		t.Fatalf("role permissions=%+v err=%v", role, err)
	}
	expect(t, "unknown role", s.do(t, http.MethodGet, "/roles/coordinador/permissions", "admin", ""), http.StatusNotFound)
}

func TestSetRolePermissionsErrors(t *testing.T) {
	s := newServer(t)
	cases := []struct {
		name, path, body string
		status           int
	}{
		{"own role", "/roles/admin/permissions", `{"permissions":[]}`, http.StatusConflict},
		{"unknown role", "/roles/coordinador/permissions", `{"permissions":[]}`, http.StatusNotFound},
		{"unknown permission", "/roles/estudiante/permissions", `{"permissions":["no.existe"]}`, http.StatusBadRequest},
		{"missing field", "/roles/estudiante/permissions", `{}`, http.StatusBadRequest},
		{"invalid json", "/roles/estudiante/permissions", `{`, http.StatusBadRequest},
	}
	for _, c := range cases {
		expect(t, c.name, s.do(t, http.MethodPut, c.path, "admin", c.body), c.status)
	}
	if len(s.repo.byRole["estudiante"]) != 1 || len(s.repo.byRole["admin"]) != 3 {
		t.Fatalf("rejected requests changed permissions: %+v", s.repo.byRole)
	}
	expect(t, "contract label", s.do(t, http.MethodPut, "/roles/Docente/permissions", "admin", `{"permissions":["files.upload"]}`), http.StatusOK)
	if len(s.repo.byRole["profesor"]) != 1 || s.repo.actor != "admin" {
		t.Fatalf("profesor=%+v actor=%q", s.repo.byRole["profesor"], s.repo.actor)
	}
}

func TestUserRoles(t *testing.T) {
	s := newServer(t)
	var body struct {
		UserID string
		Roles  []domain.Role
	}
	if err := json.NewDecoder(s.do(t, http.MethodGet, "/users/teacher/roles", "admin", "").Body).Decode(&body); err != nil || body.UserID != "teacher" || len(body.Roles) != 1 || body.Roles[0].Name != "Docente" {
		t.Fatalf("user roles=%+v err=%v", body, err)
	}
	expect(t, "unknown user", s.do(t, http.MethodGet, "/users/missing/roles", "admin", ""), http.StatusNotFound)
}

func TestDashboard(t *testing.T) {
	s := newServer(t)
	expect(t, "anonymous", s.do(t, http.MethodGet, "/users/me/dashboard", "", ""), http.StatusUnauthorized)
	expect(t, "visitante", s.do(t, http.MethodGet, "/users/me/dashboard", "guest", ""), http.StatusForbidden)
	var d domain.Dashboard
	if err := json.NewDecoder(s.do(t, http.MethodGet, "/users/me/dashboard", "ana", "").Body).Decode(&d); err != nil || d.User.ID != "ana" || d.Role.ID != "estudiante" || len(d.Permissions) != 1 {
		t.Fatalf("dashboard=%+v err=%v", d, err)
	}
	// The existing /users/me route must keep working next to /users/me/dashboard.
	expect(t, "users/me", s.do(t, http.MethodGet, "/users/me", "ana", ""), http.StatusOK)
}

func TestRoleModuleNotConfigured(t *testing.T) {
	s := newServer(t)
	handler := httpadapter.NewHandler(s.auth, healthChecker{}).Routes()
	request := httptest.NewRequest(http.MethodGet, "/roles", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	expect(t, "unconfigured", response, http.StatusInternalServerError)
}
