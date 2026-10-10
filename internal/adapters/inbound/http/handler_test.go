package httpadapter

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	application "github.com/galash-uptc/backend/internal/application/auth"
	"github.com/galash-uptc/backend/internal/domain"
)

type handlerIdentityVerifier struct{ identity domain.Identity }

func (v handlerIdentityVerifier) Verify(context.Context, string) (domain.Identity, error) {
	return v.identity, nil
}

type handlerRepository struct {
	user        domain.User
	sessionHash string
	revokedHash string
}

func (r *handlerRepository) Upsert(context.Context, domain.User) (string, error) {
	return r.user.ID, nil
}
func (r *handlerRepository) FindByID(context.Context, string) (domain.User, error) {
	return r.user, nil
}
func (r *handlerRepository) FindByEmail(context.Context, string) (domain.User, error) {
	return r.user, nil
}
func (r *handlerRepository) SetPassword(context.Context, string, string) error { return nil }
func (r *handlerRepository) Create(_ context.Context, userID, tokenHash string) error {
	r.sessionHash = tokenHash
	if userID != r.user.ID {
		return errors.New("unexpected user")
	}
	return nil
}
func (r *handlerRepository) FindUserID(_ context.Context, tokenHash string) (string, error) {
	if tokenHash != r.sessionHash {
		return "", errors.New("session not found")
	}
	return r.user.ID, nil
}
func (r *handlerRepository) Revoke(_ context.Context, tokenHash string) error {
	r.revokedHash = tokenHash
	return nil
}

type handlerHealthChecker struct{}

func (handlerHealthChecker) Ping(context.Context) error { return nil }

func newHandlerTestServer(repository *handlerRepository) http.Handler {
	service := application.NewService(
		handlerIdentityVerifier{identity: domain.Identity{UID: "firebase-id", Name: "Ana", Email: "ana@example.com"}},
		repository,
	)
	return NewHandler(service, handlerHealthChecker{}).Routes()
}

func TestCreateSessionSetsSecureCookieAndHidesFirebaseUID(t *testing.T) {
	repository := &handlerRepository{user: domain.User{ID: "user-id", FirebaseUID: "firebase-id", Name: "Ana", Email: "ana@example.com", Status: "activo"}}
	server := newHandlerTestServer(repository)
	request := httptest.NewRequest(http.MethodPost, "/auth/sessions", strings.NewReader(`{"token":"firebase-token"}`))
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusCreated {
		t.Fatalf("POST /auth/sessions status = %d, want %d", response.Code, http.StatusCreated)
	}
	cookie := response.Result().Cookies()[0]
	if cookie.Name != "galash_session" || !cookie.HttpOnly || !cookie.Secure || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("unexpected session cookie: %+v", cookie)
	}
	if strings.Contains(response.Body.String(), "firebase_uid") {
		t.Fatal("session response exposed firebase_uid")
	}
}

func TestRegisterRequiresBearerToken(t *testing.T) {
	server := newHandlerTestServer(&handlerRepository{user: domain.User{ID: "user-id"}})
	request := httptest.NewRequest(http.MethodPost, "/auth/register", strings.NewReader(`{"nombre":"Ana"}`))
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("POST /auth/register status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestCurrentUserRequiresSession(t *testing.T) {
	server := newHandlerTestServer(&handlerRepository{user: domain.User{ID: "user-id", Status: "activo"}})
	request := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("GET /users/me status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestSessionCookieSupportsCurrentUserAndLogout(t *testing.T) {
	repository := &handlerRepository{user: domain.User{ID: "user-id", Name: "Ana", Status: "activo"}}
	service := application.NewService(
		handlerIdentityVerifier{identity: domain.Identity{UID: "firebase-id", Email: "ana@example.com"}},
		repository,
	)
	login, err := service.Login(context.Background(), "firebase-token")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	server := NewHandler(service, handlerHealthChecker{}).Routes()

	currentRequest := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	currentRequest.AddCookie(&http.Cookie{Name: "galash_session", Value: login.Token})
	currentResponse := httptest.NewRecorder()
	server.ServeHTTP(currentResponse, currentRequest)
	if currentResponse.Code != http.StatusOK {
		t.Fatalf("GET /users/me status = %d, want %d", currentResponse.Code, http.StatusOK)
	}
	var user domain.User
	if err := json.Unmarshal(currentResponse.Body.Bytes(), &user); err != nil {
		t.Fatalf("decode current user: %v", err)
	}
	if user.ID != "user-id" {
		t.Fatalf("current user = %+v", user)
	}

	logoutRequest := httptest.NewRequest(http.MethodDelete, "/auth/sessions/current", nil)
	logoutRequest.AddCookie(&http.Cookie{Name: "galash_session", Value: login.Token})
	logoutResponse := httptest.NewRecorder()
	server.ServeHTTP(logoutResponse, logoutRequest)
	if logoutResponse.Code != http.StatusNoContent {
		t.Fatalf("DELETE /auth/sessions/current status = %d, want %d", logoutResponse.Code, http.StatusNoContent)
	}
	if repository.revokedHash == "" {
		t.Fatal("logout did not revoke session")
	}
}
