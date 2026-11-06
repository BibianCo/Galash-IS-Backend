package auth_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	httpadapter "github.com/galash-uptc/backend/internal/adapters/inbound/http"
	postgresadapter "github.com/galash-uptc/backend/internal/adapters/outbound/postgres"
	application "github.com/galash-uptc/backend/internal/application/auth"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type importedRepository struct {
	*repository
	imported  []domain.User
	claimedID string
	claimed   domain.User
}

func (r *importedRepository) ListImported(context.Context) ([]domain.User, error) {
	return r.imported, nil
}

func (r *importedRepository) ClaimImported(_ context.Context, id string, user domain.User) (string, error) {
	r.claimedID, r.claimed = id, user
	return id, nil
}

func sgiImported() []domain.User {
	return []domain.User{
		{ID: "u-ana", Name: "Ana María Rojas Ejemplo", Role: "estudiante"},
		{ID: "u-lucia", Name: "Lucía Peña Ejemplo", Role: "docente"},
		{ID: "u-tomas", Name: "Ejemplo Muñoz Tomás Andrés", Role: "estudiante"},
	}
}

func uptcIdentity(email string, verified bool) domain.Identity {
	return domain.Identity{UID: "firebase-new", Email: email, EmailVerified: verified}
}

func TestMatchImportedUsesVerifiedInstitutionalEmail(t *testing.T) {
	cases := []struct {
		email    string
		verified bool
		want     string
	}{
		{"ana.rojas05@uptc.edu.co", true, "u-ana"},
		{"Ana.Rojas@UPTC.edu.co", true, "u-ana"},
		{"lucia.pena@uptc.edu.co", true, "u-lucia"},
		{"tomas.ejemplo12@uptc.edu.co", true, "u-tomas"},
		{"tomas.munoz@uptc.edu.co", true, "u-tomas"},
		{"ana.rojas05@uptc.edu.co", false, ""},
		{"ana.rojas05@gmail.com", true, ""},
		{"ana.rojas05@mail.uptc.edu.co", true, ""},
		{"ana.rojas123@uptc.edu.co", true, ""},
		{"ana.perez@uptc.edu.co", true, ""},
		{"anarojas@uptc.edu.co", true, ""},
	}
	for _, c := range cases {
		repo := &importedRepository{repository: &repository{}, imported: sgiImported()}
		service := application.NewService(identityVerifier{identity: uptcIdentity(c.email, c.verified)}, repo)
		got, err := service.MatchImported(context.Background(), "firebase-token")
		if err != nil {
			t.Fatalf("%s: MatchImported() error=%v", c.email, err)
		}
		if c.want == "" && len(got) != 0 || c.want != "" && (len(got) != 1 || got[0].ID != c.want) {
			t.Fatalf("%s (verified=%v): matches=%+v, want %q", c.email, c.verified, got, c.want)
		}
	}
}

func TestMatchImportedRejectsInvalidToken(t *testing.T) {
	repo := &importedRepository{repository: &repository{}, imported: sgiImported()}
	service := application.NewService(identityVerifier{err: errors.New("bad token")}, repo)
	if _, err := service.MatchImported(context.Background(), "x"); !errors.Is(err, domain.ErrInvalidIdentity) {
		t.Fatalf("MatchImported() error=%v, want ErrInvalidIdentity", err)
	}
}

func claimRegistration(claimID string) application.RegisterCommand {
	command := validRegistration()
	command.Name, command.LastName, command.Email, command.ClaimID = "Ana María", "Rojas Ejemplo", "ana.rojas05@uptc.edu.co", claimID
	return command
}

func TestRegisterClaimsMatchingImportedAccount(t *testing.T) {
	repo := &importedRepository{repository: &repository{user: domain.User{ID: "new-id"}}, imported: sgiImported()}
	service := application.NewService(identityVerifier{identity: uptcIdentity("ana.rojas05@uptc.edu.co", true)}, repo)

	result, err := service.Register(context.Background(), "firebase-token", claimRegistration("u-ana"))
	if err != nil || result.ID != "u-ana" {
		t.Fatalf("Register() result=%+v error=%v", result, err)
	}
	if repo.claimedID != "u-ana" || repo.claimed.FirebaseUID != "firebase-new" || repo.claimed.Name != "Ana María" {
		t.Fatalf("ClaimImported() id=%q user=%+v", repo.claimedID, repo.claimed)
	}
	if repo.upsertedUser.FirebaseUID != "" {
		t.Fatalf("Upsert() must not run on claim, got %+v", repo.upsertedUser)
	}
}

