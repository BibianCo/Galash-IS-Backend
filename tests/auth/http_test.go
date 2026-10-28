package auth_test

import (
	"context"
	"errors"
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

func loginCookie(t *testing.T, server http.Handler) *http.Cookie {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/auth/sessions", strings.NewReader(`{"token":"firebase-token"}`))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("login status=%d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
	}
	cookies := response.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("login did not set a session cookie")
	}
	return cookies[0]
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

func TestCreateSessionAcceptsAuthorizationHeaderWithoutBody(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/auth/sessions", nil)
	request.Header.Set("Authorization", "Bearer firebase-token")
	response := httptest.NewRecorder()

	newHTTPServer(&repository{user: domain.User{ID: "user-id", Status: "activo"}}).ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status=%d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
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

func TestLocalFrontendPreflightIsAccepted(t *testing.T) {
	request := httptest.NewRequest(http.MethodOptions, "/users/me", nil)
	request.Header.Set("Origin", "http://localhost:5173")
	request.Header.Set("Access-Control-Request-Method", http.MethodGet)
	request.Header.Set("Access-Control-Request-Headers", "Authorization")
	response := httptest.NewRecorder()

	newHTTPServer(&repository{}).ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("status=%d, want %d: %s", response.Code, http.StatusNoContent, response.Body.String())
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" ||
		response.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatalf("missing local CORS headers: %v", response.Header())
	}
}

func TestAuthorizationHeaderTakesPrecedenceOverStaleSessionCookie(t *testing.T) {
	repository := &repository{user: domain.User{ID: "user-id", Name: "Ana", Status: "activo"}}
	server := newHTTPServer(repository)

	validCookie := loginCookie(t, server)
	request := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	request.AddCookie(&http.Cookie{Name: "galash_session", Value: "stale-session"})
	request.Header.Set("Authorization", "Bearer "+validCookie.Value)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
}

func TestAdminMembersRejectsAuthenticatedNonAdmin(t *testing.T) {
	repository := &repository{user: domain.User{ID: "user-id", Role: "estudiante", Status: "activo"}}
	server := newHTTPServer(repository)

	loginRequest := httptest.NewRequest(http.MethodPost, "/auth/sessions", strings.NewReader(`{"token":"firebase-token"}`))
	loginResponse := httptest.NewRecorder()
	server.ServeHTTP(loginResponse, loginRequest)
	if loginResponse.Code != http.StatusCreated {
		t.Fatalf("login status=%d, want %d", loginResponse.Code, http.StatusCreated)
	}

	request := httptest.NewRequest(http.MethodGet, "/admin/members?q=ana", nil)
	request.AddCookie(loginResponse.Result().Cookies()[0])
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status=%d, want %d: %s", response.Code, http.StatusForbidden, response.Body.String())
	}
}

func TestPublicMembersRequiresQueryAndOmitsPrivateFields(t *testing.T) {
	repository := &repository{
		user: domain.User{ID: "session-user", Status: "activo"},
		searchUsers: []domain.User{{
			ID: "public-id", FirebaseUID: "private-firebase-uid", Name: "Ana", LastName: "Pérez",
			Email: "private@example.test", DocumentType: "CC", DocumentNumber: "private-document", Role: "profesor", Status: "activo",
		}},
	}
	server := newHTTPServer(repository)

	missingQuery := httptest.NewRecorder()
	server.ServeHTTP(missingQuery, httptest.NewRequest(http.MethodGet, "/members", nil))
	if missingQuery.Code != http.StatusBadRequest {
		t.Fatalf("missing-query status=%d, want %d", missingQuery.Code, http.StatusBadRequest)
	}

	response := httptest.NewRecorder()
	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/members?q=ana", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	body := response.Body.String()
	for _, expected := range []string{`"id":"public-id"`, `"name":"Ana"`, `"role":"profesor"`} {
		if !strings.Contains(body, expected) {
			t.Errorf("public response missing %s: %s", expected, body)
		}
	}
	for _, private := range []string{"private-firebase-uid", "private@example.test", "private-document", "document_type", "firebase_uid"} {
		if strings.Contains(body, private) {
			t.Errorf("public response exposes %q: %s", private, body)
		}
	}
}

