package links_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	httpadapter "github.com/galash-uptc/backend/internal/adapters/inbound/http"
	authapp "github.com/galash-uptc/backend/internal/application/auth"
	linkapp "github.com/galash-uptc/backend/internal/application/officiallink"
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
	u, ok := a.users[id]
	if !ok {
		return domain.User{}, errors.New("user not found")
	}
	return u, nil
}
func (a *authStore) FindByEmail(context.Context, string) (domain.User, error) {
	return domain.User{}, errors.New("not used")
}
func (a *authStore) FindByFirebaseUID(ctx context.Context, uid string) (domain.User, error) {
	return a.FindByID(ctx, uid)
}
func (*authStore) SetPassword(context.Context, string, string) error                { return nil }
func (*authStore) Search(context.Context, string, int, bool) ([]domain.User, error) { return nil, nil }
func (*authStore) UpdateRole(context.Context, string, string) error                 { return nil }
func (*authStore) Upsert(context.Context, domain.User) (string, error)              { return "", nil }
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
func (*authStore) Revoke(context.Context, string) error           { return nil }
func (*authStore) RevokeAllForUser(context.Context, string) error { return nil }
func (*authStore) DeleteExpiredSessions(context.Context) error    { return nil }

// memoryRepo mirrors the PostgreSQL adapter: public listing only returns
// active and visible links; activating forces visible, other states hide.
type memoryRepo struct {
	items      map[string]domain.OfficialLink
	includeAll []bool
	next       int
}

func (r *memoryRepo) ListVisible(_ context.Context, category string, includeAll bool) ([]domain.OfficialLink, error) {
	r.includeAll = append(r.includeAll, includeAll)
	out := []domain.OfficialLink{}
	for _, l := range r.items {
		if (includeAll || l.Visible && l.Status == "activo") && (category == "" || l.Category == category) {
			out = append(out, l)
		}
	}
	return out, nil
}
func (r *memoryRepo) Find(_ context.Context, id string) (domain.OfficialLink, error) {
	l, ok := r.items[id]
	if !ok {
		return l, domain.ErrOfficialLinkNotFound
	}
	return l, nil
}
func (r *memoryRepo) Create(_ context.Context, l domain.OfficialLink) (domain.OfficialLink, error) {
	r.next++
	l.ID, l.CreatedAt, l.UpdatedAt = fmt.Sprint(r.next), time.Now(), time.Now()
	r.items[l.ID] = l
	return l, nil
}
func (r *memoryRepo) Update(_ context.Context, id, actor string, l domain.OfficialLink) (domain.OfficialLink, error) {
	if _, ok := r.items[id]; !ok {
		return l, domain.ErrOfficialLinkNotFound
	}
	l.ID, l.UpdatedBy, l.UpdatedAt = id, actor, time.Now()
	r.items[id] = l
	return l, nil
}
func (r *memoryRepo) SetStatus(_ context.Context, id, actor, status string) (domain.OfficialLink, error) {
	l, ok := r.items[id]
	if !ok {
		return l, domain.ErrOfficialLinkNotFound
	}
	l.Status, l.Visible, l.UpdatedBy = status, status == "activo", actor
	r.items[id] = l
	return l, nil
}

type fixture struct {
	server  http.Handler
	repo    *memoryRepo
	cookies map[string]*http.Cookie
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	store := &authStore{sessions: map[string]string{}, users: map[string]domain.User{
		"admin":   {ID: "admin", Role: "Admin", Status: "activo"},
		"teacher": {ID: "teacher", Role: "docente", Status: "activo"},
		"student": {ID: "student", Role: "estudiante", Status: "activo"},
	}}
	repo := &memoryRepo{items: map[string]domain.OfficialLink{
		"1": {ID: "1", Name: "GrupLAC", Description: "Grupo", URL: "https://scienti.minciencias.gov.co", Category: "gruplac", Visible: true, Status: "activo"},
		"2": {ID: "2", Name: "Antiguo", Description: "Obsoleto", URL: "https://old.example.com", Category: "repositorios", Visible: false, Status: "oculto"},
	}, next: 2}
	auth := authapp.NewService(verifier{}, store)
	server := httpadapter.NewHandlerWithOfficialLinks(auth, healthChecker{}, nil, nil, nil, nil, nil, nil, nil, nil, linkapp.NewService(repo, store)).Routes()
	f := fixture{server: server, repo: repo, cookies: map[string]*http.Cookie{}}
	for id := range store.users {
		res := f.do(t, "", http.MethodPost, "/auth/sessions", `{"token":"`+id+`"}`)
		if res.Code != http.StatusCreated {
			t.Fatalf("login %s status=%d body=%s", id, res.Code, res.Body.String())
		}
		f.cookies[id] = res.Result().Cookies()[0]
	}
	return f
}