func TestRegisterRejectsClaimThatDoesNotMatchIdentity(t *testing.T) {
	for _, identity := range []domain.Identity{
		uptcIdentity("ana.rojas05@uptc.edu.co", true), // matches u-ana, not u-lucia
		uptcIdentity("ana.rojas05@uptc.edu.co", false),
	} {
		repo := &importedRepository{repository: &repository{}, imported: sgiImported()}
		service := application.NewService(identityVerifier{identity: identity}, repo)
		claimID := "u-lucia"
		if !identity.EmailVerified {
			claimID = "u-ana"
		}
		if _, err := service.Register(context.Background(), "firebase-token", claimRegistration(claimID)); !errors.Is(err, domain.ErrClaimNotAllowed) {
			t.Fatalf("Register(claim %s, verified=%v) error=%v, want ErrClaimNotAllowed", claimID, identity.EmailVerified, err)
		}
		if repo.claimedID != "" {
			t.Fatalf("ClaimImported() must not run, got %q", repo.claimedID)
		}
	}
}

func TestClaimImportedKeepsRoleAndRejectsSecondClaim(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	defer db.Close()

	suffix := time.Now().UnixNano()
	importedUID := fmt.Sprintf("sgi-uptc:test-%d", suffix)
	realUID := fmt.Sprintf("claim-test-%d", suffix)
	defer func() {
		_, _ = db.Exec(ctx, `DELETE FROM usuarios WHERE firebase_uid IN ($1, $2)`, importedUID, realUID)
	}()
	var id string
	if err := db.QueryRow(ctx, `INSERT INTO usuarios (firebase_uid, nombre, email, rol) VALUES ($1, 'Test Claim Docente', $2, 'docente') RETURNING id`,
		importedUID, fmt.Sprintf("sgi-%d@sin-correo.invalid", suffix)).Scan(&id); err != nil {
		t.Fatalf("insert imported user: %v", err)
	}

	repo := postgresadapter.NewUserRepository(db)
	imported, err := repo.ListImported(ctx)
	if err != nil {
		t.Fatalf("ListImported(): %v", err)
	}
	found := false
	for _, u := range imported {
		found = found || u.ID == id
	}
	if !found {
		t.Fatalf("ListImported() does not include %s", id)
	}

	user := domain.User{FirebaseUID: realUID, Name: "Test", LastName: "Claim", Email: realUID + "@uptc.edu.co"}
	if got, err := repo.ClaimImported(ctx, id, user); err != nil || got != id {
		t.Fatalf("ClaimImported() id=%q error=%v", got, err)
	}
	var uid, role, email string
	if err := db.QueryRow(ctx, `SELECT firebase_uid, rol, email FROM usuarios WHERE id = $1`, id).Scan(&uid, &role, &email); err != nil {
		t.Fatalf("read claimed user: %v", err)
	}
	if uid != realUID || role != "docente" || email != user.Email {
		t.Fatalf("claimed user uid=%q role=%q email=%q", uid, role, email)
	}
	if _, err := repo.ClaimImported(ctx, id, user); !errors.Is(err, domain.ErrClaimNotAllowed) {
		t.Fatalf("second ClaimImported() error=%v, want ErrClaimNotAllowed", err)
	}
}

func TestRegisterMatchesEndpointReturnsNameAndRoleOnly(t *testing.T) {
	repo := &importedRepository{repository: &repository{}, imported: sgiImported()}
	service := application.NewService(identityVerifier{identity: uptcIdentity("lucia.pena@uptc.edu.co", true)}, repo)
	server := httpadapter.NewHandler(service, healthChecker{}).Routes()

	request := httptest.NewRequest(http.MethodGet, "/auth/register/matches", nil)
	request.Header.Set("Authorization", "Bearer firebase-token")
	response := httptest.NewRecorder()
	server.ServeHTTP(response, request)
	want := `{"matches":[{"id":"u-lucia","name":"Lucía Peña Ejemplo","role":"docente"}]}`
	if response.Code != http.StatusOK || strings.TrimSpace(response.Body.String()) != want {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}

	unauthenticated := httptest.NewRecorder()
	server.ServeHTTP(unauthenticated, httptest.NewRequest(http.MethodGet, "/auth/register/matches", nil))
	if unauthenticated.Code != http.StatusUnauthorized {
		t.Fatalf("without Bearer status=%d, want 401", unauthenticated.Code)
	}
}
