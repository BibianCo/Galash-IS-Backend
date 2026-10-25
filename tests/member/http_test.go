package member_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpadapter "github.com/galash-uptc/backend/internal/adapters/inbound/http"
	authapp "github.com/galash-uptc/backend/internal/application/auth"
	memberapp "github.com/galash-uptc/backend/internal/application/member"
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
	user, ok := a.users[id]
	if !ok {
		return domain.User{}, errors.New("user not found")
	}
	return user, nil
}
func (a *authStore) FindByEmail(context.Context, string) (domain.User, error) {
	return domain.User{}, errors.New("not used")
}
func (a *authStore) FindByFirebaseUID(ctx context.Context, uid string) (domain.User, error) {
	return a.FindByID(ctx, uid)
}
func (a *authStore) SetPassword(context.Context, string, string) error          { return nil }
func (a *authStore) Search(context.Context, string, int) ([]domain.User, error) { return nil, nil }
func (a *authStore) UpdateRole(context.Context, string, string) error           { return nil }
func (a *authStore) Upsert(context.Context, domain.User) (string, error)        { return "", nil }
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
	store := &authStore{sessions: map[string]string{}, users: map[string]domain.User{
		adminID: {ID: adminID, Role: "admin", Status: "activo"},
		anaID:   {ID: anaID, Role: "estudiante", Status: "activo"},
		otherID: {ID: otherID, Role: "estudiante", Status: "activo"},
	}}
	auth := authapp.NewService(verifier{}, store)
	repo := newRepository()
	handler := httpadapter.NewHandlerWithMembers(auth, healthChecker{}, nil, nil, memberapp.NewService(repo)).Routes()
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

func expect(t *testing.T, name string, response *httptest.ResponseRecorder, status int) {
	t.Helper()
	if response.Code != status {
		t.Errorf("%s: status=%d, want %d: %s", name, response.Code, status, response.Body.String())
	}
}

const memberBody = `{"firebase_uid":"uid-new","name":"Luis","last_name":"Rojas","email":"luis@example.com","role":"estudiante"}`

func TestMemberWritesRequireAdmin(t *testing.T) {
	s := newServer(t)
	writes := []struct{ method, path, body string }{
		{http.MethodPost, "/members", memberBody},
		{http.MethodPatch, "/members/" + anaID, `{"area":"IA"}`},
		{http.MethodPatch, "/members/" + anaID + "/status", `{"status":"inactivo"}`},
	}
	for _, w := range writes {
		expect(t, "anonymous "+w.method+" "+w.path, s.do(t, w.method, w.path, "", w.body), http.StatusUnauthorized)
		expect(t, "student "+w.method+" "+w.path, s.do(t, w.method, w.path, anaID, w.body), http.StatusForbidden)
	}
	if s.repo.created != nil || s.repo.updated != nil || s.repo.status != "" {
		t.Fatal("non-admin write reached repository")
	}
	expect(t, "admin create", s.do(t, http.MethodPost, "/members", adminID, memberBody), http.StatusCreated)
	expect(t, "admin update", s.do(t, http.MethodPatch, "/members/"+anaID, adminID, `{"area":"IA"}`), http.StatusOK)
	expect(t, "admin status", s.do(t, http.MethodPatch, "/members/"+anaID+"/status", adminID, `{"status":"inactivo"}`), http.StatusOK)
}

func TestMemberWriteErrorsMapToClientStatus(t *testing.T) {
	s := newServer(t)
	expect(t, "create missing fields", s.do(t, http.MethodPost, "/members", adminID, `{"name":"Luis"}`), http.StatusBadRequest)
	expect(t, "create bad json", s.do(t, http.MethodPost, "/members", adminID, `{`), http.StatusBadRequest)
	expect(t, "update bad cvlac", s.do(t, http.MethodPatch, "/members/"+anaID, adminID, `{"cvlac":"cvlac"}`), http.StatusBadRequest)
	expect(t, "update missing", s.do(t, http.MethodPatch, "/members/"+missingID, adminID, `{"area":"IA"}`), http.StatusNotFound)
	expect(t, "status invalid", s.do(t, http.MethodPatch, "/members/"+anaID+"/status", adminID, `{"status":"eliminado"}`), http.StatusBadRequest)
	expect(t, "status missing", s.do(t, http.MethodPatch, "/members/"+missingID+"/status", adminID, `{"status":"activo"}`), http.StatusNotFound)
}

func TestGetMemberShowsPrivateFieldsOnlyToAdminOrSelf(t *testing.T) {
	s := newServer(t)
	for _, viewer := range []string{"", otherID} {
		response := s.do(t, http.MethodGet, "/members/"+anaID, viewer, "")
		expect(t, "viewer "+viewer, response, http.StatusOK)
		if strings.Contains(response.Body.String(), "ana@example.com") || strings.Contains(response.Body.String(), `"document_number"`) {
			t.Errorf("viewer %q sees private fields: %s", viewer, response.Body.String())
		}
	}
	for _, viewer := range []string{anaID, adminID} {
		response := s.do(t, http.MethodGet, "/members/"+anaID, viewer, "")
		expect(t, "viewer "+viewer, response, http.StatusOK)
		if !strings.Contains(response.Body.String(), "ana@example.com") {
			t.Errorf("viewer %q misses private fields: %s", viewer, response.Body.String())
		}
	}
	expect(t, "inactive anonymous", s.do(t, http.MethodGet, "/members/"+offID, "", ""), http.StatusNotFound)
	expect(t, "inactive admin", s.do(t, http.MethodGet, "/members/"+offID, adminID, ""), http.StatusOK)
	expect(t, "missing anonymous", s.do(t, http.MethodGet, "/members/"+missingID, "", ""), http.StatusNotFound)
}

func TestGetMemberHidesNonPublicProfileFromVisitors(t *testing.T) {
	s := newServer(t)
	expect(t, "hidden anonymous", s.do(t, http.MethodGet, "/members/"+hiddenID, "", ""), http.StatusNotFound)
	expect(t, "hidden other", s.do(t, http.MethodGet, "/members/"+hiddenID, otherID, ""), http.StatusNotFound)
	expect(t, "hidden admin", s.do(t, http.MethodGet, "/members/"+hiddenID, adminID, ""), http.StatusOK)
}

func TestMemberPublicationsRespectVisibility(t *testing.T) {
	s := newServer(t)
	response := s.do(t, http.MethodGet, "/members/"+anaID+"/publications", "", "")
	expect(t, "public", response, http.StatusOK)
	if !strings.Contains(response.Body.String(), `"title":"Artículo"`) {
		t.Errorf("publications missing: %s", response.Body.String())
	}
	for _, id := range []string{offID, hiddenID, missingID} {
		expect(t, id, s.do(t, http.MethodGet, "/members/"+id+"/publications", "", ""), http.StatusNotFound)
	}
}

func TestPublicMembersPassesDirectoryFilters(t *testing.T) {
	s := newServer(t)
	response := s.do(t, http.MethodGet, "/members?q=an&type=profesor&seedbedId=s1&area=IA&limit=10", "", "")
	expect(t, "directory", response, http.StatusOK)
	want := memberapp.Filter{Query: "an", Role: "profesor", SeedbedID: "s1", Area: "IA", Limit: 10}
	if s.repo.filter != want {
		t.Errorf("filter=%+v, want %+v", s.repo.filter, want)
	}
	if !strings.Contains(response.Body.String(), `"members":[]`) {
		t.Errorf("empty directory must be []: %s", response.Body.String())
	}
}
