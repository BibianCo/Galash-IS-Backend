package content_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"
	"time"

	httpadapter "github.com/galash-uptc/backend/internal/adapters/inbound/http"
	authapp "github.com/galash-uptc/backend/internal/application/auth"
	contentapp "github.com/galash-uptc/backend/internal/application/content"
	"github.com/galash-uptc/backend/internal/domain"
)

type healthChecker struct{}

func (healthChecker) Ping(context.Context) error { return nil }

// verifier maps each Firebase token to the user with the same ID.
type verifier struct{}

func (verifier) Verify(_ context.Context, token string) (domain.Identity, error) {
	return domain.Identity{UID: token, Email: token + "@example.com"}, nil
}

type users map[string]domain.User

func (u users) FindByID(_ context.Context, id string) (domain.User, error) {
	user, ok := u[id]
	if !ok {
		return domain.User{}, errors.New("user not found")
	}
	return user, nil
}
func (u users) FindByEmail(context.Context, string) (domain.User, error) {
	return domain.User{}, errors.New("not used")
}
func (u users) FindByFirebaseUID(ctx context.Context, uid string) (domain.User, error) {
	return u.FindByID(ctx, uid)
}
func (u users) SetPassword(context.Context, string, string) error          { return nil }
func (u users) Search(context.Context, string, int) ([]domain.User, error) { return nil, nil }
func (u users) UpdateRole(context.Context, string, string) error           { return nil }

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

// repository mirrors the PostgreSQL adapter (slug order, published filter) in memory.
type repository struct {
	pages map[string]domain.ContentPage
	next  int
}

func (r *repository) List(_ context.Context, drafts bool) ([]domain.ContentPage, error) {
	out := []domain.ContentPage{}
	for _, p := range r.pages {
		if drafts || p.Status == "publicado" {
			out = append(out, p)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Slug < out[j].Slug })
	return out, nil
}
func (r *repository) FindBySlug(_ context.Context, slug string) (domain.ContentPage, error) {
	for _, p := range r.pages {
		if p.Slug == slug && p.Status == "publicado" {
			return p, nil
		}
	}
	return domain.ContentPage{}, domain.ErrContentNotFound
}
func (r *repository) Find(_ context.Context, id string) (domain.ContentPage, error) {
	p, ok := r.pages[id]
	if !ok {
		return p, domain.ErrContentNotFound
	}
	return p, nil
}
func (r *repository) Create(_ context.Context, p domain.ContentPage) (domain.ContentPage, error) {
	r.next++
	p.ID, p.CreatedAt, p.UpdatedAt = fmt.Sprint(r.next), time.Now(), time.Now()
	r.pages[p.ID] = p
	return p, nil
}
func (r *repository) Update(_ context.Context, id, title, content, slug string) (domain.ContentPage, error) {
	p := r.pages[id]
	p.Title, p.Content, p.Slug, p.UpdatedAt = title, content, slug, time.Now()
	r.pages[id] = p
	return p, nil
}
func (r *repository) Publish(_ context.Context, id string) (domain.ContentPage, error) {
	p := r.pages[id]
	now := time.Now()
	p.Status, p.PublishedAt, p.UpdatedAt = "publicado", &now, now
	r.pages[id] = p
	return p, nil
}

type server struct {
	handler http.Handler
	auth    *authapp.Service
	repo    *repository
}

