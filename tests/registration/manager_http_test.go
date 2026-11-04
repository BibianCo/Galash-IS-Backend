package registration_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"testing"

	httpadapter "github.com/galash-uptc/backend/internal/adapters/inbound/http"
	authapp "github.com/galash-uptc/backend/internal/application/auth"
	application "github.com/galash-uptc/backend/internal/application/registration"
	seedbedapp "github.com/galash-uptc/backend/internal/application/seedbed"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
)

// accounts doubles as user reader and session store; Firebase tokens equal user IDs.
type accounts struct {
	byID     map[string]domain.User
	sessions map[string]string
}

func (a *accounts) FindByID(_ context.Context, id string) (domain.User, error) {
	if u, ok := a.byID[id]; ok {
		return u, nil
	}
	return domain.User{}, errors.New("user not found")
}
func (a *accounts) FindByEmail(context.Context, string) (domain.User, error) {
	return domain.User{}, errors.New("not used")
}
func (a *accounts) FindByFirebaseUID(ctx context.Context, uid string) (domain.User, error) {
	return a.FindByID(ctx, uid)
}
func (a *accounts) SetPassword(context.Context, string, string) error                { return nil }
func (a *accounts) Search(context.Context, string, int, bool) ([]domain.User, error) { return nil, nil }
func (a *accounts) UpdateRole(context.Context, string, string) error                 { return nil }
func (a *accounts) Upsert(context.Context, domain.User) (string, error)              { return "", nil }
func (a *accounts) Create(_ context.Context, userID, hash string) error {
	a.sessions[hash] = userID
	return nil
}
func (a *accounts) FindUserID(_ context.Context, hash string) (string, error) {
	if id, ok := a.sessions[hash]; ok {
		return id, nil
	}
	return "", errors.New("session not found")
}
func (a *accounts) Revoke(context.Context, string) error           { return nil }
func (a *accounts) RevokeAllForUser(context.Context, string) error { return nil }
func (a *accounts) DeleteExpiredSessions(context.Context) error    { return nil }

type tokenVerifier struct{}

func (tokenVerifier) Verify(_ context.Context, token string) (domain.Identity, error) {
	return domain.Identity{UID: token, Email: token + "@example.com"}, nil
}

// seedbeds: "s1" is directed by "director" with "delegate" as delegate; "s2" belongs to "outsider".
type seedbeds struct{ ports.SeedbedRepository }

func (seedbeds) Find(_ context.Context, id string) (domain.Seedbed, error) {
	if id == "s1" || id == "s2" {
		return domain.Seedbed{ID: id}, nil
	}
	return domain.Seedbed{}, domain.ErrSeedbedNotFound
}
func (seedbeds) ManagedSeedbeds(_ context.Context, uid string) ([]string, error) {
	return map[string][]string{"director": {"s1"}, "delegate": {"s1"}, "outsider": {"s2"}}[uid], nil
}

// requests stores one request per seedbed; "legacy" has no seedbed.
type requests struct {
	repository
	scope   []string
	updates int
}

func (r *requests) ListRegistrations(_ context.Context, _, _, _, _, _ string, scope []string) ([]domain.RegistrationRequest, error) {
	r.scope = scope
	return []domain.RegistrationRequest{}, nil
}
func (r *requests) GetRegistrationDetail(_ context.Context, id string) (domain.RegistrationDetail, error) {
	seedbed, ok := map[string]string{"req-s1": "s1", "legacy": ""}[id]
	if !ok {
		return domain.RegistrationDetail{}, domain.ErrRegistrationNotFound
	}
	return domain.RegistrationDetail{RegistrationRequest: domain.RegistrationRequest{ID: id, SeedbedID: seedbed}}, nil
}
func (r *requests) UpdateRegistrationStatus(_ context.Context, id, status, _, _ string) (domain.RegistrationRequest, error) {
	r.updates++
	return domain.RegistrationRequest{ID: id, Status: status}, nil
}