func TestAdminMembersRequiresSessionAndReturnsAdminFields(t *testing.T) {
	repository := &repository{
		user: domain.User{ID: "admin-id", FirebaseUID: "firebase-id", Role: "Admin", Status: "activo"},
		searchUsers: []domain.User{{
			ID: "member-id", FirebaseUID: "member-firebase-uid", Name: "Ana", LastName: "Pérez",
			Email: "ana@example.test", DocumentType: "CC", DocumentNumber: "12345", Role: "profesor", Status: "activo",
		}},
	}
	server := newHTTPServer(repository)

	unauthorized := httptest.NewRecorder()
	server.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/admin/members?q=ana", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated status=%d, want %d", unauthorized.Code, http.StatusUnauthorized)
	}

	legacy := httptest.NewRecorder()
	server.ServeHTTP(legacy, httptest.NewRequest(http.MethodGet, "/find/user?q=ana", nil))
	if legacy.Code != http.StatusNotFound {
		t.Fatalf("legacy alias status=%d, want %d", legacy.Code, http.StatusNotFound)
	}

	request := httptest.NewRequest(http.MethodGet, "/admin/members?q=ana", nil)
	request.AddCookie(loginCookie(t, server))
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status=%d, want %d: %s", response.Code, http.StatusOK, response.Body.String())
	}
	if !repository.searchInactive {
		t.Error("admin search must include inactive accounts")
	}
	for _, expected := range []string{"member-firebase-uid", "ana@example.test", "12345"} {
		if !strings.Contains(response.Body.String(), expected) {
			t.Errorf("admin response missing %q: %s", expected, response.Body.String())
		}
	}
}

func TestUpdateRoleAcceptsHeaderAndPathTargets(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		header string
		wantID string
	}{
		{name: "uid header", path: "/users/roles", header: "selected-uid", wantID: "selected-uid"},
		{name: "path uid", path: "/users/path-uid/roles", wantID: "path-uid"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &repository{user: domain.User{ID: "admin-id", FirebaseUID: "firebase-id", Role: "Admin", Status: "activo"}}
			server := newHTTPServer(repository)
			request := httptest.NewRequest(http.MethodPut, test.path, strings.NewReader(`{"role":"profesor"}`))
			request.AddCookie(loginCookie(t, server))
			if test.header != "" {
				request.Header.Set("X-User-UID", test.header)
			}
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			if response.Code != http.StatusNoContent {
				t.Fatalf("status=%d, want %d: %s", response.Code, http.StatusNoContent, response.Body.String())
			}
			if repository.updatedUID != test.wantID || repository.updatedRole != "profesor" {
				t.Fatalf("updated UID=%q role=%q, want %q/profesor", repository.updatedUID, repository.updatedRole, test.wantID)
			}
		})
	}
}

func TestUpdateRoleRejectsUnauthenticatedAndConflictingTargets(t *testing.T) {
	t.Run("unauthenticated", func(t *testing.T) {
		server := newHTTPServer(&repository{user: domain.User{ID: "admin-id", Role: "admin", Status: "activo"}})
		request := httptest.NewRequest(http.MethodPut, "/users/roles", strings.NewReader(`{"role":"profesor"}`))
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized {
			t.Fatalf("status=%d, want %d", response.Code, http.StatusUnauthorized)
		}
	})

	t.Run("conflicting header and path", func(t *testing.T) {
		repository := &repository{user: domain.User{ID: "admin-id", FirebaseUID: "firebase-id", Role: "admin", Status: "activo"}}
		server := newHTTPServer(repository)
		request := httptest.NewRequest(http.MethodPut, "/users/path-uid/roles", strings.NewReader(`{"role":"profesor"}`))
		request.AddCookie(loginCookie(t, server))
		request.Header.Set("X-User-UID", "different-uid")
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request)
		if response.Code != http.StatusBadRequest {
			t.Fatalf("status=%d, want %d", response.Code, http.StatusBadRequest)
		}
		if repository.updatedUID != "" {
			t.Fatalf("conflicting request updated UID %q", repository.updatedUID)
		}
	})
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