func (f fixture) do(t *testing.T, user, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if c := f.cookies[user]; c != nil {
		req.AddCookie(c)
	}
	res := httptest.NewRecorder()
	f.server.ServeHTTP(res, req)
	return res
}

func listNames(t *testing.T, res *httptest.ResponseRecorder) []string {
	t.Helper()
	var body struct {
		Links []domain.OfficialLink `json:"links"`
	}
	if res.Code != http.StatusOK || json.Unmarshal(res.Body.Bytes(), &body) != nil {
		t.Fatalf("list status=%d body=%s", res.Code, res.Body.String())
	}
	out := []string{}
	for _, l := range body.Links {
		out = append(out, l.Name)
	}
	return out
}

// PT-LIN-01: visitors and non-admins only see active visible links; Admin sees all.
func TestLinksListScopeAndCategoryFilter(t *testing.T) {
	f := newFixture(t)
	for _, user := range []string{"", "student", "teacher"} {
		if got := listNames(t, f.do(t, user, http.MethodGet, "/links", "")); len(got) != 1 || got[0] != "GrupLAC" {
			t.Fatalf("%q sees %v, want only active visible link", user, got)
		}
	}
	if got := listNames(t, f.do(t, "admin", http.MethodGet, "/links", "")); len(got) != 2 {
		t.Fatalf("admin sees %v, want all states", got)
	}
	if got := listNames(t, f.do(t, "", http.MethodGet, "/links?category=GrupLAC", "")); len(got) != 1 {
		t.Fatalf("category filter (case-insensitive) = %v", got)
	}
	if got := listNames(t, f.do(t, "", http.MethodGet, "/links?category=cvlac", "")); len(got) != 0 {
		t.Fatalf("empty category = %v", got)
	}
	if res := f.do(t, "", http.MethodGet, "/links?category=tiktok", ""); res.Code != http.StatusBadRequest {
		t.Fatalf("invalid category status=%d", res.Code)
	}
}

// PT-LIN-02: writes require an active Admin session.
func TestLinksWritesRequireAdmin(t *testing.T) {
	f := newFixture(t)
	valid := `{"name":"CvLAC","description":"Hoja de vida","url":"https://cvlac.example.com","category":"cvlac"}`
	cases := []struct{ method, path, body string }{
		{http.MethodPost, "/links", valid},
		{http.MethodPatch, "/links/1", `{"name":"X"}`},
		{http.MethodPatch, "/links/1/status", `{"status":"inactivo"}`},
	}
	for _, c := range cases {
		if res := f.do(t, "", c.method, c.path, c.body); res.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s without session status=%d", c.method, c.path, res.Code)
		}
		for _, user := range []string{"student", "teacher"} {
			if res := f.do(t, user, c.method, c.path, c.body); res.Code != http.StatusForbidden {
				t.Fatalf("%s %s as %s status=%d", c.method, c.path, user, res.Code)
			}
		}
	}
	if len(f.repo.items) != 2 || f.repo.items["1"].Name != "GrupLAC" || f.repo.items["1"].Status != "activo" {
		t.Fatalf("unauthorized writes changed data: %+v", f.repo.items)
	}
}

