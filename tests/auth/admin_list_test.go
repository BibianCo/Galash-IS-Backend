package auth_test

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	postgresadapter "github.com/galash-uptc/backend/internal/adapters/outbound/postgres"
	application "github.com/galash-uptc/backend/internal/application/auth"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestAdminListWithoutQueryHasNoResultLimit(t *testing.T) {
	repo := &repository{user: domain.User{ID: "admin", Role: "admin", Status: "activo"}}
	server := newHTTPServer(repo)
	cookie := loginCookie(t, server)
	for _, path := range []string{"/admin/members", "/admin/members?q=%20%20&limit=5"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		request.AddCookie(cookie)
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("status=%d, want 200", response.Code)
		}
		if repo.searchQuery != "" || repo.searchLimit != 0 || !repo.searchInactive {
			t.Fatalf("unfiltered admin request must include all accounts without a limit")
		}
	}
}

func TestAdminListWithoutQueryRequiresAdminSession(t *testing.T) {
	for _, role := range []string{"visitante", "estudiante", "docente"} {
		t.Run(role, func(t *testing.T) {
			repo := &repository{user: domain.User{ID: "user", Role: role, Status: "activo"}}
			server := newHTTPServer(repo)
			anonymous := httptest.NewRecorder()
			server.ServeHTTP(anonymous, httptest.NewRequest(http.MethodGet, "/admin/members", nil))
			if anonymous.Code != http.StatusUnauthorized {
				t.Fatalf("anonymous status=%d", anonymous.Code)
			}
			request := httptest.NewRequest(http.MethodGet, "/admin/members", nil)
			request.AddCookie(loginCookie(t, server))
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			if response.Code != http.StatusForbidden {
				t.Fatalf("non-admin status=%d", response.Code)
			}
			if repo.searchInactive {
				t.Fatal("unauthorized list reached repository")
			}
		})
	}
}

func TestAdminSearchStillFiltersAndBoundsResults(t *testing.T) {
	repo := &repository{user: domain.User{ID: "admin", Role: "admin", Status: "activo"}}
	server := newHTTPServer(repo)
	cookie := loginCookie(t, server)
	service := application.NewService(identityVerifier{}, repo)
	_, err := service.SearchAdminUsers(context.Background(), cookie.Value, " ana ", 100)
	if err != nil {
		t.Fatal(err)
	}
	if repo.searchQuery != "ana" || repo.searchLimit != 50 || !repo.searchInactive {
		t.Fatal("filtered search must retain its filter, bound and inactive accounts")
	}
}

func TestAdminRepositoryListsAllStoredAccounts(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	db, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var count int
	if err = db.QueryRow(ctx, "SELECT count(*) FROM usuarios").Scan(&count); err != nil {
		t.Fatal(err)
	}
	users, err := postgresadapter.NewUserRepository(db).Search(ctx, "", 0, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(users) != count {
		t.Fatalf("listed=%d stored=%d", len(users), count)
	}
}

// Opt-in live smoke test: only its uniquely generated synthetic account/session
// are inserted and removed. Existing accounts and their data are never changed.
func TestLiveAdminEndpointListsAllAccounts(t *testing.T) {
	url, base := os.Getenv("TEST_DATABASE_URL"), os.Getenv("TEST_LIVE_API_URL")
	if url == "" || base == "" {
		t.Skip("live database/API test not configured")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	db, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	raw := make([]byte, 32)
	if _, err = rand.Read(raw); err != nil {
		t.Fatal(err)
	}
	token := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	hashText := base64.RawURLEncoding.EncodeToString(hash[:])
	uid := "admin-list-smoke-" + hex.EncodeToString(raw[:12])
	var id string
	err = db.QueryRow(ctx, `INSERT INTO usuarios(firebase_uid,nombre,email,rol,estado) VALUES($1,'Synthetic smoke test',$2,'admin','activo') RETURNING id`, uid, uid+"@example.invalid").Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanup, cancelCleanup := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelCleanup()
		if _, e := db.Exec(cleanup, "DELETE FROM sesiones WHERE usuario_id=$1 AND token_hash=$2", id, hashText); e != nil {
			t.Error("synthetic session cleanup failed")
		}
		if _, e := db.Exec(cleanup, "DELETE FROM usuarios WHERE id=$1 AND firebase_uid=$2", id, uid); e != nil {
			t.Error("synthetic account cleanup failed")
		}
	}()
	if err = postgresadapter.NewUserRepository(db).Create(ctx, id, hashText); err != nil {
		t.Fatal(err)
	}
	client := &http.Client{Timeout: 10 * time.Second}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/admin/members", nil)
	if err != nil {
		t.Fatal(err)
	}
	request.AddCookie(&http.Cookie{Name: "galash_session", Value: token})
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("live list status=%d", response.StatusCode)
	}
	var result struct {
		Users []domain.AdminProfile `json:"users"`
	}
	if err = json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = db.QueryRow(ctx, "SELECT count(*) FROM usuarios").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if len(result.Users) != count {
		t.Fatalf("live list count=%d database count=%d", len(result.Users), count)
	}
	t.Log("live API returned all stored accounts; no personal data captured")
}
