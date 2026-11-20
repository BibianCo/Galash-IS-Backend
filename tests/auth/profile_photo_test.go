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

func putUserPhoto(server http.Handler, cookie *http.Cookie, userID, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPut, "/users/"+userID+"/photo", strings.NewReader(body))
	if cookie != nil {
		request.AddCookie(cookie)
	}
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	return response
}

// Admin edits another account's image link; "" clears it.
func TestAdminSetsAndClearsUserPhoto(t *testing.T) {
	repository := &repository{user: domain.User{ID: "admin-id", Role: "admin", Status: "activo"}}
	server := newHTTPServer(repository)
	cookie := loginCookie(t, server)

	response := putUserPhoto(server, cookie, "member-7", `{"photo_url":" https://cdn.example.com/m7.jpg "}`)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"photo_url":"https://cdn.example.com/m7.jpg"`) {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
	if repository.photoUserID != "member-7" {
		t.Fatalf("photo linked to %q, want member-7", repository.photoUserID)
	}
	if response := putUserPhoto(server, cookie, "member-7", `{"photo_url":""}`); response.Code != http.StatusOK || repository.user.PhotoURL != "" {
		t.Fatalf("clear status=%d stored=%q", response.Code, repository.user.PhotoURL)
	}
}

func TestUserPhotoRejectsNonAdminAndInvalidInput(t *testing.T) {
	teacher := &repository{user: domain.User{ID: "teacher-id", Role: "docente", Status: "activo"}}
	server := newHTTPServer(teacher)
	cookie := loginCookie(t, server)
	if response := putUserPhoto(server, cookie, "member-7", `{"photo_url":"https://cdn.example.com/a.png"}`); response.Code != http.StatusForbidden {
		t.Fatalf("non-admin status=%d, want 403", response.Code)
	}
	if response := putUserPhoto(newHTTPServer(teacher), nil, "member-7", `{"photo_url":"https://cdn.example.com/a.png"}`); response.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous status=%d, want 401", response.Code)
	}
	admin := &repository{user: domain.User{ID: "admin-id", Role: "admin", Status: "activo"}}
	adminServer := newHTTPServer(admin)
	adminCookie := loginCookie(t, adminServer)
	for _, body := range []string{`{"photo_url":"javascript:alert(1)"}`, `{"photo_url":"/relative.png"}`, `{}`, `{bad json`} {
		if response := putUserPhoto(adminServer, adminCookie, "member-7", body); response.Code != http.StatusBadRequest {
			t.Errorf("body %q: status=%d, want 400", body, response.Code)
		}
	}
	if teacher.photoUserID != "" || admin.photoUserID != "" {
		t.Fatal("rejected request reached the repository")
	}
}