// PT-LIN-02: URL and field validation before saving; author recorded.
func TestLinksCreateValidation(t *testing.T) {
	f := newFixture(t)
	for _, body := range []string{
		`{`,
		`{"name":"","description":"d","url":"https://a.example.com","category":"cvlac"}`,
		`{"name":"n","description":" ","url":"https://a.example.com","category":"cvlac"}`,
		`{"name":"n","description":"d","url":"ftp://a.example.com","category":"cvlac"}`,
		`{"name":"n","description":"d","url":"javascript:alert(1)","category":"cvlac"}`,
		`{"name":"n","description":"d","url":"/relativa","category":"cvlac"}`,
		`{"name":"n","description":"d","url":"https://","category":"cvlac"}`,
		`{"name":"n","description":"d","url":"https://a.example.com","category":"otra"}`,
		`{"name":"n","description":"d","url":"https://a.example.com","category":"cvlac","order":-1}`,
	} {
		if res := f.do(t, "admin", http.MethodPost, "/links", body); res.Code != http.StatusBadRequest {
			t.Fatalf("body %s status=%d", body, res.Code)
		}
	}
	res := f.do(t, "admin", http.MethodPost, "/links", `{"name":"Instagram","description":"Red social","url":"https://instagram.com/galash","category":"Redes Sociales","order":3}`)
	var got domain.OfficialLink
	if res.Code != http.StatusCreated || json.Unmarshal(res.Body.Bytes(), &got) != nil {
		t.Fatalf("create status=%d body=%s", res.Code, res.Body.String())
	}
	if got.Category != "redes-sociales" || got.Status != "activo" || !got.Visible || got.Order != 3 || got.UpdatedBy != "admin" {
		t.Fatalf("created = %+v", got)
	}
	res = f.do(t, "admin", http.MethodPost, "/links", `{"name":"Borrador","description":"d","url":"http://a.example.com","category":"minciencias","visible":false}`)
	if res.Code != http.StatusCreated || json.Unmarshal(res.Body.Bytes(), &got) != nil || got.Visible {
		t.Fatalf("explicit visible=false status=%d body=%s", res.Code, res.Body.String())
	}
}

// PT-LIN-02 / PT-LIN-01: partial update, ordering/visibility and status changes without deletion.
func TestLinksPatchAndStatus(t *testing.T) {
	f := newFixture(t)
	res := f.do(t, "admin", http.MethodPatch, "/links/1", `{"order":7,"visible":false}`)
	var got domain.OfficialLink
	if res.Code != http.StatusOK || json.Unmarshal(res.Body.Bytes(), &got) != nil {
		t.Fatalf("patch status=%d body=%s", res.Code, res.Body.String())
	}
	if got.Name != "GrupLAC" || got.URL != "https://scienti.minciencias.gov.co" || got.Order != 7 || got.Visible || got.UpdatedBy != "admin" {
		t.Fatalf("partial patch lost fields: %+v", got)
	}
	for _, body := range []string{`{"url":"notaurl"}`, `{"category":"x"}`, `{"name":""}`, `{"order":-2}`, `{`} {
		if res := f.do(t, "admin", http.MethodPatch, "/links/1", body); res.Code != http.StatusBadRequest {
			t.Fatalf("patch %s status=%d", body, res.Code)
		}
	}
	if res := f.do(t, "admin", http.MethodPatch, "/links/999", `{"name":"x"}`); res.Code != http.StatusNotFound {
		t.Fatalf("missing link status=%d", res.Code)
	}
	for _, body := range []string{`{"status":"borrado"}`, `{"status":""}`, `{`} {
		if res := f.do(t, "admin", http.MethodPatch, "/links/1/status", body); res.Code != http.StatusBadRequest {
			t.Fatalf("status %s code=%d", body, res.Code)
		}
	}
	if res := f.do(t, "admin", http.MethodPatch, "/links/999/status", `{"status":"activo"}`); res.Code != http.StatusNotFound {
		t.Fatalf("missing link status=%d", res.Code)
	}
	for _, status := range []string{"Inactivo", "oculto", "activo"} {
		res := f.do(t, "admin", http.MethodPatch, "/links/1/status", `{"status":"`+status+`"}`)
		if res.Code != http.StatusOK || json.Unmarshal(res.Body.Bytes(), &got) != nil || got.Status != strings.ToLower(status) {
			t.Fatalf("status %s code=%d body=%s", status, res.Code, res.Body.String())
		}
		public := listNames(t, f.do(t, "", http.MethodGet, "/links", ""))
		if visible := len(public) == 1; visible != (status == "activo") {
			t.Fatalf("after %s public list = %v", status, public)
		}
	}
	if _, ok := f.repo.items["1"]; !ok {
		t.Fatal("status change deleted the link")
	}
}
