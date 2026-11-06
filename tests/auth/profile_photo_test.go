package auth_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/galash-uptc/backend/internal/domain"
)

func putPhoto(server http.Handler, cookie *http.Cookie, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPut, "/users/me/photo", strings.NewReader(body))
	if cookie != nil {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	return response
}

func TestSetProfilePhotoStoresLinkForSessionUser(t *testing.T) {
	repository := &repository{user: domain.User{ID: "user-id", Status: "activo"}}
	server := newHTTPServer(repository)
	cookie := loginCookie(t, server)

	response := putPhoto(server, cookie, `{"photo_url":" https://cdn.example.com/a.png "}`)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"photo_url":"https://cdn.example.com/a.png"`) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if repository.photoUserID != "user-id" {
		t.Fatalf("photo linked to %q, want session user", repository.photoUserID)
	}

	me := httptest.NewRequest(http.MethodGet, "/users/me", nil)
	me.AddCookie(cookie)
	meResponse := httptest.NewRecorder()
	server.ServeHTTP(meResponse, me)
	var profile map[string]any
	if err := json.Unmarshal(meResponse.Body.Bytes(), &profile); err != nil || profile["photo_url"] != "https://cdn.example.com/a.png" {
		t.Fatalf("GET /users/me photo_url missing: %s", meResponse.Body.String())
	}
}

func TestSetProfilePhotoRejectsInvalidInput(t *testing.T) {
	repository := &repository{user: domain.User{ID: "user-id", Status: "activo"}}
	server := newHTTPServer(repository)
	cookie := loginCookie(t, server)

	for _, body := range []string{
		`{"photo_url":""}`,
		`{"photo_url":"javascript:alert(1)"}`,
		`{"photo_url":"ftp://example.com/a.png"}`,
		`{"photo_url":"/relative.png"}`,
		`{"photo_url":"https://example.com/` + strings.Repeat("a", 2048) + `"}`,
		`{bad json`,
	} {
		if response := putPhoto(server, cookie, body); response.Code != http.StatusBadRequest {
			t.Errorf("body %.40q: status=%d, want 400", body, response.Code)
		}
	}
	if repository.photoUserID != "" {
		t.Fatal("invalid input reached the repository")
	}
}

func TestSetProfilePhotoRequiresSession(t *testing.T) {
	repository := &repository{user: domain.User{ID: "user-id", Status: "activo"}}
	if response := putPhoto(newHTTPServer(repository), nil, `{"photo_url":"https://example.com/a.png"}`); response.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", response.Code)
	}
	if repository.photoUserID != "" {
		t.Fatal("anonymous request reached the repository")
	}
}