func TestCreateSessionReturnsRegistrationRequiredForFirebaseOnlyUser(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "/auth/sessions", strings.NewReader(`{"token":"firebase-token"}`))
	response := httptest.NewRecorder()

	newHTTPServerWithVerifier(&repository{findUIDErr: errors.New("usuario no encontrado")}, identityVerifier{identity: domain.Identity{UID: "firebase-id", Email: "ana@example.com"}}).ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status=%d, want %d", response.Code, http.StatusForbidden)
	}
	body := response.Body.String()
	for _, expected := range []string{`"code":"REGISTRATION_REQUIRED"`, `"action":"complete_registration"`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("response=%s, missing %s", body, expected)
		}
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

func TestRegisterIgnoresClientSuppliedRole(t *testing.T) {
	repository := &repository{user: domain.User{ID: "user-id"}}
	verifier := identityVerifier{identity: domain.Identity{UID: "firebase-id", Email: "ana@example.com"}}
	body := `{"nombre":"Ana","apellido":"Pérez","tipo_dni":"CC","dni":"123","email":"ana@example.com","rol":"admin"}`
	request := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer firebase-token")
	response := httptest.NewRecorder()

	newHTTPServerWithVerifier(repository, verifier).ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("status=%d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
	}
	if repository.upsertedUser.Role == "admin" {
		t.Fatal("client-supplied role reached the repository")
	}
}

func TestLogoutRevokesCookieSessionEvenWhenStaleAuthorizationIsPresent(t *testing.T) {
	repository := &repository{user: domain.User{ID: "user-id", Name: "Ana", Status: "activo"}}
	server := newHTTPServer(repository)
	sessionCookie := loginCookie(t, server)

	logoutRequest := httptest.NewRequest(http.MethodDelete, "/auth/sessions/current", nil)
	logoutRequest.AddCookie(sessionCookie)
	logoutRequest.Header.Set("Authorization", "Bearer some-other-token")
	logoutResponse := httptest.NewRecorder()
	server.ServeHTTP(logoutResponse, logoutRequest)
	if logoutResponse.Code != http.StatusNoContent {
		t.Fatalf("logout status=%d, want %d", logoutResponse.Code, http.StatusNoContent)
	}

	currentUserRequest := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	currentUserRequest.AddCookie(sessionCookie)
	currentUserResponse := httptest.NewRecorder()
	server.ServeHTTP(currentUserResponse, currentUserRequest)
	if currentUserResponse.Code != http.StatusUnauthorized {
		t.Fatalf("cookie session survived logout: status=%d, want %d", currentUserResponse.Code, http.StatusUnauthorized)
	}
}

func TestLoginRotatesPreviousSessionCookie(t *testing.T) {
	repository := &repository{user: domain.User{ID: "user-id", Status: "activo"}}
	server := newHTTPServer(repository)
	previousCookie := loginCookie(t, server)

	request := httptest.NewRequest(http.MethodPost, "/auth/sessions", strings.NewReader(`{"token":"firebase-token"}`))
	request.AddCookie(previousCookie)
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("second login status=%d, want %d: %s", response.Code, http.StatusCreated, response.Body.String())
	}
	if !repository.revoked[sessionHash(previousCookie.Value)] {
		t.Fatal("previous session cookie was not revoked on re-login")
	}
}

func TestStaleSessionCookieIsClearedOnUnauthorized(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	request.AddCookie(&http.Cookie{Name: "galash_session", Value: "expired-session"})
	response := httptest.NewRecorder()

	newHTTPServer(&repository{}).ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want %d", response.Code, http.StatusUnauthorized)
	}
	cookies := response.Result().Cookies()
	if len(cookies) == 0 || cookies[0].Name != "galash_session" || cookies[0].MaxAge >= 0 || !cookies[0].Secure {
		t.Fatalf("stale cookie was not expired: %+v", cookies)
	}
}
