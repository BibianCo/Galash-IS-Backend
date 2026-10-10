package auth_test

import (
	"context"
	"errors"
	"testing"

	application "github.com/galash-uptc/backend/internal/application/auth"
	"github.com/galash-uptc/backend/internal/domain"
)

type identityVerifier struct {
	identity domain.Identity
	err      error
}

func (v identityVerifier) Verify(context.Context, string) (domain.Identity, error) {
	return v.identity, v.err
}

type repository struct {
	user        domain.User
	createdHash string
	revokedHash string
	sessionID   string
	findErr     error
}

func (r *repository) Upsert(context.Context, domain.User) (string, error) { return r.user.ID, nil }
func (r *repository) FindByID(context.Context, string) (domain.User, error) {
	if r.findErr != nil {
		return domain.User{}, r.findErr
	}
	return r.user, nil
}
func (r *repository) FindByEmail(context.Context, string) (domain.User, error) {
	return r.user, nil
}
func (r *repository) SetPassword(context.Context, string, string) error { return nil }
func (r *repository) Create(_ context.Context, userID, tokenHash string) error {
	if userID != r.user.ID {
		return errors.New("unexpected user")
	}
	r.createdHash = tokenHash
	return nil
}
func (r *repository) FindUserID(_ context.Context, tokenHash string) (string, error) {
	if r.findErr != nil || tokenHash == "" || tokenHash != r.createdHash {
		return "", errors.New("session not found")
	}
	return r.sessionID, nil
}
func (r *repository) Revoke(_ context.Context, tokenHash string) error {
	r.revokedHash = tokenHash
	return nil
}

func validRegistration() application.RegisterCommand {
	return application.RegisterCommand{
		Name: "Ana", LastName: "Pérez", DocumentType: "CC", DocumentNumber: "123",
		Email: "ana@example.com", Role: "estudiante",
	}
}

func TestRegisterPersistsIdentityFields(t *testing.T) {
	repository := &repository{user: domain.User{ID: "user-id"}}
	service := application.NewService(identityVerifier{identity: domain.Identity{UID: "firebase-id", Email: "ana@example.com"}}, repository)

	result, err := service.Register(context.Background(), "firebase-token", validRegistration())
	if err != nil || result.ID != "user-id" {
		t.Fatalf("Register() result=%+v error=%v", result, err)
	}
}

func TestRegisterRejectsInvalidInput(t *testing.T) {
	identity := identityVerifier{identity: domain.Identity{UID: "firebase-id", Email: "ana@example.com"}}
	inputs := []struct {
		name   string
		change func(*application.RegisterCommand)
	}{
		{"missing name", func(c *application.RegisterCommand) { c.Name = "" }},
		{"missing last name", func(c *application.RegisterCommand) { c.LastName = "" }},
		{"email mismatch", func(c *application.RegisterCommand) { c.Email = "other@example.com" }},
		{"invalid role", func(c *application.RegisterCommand) { c.Role = "administrador" }},
	}
	for _, input := range inputs {
		t.Run(input.name, func(t *testing.T) {
			command := validRegistration()
			input.change(&command)
			_, err := application.NewService(identity, &repository{user: domain.User{ID: "user-id"}}).Register(context.Background(), "token", command)
			if !errors.Is(err, domain.ErrIncompleteIdentity) {
				t.Fatalf("Register() error=%v, want ErrIncompleteIdentity", err)
			}
		})
	}
}

func TestRegisterRejectsInvalidFirebaseToken(t *testing.T) {
	_, err := application.NewService(identityVerifier{err: errors.New("invalid token")}, &repository{}).
		Register(context.Background(), "bad-token", validRegistration())
	if !errors.Is(err, domain.ErrInvalidIdentity) {
		t.Fatalf("Register() error=%v, want ErrInvalidIdentity", err)
	}
}

func TestLoginCreatesHashedOpaqueSession(t *testing.T) {
	repository := &repository{user: domain.User{ID: "user-id", Email: "ana@example.com", Status: "activo"}}
	service := application.NewService(identityVerifier{identity: domain.Identity{UID: "firebase-id", Email: "ana@example.com"}}, repository)

	result, err := service.Login(context.Background(), "firebase-token")
	if err != nil || result.Token == "" {
		t.Fatalf("Login() result=%+v error=%v", result, err)
	}
	if repository.createdHash == "" || repository.createdHash == result.Token {
		t.Fatalf("session token was not stored as a hash: %q", repository.createdHash)
	}
}

func TestLoginRejectsInvalidAndInactiveUsers(t *testing.T) {
	tests := []struct {
		name     string
		verifier identityVerifier
		user     domain.User
		want     error
	}{
		{"invalid token", identityVerifier{err: errors.New("invalid")}, domain.User{}, domain.ErrInvalidCredentials},
		{"inactive user", identityVerifier{identity: domain.Identity{UID: "uid", Email: "ana@example.com"}}, domain.User{ID: "id", Status: "inactivo"}, domain.ErrInactiveUser},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository := &repository{user: test.user}
			_, err := application.NewService(test.verifier, repository).Login(context.Background(), "token")
			if !errors.Is(err, test.want) {
				t.Fatalf("Login() error=%v, want %v", err, test.want)
			}
		})
	}
}

func TestCurrentUserAndLogoutUseSessionHash(t *testing.T) {
	sessionRepo := &repository{user: domain.User{ID: "user-id", Name: "Ana", Status: "activo"}, sessionID: "user-id", createdHash: "session-hash"}
	service := application.NewService(identityVerifier{}, sessionRepo)

	// Use a token whose hash is accepted by the fake repository through a login-created session.
	loginRepository := &repository{user: sessionRepo.user}
	loginService := application.NewService(identityVerifier{identity: domain.Identity{UID: "uid", Email: "ana@example.com"}}, loginRepository)
	login, err := loginService.Login(context.Background(), "firebase-token")
	if err != nil {
		t.Fatal(err)
	}
	sessionRepo.createdHash = loginRepository.createdHash

	user, err := service.CurrentUser(context.Background(), login.Token)
	if err != nil || user.ID != "user-id" {
		t.Fatalf("CurrentUser() user=%+v error=%v", user, err)
	}
	if err := service.Logout(context.Background(), login.Token); err != nil {
		t.Fatal(err)
	}
	if sessionRepo.revokedHash == "" {
		t.Fatal("Logout() did not revoke session")
	}
}
