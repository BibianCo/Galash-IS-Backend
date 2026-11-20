package research_test

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
	seedbedapp "github.com/galash-uptc/backend/internal/application/seedbed"
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
	service, r, u := fixturesWithUsers()
	auth := authapp.NewService(verifier{}, &authStore{users: u, sessions: map[string]string{}})
	handler := httpadapter.NewHandlerWithResearch(auth, healthChecker{}, seedbedapp.NewService(seedbeds{}, u), nil, service).Routes()
	return server{handler: handler, auth: auth, repo: r}
}

// do sends a request as userID and decodes the JSON response into out when given.
func (s server) do(t *testing.T, method, path, userID, body string, status int, out any) {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	login, err := s.auth.Login(context.Background(), userID)
	if err != nil {
		t.Fatalf("login %s: %v", userID, err)
	}
	request.Header.Set("Authorization", "Bearer "+login.Token)
	response := httptest.NewRecorder()
	s.handler.ServeHTTP(response, request)
	if response.Code != status {
		t.Fatalf("%s %s: want %d, got %d %s", method, path, status, response.Code, response.Body)
	}
	if out != nil {
		if err := json.Unmarshal(response.Body.Bytes(), out); err != nil {
			t.Fatalf("decode %s: %v", response.Body, err)
		}
	}
}

func TestPatchKeepsOmittedFields(t *testing.T) {
	s := newServer(t)
	var project domain.Project
	s.do(t, "PATCH", "/projects/proj", "teacher", `{"status":"Finalizado"}`, 200, &project)
	if project.Title != "P" || project.Progress != 10 || project.Status != "Finalizado" || len(s.repo.changes) != 1 {
		t.Fatalf("project=%+v changes=%v", project, s.repo.changes)
	}
	var activity domain.Activity
	s.do(t, "PATCH", "/activities/act", "teacher", `{"status":"en curso"}`, 200, &activity)
	if activity.Title != "Taller" || activity.Status != "en curso" {
		t.Fatalf("activity=%+v", activity)
	}
	s.do(t, "PATCH", "/activities/act", "teacher", `{"title":" "}`, 400, nil)
	s.do(t, "PATCH", "/research-proposals/prop", "student", `{"title":"Nuevo"}`, 200, nil)
	if p := s.repo.saved; p.Title != "Nuevo" || p.Problem != "P" || p.Methodology != "M" || p.Status != "Radicada" {
		t.Fatalf("proposal=%+v", p)
	}
}

// raw sends a request with the given Authorization header ("" = none) and returns status and body.
func (s server) raw(method, path, authorization string) (int, string) {
	request := httptest.NewRequest(method, path, nil)
	if authorization != "" {
		request.Header.Set("Authorization", authorization)
	}
	response := httptest.NewRecorder()
	s.handler.ServeHTTP(response, request)
	return response.Code, response.Body.String()
}

func TestProjectProgressHiddenForVisitorsAndAnonymous(t *testing.T) {
	s := newServer(t)
	for _, path := range []string{"/projects", "/projects/proj"} {
		code, body := s.raw("GET", path, "")
		if code != 200 || strings.Contains(body, `"progress"`) || !strings.Contains(body, `"title":"P"`) {
			t.Fatalf("anonymous %s: %d %s", path, code, body)
		}
		if code, body := s.raw("GET", path, "Bearer invalida"); code != 401 {
			t.Fatalf("invalid session %s: want 401, got %d %s", path, code, body)
		}
		var visitor map[string]any
		s.do(t, "GET", path, "visitor", "", 200, &visitor)
		if strings.Contains(mustJSON(t, visitor), `"progress"`) {
			t.Fatalf("visitor %s sees progress: %v", path, visitor)
		}
		for _, user := range []string{"student", "teacher", "admin"} {
			var got map[string]any
			s.do(t, "GET", path, user, "", 200, &got)
			if !strings.Contains(mustJSON(t, got), `"progress":10`) {
				t.Fatalf("%s %s missing progress: %v", user, path, got)
			}
		}
	}
	s.do(t, "GET", "/projects/progress", "student", "", 200, nil)
	s.do(t, "GET", "/projects/progress", "visitor", "", 403, nil)
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}
