package seedbed_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpadapter "github.com/galash-uptc/backend/internal/adapters/inbound/http"
	authapp "github.com/galash-uptc/backend/internal/application/auth"
	application "github.com/galash-uptc/backend/internal/application/seedbed"
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

type server struct {
	handler http.Handler
	auth    *authapp.Service
	repo    *repository
}

func newServer(t *testing.T) server {
	t.Helper()
	u, r := fixtures()
	r.space = domain.SeedbedSpace{Responsible: domain.ResponsibleSummary{ID: "teacher", Email: "teacher@example.com"}}
	auth := authapp.NewService(verifier{}, &authStore{users: u, sessions: map[string]string{}})
	handler := httpadapter.NewHandler(auth, healthChecker{}, application.NewService(r, u)).Routes()
	return server{handler: handler, auth: auth, repo: r}
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

const seedbedBody = `{"name":"Robótica","description":"d","researchLine":"IA","objective":"o","responsibleTeacherId":"teacher","contactEmail":"semillero@example.com"}`

func TestCreateSeedbedRequiresTeacherOrAdminSession(t *testing.T) {
	s := newServer(t)
	expect(t, "anonymous", s.do(t, http.MethodPost, "/seedbeds", "", seedbedBody), http.StatusUnauthorized)
	expect(t, "student", s.do(t, http.MethodPost, "/seedbeds", "student", seedbedBody), http.StatusForbidden)
	expect(t, "missing responsible", s.do(t, http.MethodPost, "/seedbeds", "admin", `{"name":"Robótica"}`), http.StatusBadRequest)
	response := s.do(t, http.MethodPost, "/seedbeds", "teacher", seedbedBody)
	expect(t, "teacher", response, http.StatusCreated)
	for _, field := range []string{`"id":"new-seedbed"`, `"status":"Borrador"`, `"responsibleTeacherId":"teacher"`} {
		if !strings.Contains(response.Body.String(), field) {
			t.Errorf("create response missing %s: %s", field, response.Body.String())
		}
	}
}

func TestListSeedbedsRestrictsPrivateViewToAdmin(t *testing.T) {
	s := newServer(t)
	expect(t, "public list", s.do(t, http.MethodGet, "/seedbeds", "", ""), http.StatusOK)
	if s.repo.private {
		t.Fatal("public list requested private seedbeds")
	}
	expect(t, "anonymous private", s.do(t, http.MethodGet, "/seedbeds?includeInactive=true", "", ""), http.StatusUnauthorized)
	expect(t, "teacher private", s.do(t, http.MethodGet, "/seedbeds?includeInactive=true", "teacher", ""), http.StatusForbidden)
	expect(t, "admin private", s.do(t, http.MethodGet, "/seedbeds?includeInactive=true", "admin", ""), http.StatusOK)
	if !s.repo.private {
		t.Fatal("admin list did not request private seedbeds")
	}
}

func TestGetSeedbedRespectsVisibility(t *testing.T) {
	s := newServer(t)
	expect(t, "published anonymous", s.do(t, http.MethodGet, "/seedbeds/public", "", ""), http.StatusOK)
	expect(t, "hidden anonymous", s.do(t, http.MethodGet, "/seedbeds/hidden", "", ""), http.StatusUnauthorized)
	expect(t, "hidden student", s.do(t, http.MethodGet, "/seedbeds/hidden", "student", ""), http.StatusNotFound)
	expect(t, "hidden responsible", s.do(t, http.MethodGet, "/seedbeds/hidden", "teacher", ""), http.StatusOK)
	expect(t, "hidden admin", s.do(t, http.MethodGet, "/seedbeds/hidden", "admin", ""), http.StatusOK)
	expect(t, "missing", s.do(t, http.MethodGet, "/seedbeds/missing", "", ""), http.StatusNotFound)
}

func TestSeedbedSpaceHidesPrivateData(t *testing.T) {
	s := newServer(t)
	response := s.do(t, http.MethodGet, "/seedbeds/public/space", "", "")
	expect(t, "public space", response, http.StatusOK)
	if strings.Contains(response.Body.String(), "teacher@example.com") {
		t.Errorf("public space exposes responsible email: %s", response.Body.String())
	}
	expect(t, "hidden space student", s.do(t, http.MethodGet, "/seedbeds/hidden/space", "student", ""), http.StatusNotFound)
	response = s.do(t, http.MethodGet, "/seedbeds/hidden/space", "teacher", "")
	expect(t, "hidden space responsible", response, http.StatusOK)
	if !strings.Contains(response.Body.String(), "teacher@example.com") {
		t.Errorf("responsible space omits contact email: %s", response.Body.String())
	}
}

func TestUpdateSeedbedRequiresResponsibleOrAdmin(t *testing.T) {
	s := newServer(t)
	expect(t, "anonymous", s.do(t, http.MethodPatch, "/seedbeds/public", "", seedbedBody), http.StatusUnauthorized)
	expect(t, "other teacher", s.do(t, http.MethodPatch, "/seedbeds/public", "other", seedbedBody), http.StatusForbidden)
	expect(t, "responsible", s.do(t, http.MethodPatch, "/seedbeds/public", "teacher", seedbedBody), http.StatusOK)
	expect(t, "admin", s.do(t, http.MethodPatch, "/seedbeds/public", "admin", seedbedBody), http.StatusOK)
}

func TestSetSeedbedVisibilityValidatesActorAndState(t *testing.T) {
	s := newServer(t)
	expect(t, "anonymous", s.do(t, http.MethodPatch, "/seedbeds/public/visibility", "", `{"status":"Oculto"}`), http.StatusUnauthorized)
	expect(t, "other teacher", s.do(t, http.MethodPatch, "/seedbeds/public/visibility", "other", `{"status":"Oculto"}`), http.StatusForbidden)
	expect(t, "unknown state", s.do(t, http.MethodPatch, "/seedbeds/public/visibility", "teacher", `{"status":"Eliminado"}`), http.StatusBadRequest)
	expect(t, "responsible", s.do(t, http.MethodPatch, "/seedbeds/public/visibility", "teacher", `{"visibility":"Inactivo"}`), http.StatusOK)
	if s.repo.visibility != domain.SeedbedInactive {
		t.Errorf("visibility=%q, want %q", s.repo.visibility, domain.SeedbedInactive)
	}
}

func TestSeedbedMemberHistoryIsRestricted(t *testing.T) {
	s := newServer(t)
	expect(t, "public members", s.do(t, http.MethodGet, "/seedbeds/public/members", "", ""), http.StatusOK)
	expect(t, "history anonymous", s.do(t, http.MethodGet, "/seedbeds/public/members?history=true", "", ""), http.StatusUnauthorized)
	expect(t, "history student", s.do(t, http.MethodGet, "/seedbeds/public/members?history=true", "student", ""), http.StatusForbidden)
	expect(t, "history responsible", s.do(t, http.MethodGet, "/seedbeds/public/members?history=true", "teacher", ""), http.StatusOK)
	if !s.repo.history {
		t.Error("responsible history request did not include withdrawals")
	}
	response := s.do(t, http.MethodGet, "/seedbeds/hidden/members", "", "")
	expect(t, "hidden members anonymous", response, http.StatusUnauthorized)
	if strings.Count(response.Body.String(), `"error"`) != 1 {
		t.Errorf("hidden members anonymous wrote several error bodies: %s", response.Body.String())
	}
	expect(t, "hidden members student", s.do(t, http.MethodGet, "/seedbeds/hidden/members", "student", ""), http.StatusNotFound)
}

func TestLinkAndWithdrawMemberRequireResponsibleOrAdmin(t *testing.T) {
	s := newServer(t)
	expect(t, "link anonymous", s.do(t, http.MethodPost, "/seedbeds/public/members", "", `{"userId":"student"}`), http.StatusUnauthorized)
	expect(t, "link other teacher", s.do(t, http.MethodPost, "/seedbeds/public/members", "other", `{"userId":"student"}`), http.StatusForbidden)
	expect(t, "link responsible", s.do(t, http.MethodPost, "/seedbeds/public/members", "teacher", `{"userId":"student"}`), http.StatusNoContent)
	expect(t, "withdraw student", s.do(t, http.MethodDelete, "/seedbeds/public/members/student", "student", ""), http.StatusForbidden)
	expect(t, "withdraw admin", s.do(t, http.MethodDelete, "/seedbeds/public/members/student", "admin", ""), http.StatusNoContent)
	if s.repo.added != "student" || s.repo.removed != "student" {
		t.Errorf("added=%q removed=%q, want student", s.repo.added, s.repo.removed)
	}
}
