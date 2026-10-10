package auth_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpadapter "github.com/galash-uptc/backend/internal/adapters/inbound/http"
	application "github.com/galash-uptc/backend/internal/application/auth"
	"github.com/galash-uptc/backend/internal/domain"
)

type healthChecker struct{}

func (healthChecker) Ping(context.Context) error { return nil }

func newHTTPServer(repository *repository) http.Handler {
	return newHTTPServerWithVerifier(repository, identityVerifier{identity: domain.Identity{UID: "firebase-id", Email: "ana@example.com"}})
}

func newHTTPServerWithVerifier(repository *repository, verifier identityVerifier) http.Handler {
	service := application.NewService(verifier, repository)
	return httpadapter.NewHandler(service, healthChecker{}).Routes()
}

func TestCreateSessionSetsSecureCookie(t *testing.T) {
	repository := &repository{user: domain.User{ID: "user-id", Email: "ana@example.com", Status: "activo"}}
	request := httptest.NewRequest(http.MethodPost, "/auth/sessions", strings.NewReader(`{"token":"firebase-token"}`))
	response := httptest.NewRecorder()

	newHTTPServer(repository).ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status=%d, want %d", response.Code, http.StatusCreated)
	}
	cookie := response.Result().Cookies()[0]
	if cookie.Name != "galash_session" || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("unsafe session cookie: %+v", cookie)
	}
}

func TestCurrentUserRequiresSession(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	response := httptest.NewRecorder()

	newHTTPServer(&repository{user: domain.User{ID: "user-id", Status: "activo"}}).ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestRegisterRequiresBearerToken(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{}`))
	response := httptest.NewRecorder()

	newHTTPServer(&repository{user: domain.User{ID: "user-id"}}).ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestCreateSessionReturnsGenericErrorForInvalidOrInactiveAccounts(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		verifier   identityVerifier
		repository *repository
	}{
		{
			name:       "invalid credentials",
			body:       `{"token":"invalid-token"}`,
			verifier:   identityVerifier{err: domain.ErrInvalidCredentials},
			repository: &repository{user: domain.User{ID: "user-id", Status: "activo"}},
		},
		{
			name:       "inactive account",
			body:       `{"token":"firebase-token"}`,
			verifier:   identityVerifier{identity: domain.Identity{UID: "uid", Email: "ana@example.com"}},
			repository: &repository{user: domain.User{ID: "user-id", Status: "inactivo"}},
		},
		{
			name:       "missing token",
			body:       `{}`,
			verifier:   identityVerifier{err: domain.ErrInvalidCredentials},
			repository: &repository{user: domain.User{ID: "user-id", Status: "activo"}},
		},
	}

	var expectedBody string
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/auth/sessions", strings.NewReader(test.body))
			response := httptest.NewRecorder()
			newHTTPServerWithVerifier(test.repository, test.verifier).ServeHTTP(response, request)

			if response.Code != http.StatusUnauthorized {
				t.Fatalf("status=%d, want %d", response.Code, http.StatusUnauthorized)
			}
			if expectedBody == "" {
				expectedBody = response.Body.String()
			} else if response.Body.String() != expectedBody {
				t.Fatalf("response reveals credential/account state: got %q, want %q", response.Body.String(), expectedBody)
			}
		})
	}
}

func TestCreateSessionPreservesStudentAndProfessorRoles(t *testing.T) {
	for _, role := range []string{"estudiante", "profesor"} {
		t.Run(role, func(t *testing.T) {
			repository := &repository{user: domain.User{ID: "user-id", Role: role, Status: "activo"}}
			request := httptest.NewRequest(http.MethodPost, "/auth/sessions", strings.NewReader(`{"token":"firebase-token"}`))
			response := httptest.NewRecorder()

			newHTTPServer(repository).ServeHTTP(response, request)

			if response.Code != http.StatusCreated {
				t.Fatalf("status=%d, want %d", response.Code, http.StatusCreated)
			}
			if !strings.Contains(response.Body.String(), `"role":"`+role+`"`) {
				t.Fatalf("login response does not preserve role %q: %s", role, response.Body.String())
			}
		})
	}
}

func TestLogoutInvalidatesSessionAndClearsCookie(t *testing.T) {
	repository := &repository{user: domain.User{ID: "user-id", Name: "Ana", Status: "activo"}}
	server := newHTTPServer(repository)

	loginRequest := httptest.NewRequest(http.MethodPost, "/auth/sessions", strings.NewReader(`{"token":"firebase-token"}`))
	loginResponse := httptest.NewRecorder()
	server.ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusCreated {
		t.Fatalf("login status=%d, want %d", loginResponse.Code, http.StatusCreated)
	}
	sessionCookie := loginResponse.Result().Cookies()[0]

	logoutRequest := httptest.NewRequest(http.MethodDelete, "/auth/sessions/current", nil)
	logoutRequest.AddCookie(sessionCookie)
	logoutResponse := httptest.NewRecorder()
	server.ServeHTTP(logoutResponse, logoutRequest)
	if logoutResponse.Code != http.StatusNoContent {
		t.Fatalf("logout status=%d, want %d", logoutResponse.Code, http.StatusNoContent)
	}
	clearedCookie := logoutResponse.Result().Cookies()[0]
	if clearedCookie.Name != sessionCookie.Name || clearedCookie.Value != "" || clearedCookie.MaxAge >= 0 || !clearedCookie.HttpOnly || !clearedCookie.Secure {
		t.Fatalf("logout did not clear the secure session cookie: %+v", clearedCookie)
	}

	currentUserRequest := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	currentUserRequest.AddCookie(sessionCookie)
	currentUserResponse := httptest.NewRecorder()
	server.ServeHTTP(currentUserResponse, currentUserRequest)
	if currentUserResponse.Code != http.StatusUnauthorized {
		t.Fatalf("current-user status after logout=%d, want %d", currentUserResponse.Code, http.StatusUnauthorized)
	}
}