func managerServer(t *testing.T) (func(method, path, user, body string) int, *requests) {
	t.Helper()
	store := &accounts{sessions: map[string]string{}, byID: map[string]domain.User{}}
	for id, role := range map[string]string{"admin": "admin", "director": "docente", "delegate": "estudiante", "outsider": "docente", "student": "estudiante"} {
		store.byID[id] = domain.User{ID: id, FirebaseUID: id, Role: role, Status: "activo"}
	}
	auth := authapp.NewService(tokenVerifier{}, store)
	repo := &requests{}
	handler := httpadapter.NewHandlerWithRegistrations(auth, healthChecker{}, seedbedapp.NewService(seedbeds{}, store), application.NewService(repo)).Routes()
	return func(method, path, user, body string) int {
		request := httptest.NewRequest(method, path, strings.NewReader(body))
		if user != "" {
			login, err := auth.Login(context.Background(), user)
			if err != nil {
				t.Fatalf("login %s: %v", user, err)
			}
			request.Header.Set("Authorization", "Bearer "+login.Token)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response.Code
	}, repo
}

func TestRegistrationRequestsAreHandledBySeedbedManagers(t *testing.T) {
	do, repo := managerServer(t)
	body := `{"estado":"En revisión","observacion":"ok"}`
	for _, c := range []struct {
		name, method, path, user string
		want                     int
	}{
		{"anonymous detail", http.MethodGet, "/admin/registration-requests/req-s1", "", 401},
		{"director detail", http.MethodGet, "/admin/registration-requests/req-s1", "director", 200},
		{"delegate detail", http.MethodGet, "/admin/registration-requests/req-s1", "delegate", 200},
		{"admin detail", http.MethodGet, "/admin/registration-requests/req-s1", "admin", 200},
		{"other seedbed manager", http.MethodGet, "/admin/registration-requests/req-s1", "outsider", 403},
		{"plain student", http.MethodGet, "/admin/registration-requests/req-s1", "student", 403},
		{"legacy without seedbed", http.MethodGet, "/admin/registration-requests/legacy", "director", 403},
		{"legacy admin", http.MethodGet, "/admin/registration-requests/legacy", "admin", 200},
		{"missing", http.MethodGet, "/admin/registration-requests/none", "director", 404},
		{"outsider update", http.MethodPatch, "/admin/registration-requests/req-s1/status", "outsider", 403},
		{"delegate update", http.MethodPatch, "/admin/registration-requests/req-s1/status", "delegate", 200},
	} {
		if got := do(c.method, c.path, c.user, body); got != c.want {
			t.Errorf("%s: status=%d, want %d", c.name, got, c.want)
		}
	}
	if repo.updates != 1 {
		t.Fatalf("updates=%d, want 1 (only the delegate)", repo.updates)
	}
}

func TestRegistrationListIsScopedToManagedSeedbeds(t *testing.T) {
	do, repo := managerServer(t)
	for user, want := range map[string][]string{"delegate": {"s1"}, "student": {}} {
		if got := do(http.MethodGet, "/admin/registration-requests", user, ""); got != 200 || !slices.Equal(repo.scope, want) || repo.scope == nil {
			t.Errorf("%s: status=%d scope=%v, want %v", user, got, repo.scope, want)
		}
	}
	if got := do(http.MethodGet, "/admin/registration-requests", "admin", ""); got != 200 || repo.scope != nil {
		t.Errorf("admin: status=%d scope=%v, want unrestricted", got, repo.scope)
	}
}

func TestCreateRequiresSeedbed(t *testing.T) {
	request := validRequest()
	request.SeedbedID = " "
	if _, _, err := application.NewService(&repository{}).Create(context.Background(), request, "user-id"); err != domain.ErrIncompleteRegistration {
		t.Fatalf("error=%v, want ErrIncompleteRegistration", err)
	}
}
