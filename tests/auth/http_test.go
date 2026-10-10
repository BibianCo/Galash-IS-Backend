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
	service := application.NewService(identityVerifier{identity: domain.Identity{UID: "firebase-id", Email: "ana@example.com"}}, repository)
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