func newServer(t *testing.T) server {
	t.Helper()
	u := users{
		"admin":   {ID: "admin", Role: "admin", Status: "activo"},
		"teacher": {ID: "teacher", Role: "docente", Status: "activo"},
		"student": {ID: "student", Role: "estudiante", Status: "activo"},
	}
	repo := &repository{pages: map[string]domain.ContentPage{}}
	auth := authapp.NewService(verifier{}, &authStore{users: u, sessions: map[string]string{}})
	handler := httpadapter.NewHandlerWithContent(auth, healthChecker{}, nil, nil, nil, nil, nil, contentapp.NewService(repo, u)).Routes()
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

func expect(t *testing.T, name string, response *httptest.ResponseRecorder, status int) domain.ContentPage {
	t.Helper()
	if response.Code != status {
		t.Errorf("%s: status=%d, want %d: %s", name, response.Code, status, response.Body.String())
	}
	var p domain.ContentPage
	_ = json.Unmarshal(response.Body.Bytes(), &p)
	return p
}

func listSlugs(t *testing.T, s server, path, userID string) string {
	t.Helper()
	response := s.do(t, http.MethodGet, path, userID, "")
	expect(t, "list "+path, response, http.StatusOK)
	var body struct{ Pages []domain.ContentPage }
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	slugs := []string{}
	for _, p := range body.Pages {
		slugs = append(slugs, p.Slug)
	}
	return strings.Join(slugs, ",")
}

func page(slug, content string) string {
	b, _ := json.Marshal(map[string]string{"slug": slug, "title": "Título " + slug, "content": content})
	return string(b)
}

// PT-INFO-01 (backend): visitors browse every published section by slug; drafts stay hidden.
func TestVisitorBrowsesPublishedSections(t *testing.T) {
	s := newServer(t)
	sections := []string{"historia", "objetivos", "mision", "vision", "areas-de-conocimiento"}
	for _, slug := range sections {
		p := expect(t, "create "+slug, s.do(t, http.MethodPost, "/admin/content/pages", "admin", page(slug, "<p>Texto "+slug+"</p>")), http.StatusCreated)
		expect(t, "publish "+slug, s.do(t, http.MethodPost, "/admin/content/pages/"+p.ID+"/publish", "admin", ""), http.StatusOK)
	}
	expect(t, "draft", s.do(t, http.MethodPost, "/admin/content/pages", "admin", page("borrador-interno", "<p>x</p>")), http.StatusCreated)

	want := "areas-de-conocimiento,historia,mision,objetivos,vision"
	for _, user := range []string{"", "student"} {
		if got := listSlugs(t, s, "/content/pages", user); got != want {
			t.Errorf("public list as %q = %q, want %q", user, got, want)
		}
	}
	for _, slug := range sections {
		got := expect(t, "detail "+slug, s.do(t, http.MethodGet, "/content/pages/"+slug, "", ""), http.StatusOK)
		if got.Slug != slug || got.Content != "<p>Texto "+slug+"</p>" || got.Status != "publicado" || got.PublishedAt == nil {
			t.Errorf("detail %s = %+v", slug, got)
		}
	}
	expect(t, "draft detail", s.do(t, http.MethodGet, "/content/pages/borrador-interno", "", ""), http.StatusNotFound)
	expect(t, "unknown slug", s.do(t, http.MethodGet, "/content/pages/no-existe", "", ""), http.StatusNotFound)
	expect(t, "invalid slug", s.do(t, http.MethodGet, "/content/pages/Historia_1", "", ""), http.StatusNotFound)
}

// PT-INFO-02: save draft, publish and query; author, date, safe format and rejection without permission.
func TestDraftPublishAuthorshipAndPermissions(t *testing.T) {
	s := newServer(t)
	body := page("mision", "<p>Misión</p>")
	expect(t, "anonymous create", s.do(t, http.MethodPost, "/admin/content/pages", "", body), http.StatusUnauthorized)
	expect(t, "teacher create", s.do(t, http.MethodPost, "/admin/content/pages", "teacher", body), http.StatusForbidden)
	expect(t, "student create", s.do(t, http.MethodPost, "/admin/content/pages", "student", body), http.StatusForbidden)
	expect(t, "anonymous admin list", s.do(t, http.MethodGet, "/admin/content/pages", "", ""), http.StatusUnauthorized)
	expect(t, "teacher admin list", s.do(t, http.MethodGet, "/admin/content/pages", "teacher", ""), http.StatusForbidden)

	for name, invalid := range map[string]string{
		"missing title":   `{"slug":"mision","content":"<p>x</p>"}`,
		"blank title":     `{"slug":"mision","title":"  ","content":"<p>x</p>"}`,
		"missing slug":    `{"title":"Misión","content":"<p>x</p>"}`,
		"bad slug":        `{"slug":"Misión Grupo","title":"Misión","content":"<p>x</p>"}`,
		"missing content": `{"slug":"mision","title":"Misión"}`,
		"only unsafe":     `{"slug":"mision","title":"Misión","content":"<img src=x onerror=alert(1)>"}`,
		"invalid json":    `{`,
	} {
		expect(t, name, s.do(t, http.MethodPost, "/admin/content/pages", "admin", invalid), http.StatusBadRequest)
	}
	if len(s.repo.pages) != 0 {
		t.Fatalf("rejected requests persisted %d pages", len(s.repo.pages))
	}

	draft := expect(t, "admin create", s.do(t, http.MethodPost, "/admin/content/pages", "admin", body), http.StatusCreated)
	if draft.Status != "borrador" || draft.AuthorID != "admin" || draft.PublishedAt != nil || draft.UpdatedAt.IsZero() {
		t.Errorf("draft = %+v", draft)
	}
	if got := listSlugs(t, s, "/admin/content/pages", "admin"); got != "mision" {
		t.Errorf("admin list = %q", got)
	}
	if got := listSlugs(t, s, "/content/pages", ""); got != "" {
		t.Errorf("draft leaked to public list: %q", got)
	}

	path := "/admin/content/pages/" + draft.ID
	expect(t, "teacher edit", s.do(t, http.MethodPatch, path, "teacher", `{"title":"x"}`), http.StatusForbidden)
	expect(t, "anonymous edit", s.do(t, http.MethodPatch, path, "", `{"title":"x"}`), http.StatusUnauthorized)
	expect(t, "invalid patch slug", s.do(t, http.MethodPatch, path, "admin", `{"slug":"MAL slug"}`), http.StatusBadRequest)
	expect(t, "unknown edit", s.do(t, http.MethodPatch, "/admin/content/pages/999", "admin", `{"title":"x"}`), http.StatusNotFound)

	before := s.repo.pages[draft.ID].UpdatedAt
	time.Sleep(time.Millisecond)
	edited := expect(t, "partial edit", s.do(t, http.MethodPatch, path, "admin", `{"title":"Misión GALASH"}`), http.StatusOK)
	if edited.Title != "Misión GALASH" || edited.Content != "<p>Misión</p>" || edited.Slug != "mision" || edited.Status != "borrador" || !edited.UpdatedAt.After(before) {
		t.Errorf("partial edit = %+v", edited)
	}

	expect(t, "teacher publish", s.do(t, http.MethodPost, path+"/publish", "teacher", ""), http.StatusForbidden)
	expect(t, "anonymous publish", s.do(t, http.MethodPost, path+"/publish", "", ""), http.StatusUnauthorized)
	published := expect(t, "admin publish", s.do(t, http.MethodPost, path+"/publish", "admin", ""), http.StatusOK)
	if published.Status != "publicado" || published.PublishedAt == nil || published.AuthorID != "admin" {
		t.Errorf("published = %+v", published)
	}
	expect(t, "publish twice", s.do(t, http.MethodPost, path+"/publish", "admin", ""), http.StatusBadRequest)
	expect(t, "unknown publish", s.do(t, http.MethodPost, "/admin/content/pages/999/publish", "admin", ""), http.StatusNotFound)

	got := expect(t, "public detail", s.do(t, http.MethodGet, "/content/pages/mision", "", ""), http.StatusOK)
	if got.Title != "Misión GALASH" || got.AuthorID != "admin" || got.UpdatedAt.IsZero() {
		t.Errorf("public detail = %+v", got)
	}
}

// PT-INFO-02: published content keeps only the safe formatting vocabulary.
func TestContentSanitization(t *testing.T) {
	for name, tc := range map[string]struct{ in, want string }{
		"allowed tags":       {`<h2>Visión</h2><p><strong>a</strong> <em>b</em></p><ul><li>c</li></ul><ol><li>d</li></ol>`, `<h2>Visión</h2><p><strong>a</strong> <em>b</em></p><ul><li>c</li></ul><ol><li>d</li></ol>`},
		"script removed":     {`<p>ok</p><script>alert(1)</script>`, `<p>ok</p>alert(1)`},
		"event handler":      {`<p onclick="alert(1)">x</p>`, `<p>x</p>`},
		"javascript href":    {`<a href="javascript:alert(1)">x</a>`, `<a>x</a>`},
		"https href kept":    {`<a href="https://uptc.edu.co" target="_blank">x</a>`, `<a href="https://uptc.edu.co">x</a>`},
		"href quote escaped": {`<a href='https://x.co/"onmouseover="a'>x</a>`, `<a href="https://x.co/&#34;onmouseover=&#34;a">x</a>`},
		"text escaped":       {`<p>1 &lt; 2 &amp; &lt;b&gt;</p>`, `<p>1 &lt; 2 &amp; &lt;b&gt;</p>`},
		"iframe and style":   {`<iframe src="https://x"></iframe><style>p{}</style><p>y</p>`, `p{}<p>y</p>`},
		"self-closing br":    {`<p>a<br/>b</p>`, "<p>a<br>\nb</p>"},
	} {
		if got := contentapp.Sanitize(tc.in); got != tc.want {
			t.Errorf("%s: Sanitize(%q) = %q, want %q", name, tc.in, got, tc.want)
		}
	}

	s := newServer(t)
	p := expect(t, "create unsafe", s.do(t, http.MethodPost, "/admin/content/pages", "admin", page("historia", `<p onclick="x">Historia</p><script>steal()</script>`)), http.StatusCreated)
	if strings.Contains(p.Content, "<script") || strings.Contains(p.Content, "onclick") {
		t.Errorf("stored unsafe content: %q", p.Content)
	}
	p = expect(t, "edit unsafe", s.do(t, http.MethodPatch, "/admin/content/pages/"+p.ID, "admin", `{"content":"<p>Nueva</p><img src=x onerror=alert(1)>"}`), http.StatusOK)
	if p.Content != "<p>Nueva</p>" {
		t.Errorf("edited content = %q", p.Content)
	}
}
