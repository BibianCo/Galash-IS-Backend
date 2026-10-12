package auth_test

import (
	"context"
	"errors"
	"testing"

	application "github.com/galash-uptc/backend/internal/application/auth"
	"github.com/galash-uptc/backend/internal/domain"
	"golang.org/x/crypto/bcrypt"
)

type identityVerifier struct {
	identity domain.Identity
	err      error
}

func (v identityVerifier) Verify(context.Context, string) (domain.Identity, error) {
	return v.identity, v.err
}

type repository struct {
	user                 domain.User
	upsertedUser         domain.User
	createdHash          string
	revokedHash          string
	sessionID            string
	findErr              error
	findUIDErr           error
	findEmailErr         error
	recoveryHash         string
	recoveryUserID       string
	consumedRecoveryHash string
	recoveryConsumed     bool
	recoveryConsumeErr   error
	passwordHash         string
	updatedUID           string
	updatedRole          string
}

func (r *repository) Upsert(_ context.Context, user domain.User) (string, error) {
	r.upsertedUser = user
	return r.user.ID, nil
}
func (r *repository) FindByID(context.Context, string) (domain.User, error) {
	if r.findErr != nil {
		return domain.User{}, r.findErr
	}
	return r.user, nil
}

func (r *repository) FindByEmail(context.Context, string) (domain.User, error) {
	if r.findEmailErr != nil {
		return domain.User{}, r.findEmailErr
	}
	return r.user, nil
}

func (r *repository) FindByFirebaseUID(context.Context, string) (domain.User, error) {
	if r.findUIDErr != nil {
		return domain.User{}, r.findUIDErr
	}
	return r.user, nil
}

func (r *repository) SetPassword(_ context.Context, _ string, hash string) error {
	r.passwordHash = hash
	return nil
}
func (r *repository) Search(context.Context, string, int) ([]domain.User, error) {
	return []domain.User{r.user}, nil
}
func (r *repository) UpdateRole(_ context.Context, uid, role string) error {
	r.updatedUID, r.updatedRole = uid, role
	return nil
}

func (r *repository) Create(_ context.Context, userID, tokenHash string) error {
	if userID != r.user.ID {
		return errors.New("unexpected user")
	}
	r.createdHash = tokenHash
	r.sessionID = userID
	return nil
}
func (r *repository) FindUserID(_ context.Context, tokenHash string) (string, error) {
	if r.findErr != nil || tokenHash == "" || tokenHash != r.createdHash || tokenHash == r.revokedHash {
		return "", errors.New("session not found")
	}
	return r.sessionID, nil
}
func (r *repository) Revoke(_ context.Context, tokenHash string) error {
	r.revokedHash = tokenHash
	return nil
}

func (r *repository) CreateRecovery(_ context.Context, userID, tokenHash string) error {
	r.recoveryUserID = userID
	r.recoveryHash = tokenHash
	return nil
}

func (r *repository) Consume(_ context.Context, tokenHash string) (string, error) {
	if r.recoveryConsumeErr != nil || tokenHash == "" || r.recoveryConsumed || (r.recoveryHash != "" && tokenHash != r.recoveryHash) {
		return "", errors.New("recovery token not found, expired, or already used")
	}
	r.recoveryHash = tokenHash
	r.consumedRecoveryHash = tokenHash
	r.recoveryConsumed = true
	return r.recoveryUserID, nil
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
	want := domain.User{
		FirebaseUID: "firebase-id", Name: "Ana", LastName: "Pérez", Email: "ana@example.com",
		DocumentType: "CC", DocumentNumber: "123", Role: "estudiante", Status: "activo",
	}
	if repository.upsertedUser != want {
		t.Fatalf("Upsert() user=%+v, want %+v", repository.upsertedUser, want)
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

func TestRegisterRejectsMissingProfileFields(t *testing.T) {
	identity := identityVerifier{identity: domain.Identity{UID: "firebase-id", Email: "ana@example.com"}}
	inputs := []struct {
		name   string
		change func(*application.RegisterCommand)
	}{
		{"missing document type", func(c *application.RegisterCommand) { c.DocumentType = "" }},
		{"missing document number", func(c *application.RegisterCommand) { c.DocumentNumber = "" }},
		{"missing email", func(c *application.RegisterCommand) { c.Email = "" }},
		{"missing role", func(c *application.RegisterCommand) { c.Role = "" }},
	}
	for _, input := range inputs {
		t.Run(input.name, func(t *testing.T) {
			repository := &repository{user: domain.User{ID: "user-id"}}
			command := validRegistration()
			input.change(&command)

			_, err := application.NewService(identity, repository).Register(context.Background(), "token", command)
			if !errors.Is(err, domain.ErrIncompleteIdentity) {
				t.Fatalf("Register() error=%v, want ErrIncompleteIdentity", err)
			}
			if repository.upsertedUser != (domain.User{}) {
				t.Fatalf("invalid registration was persisted: %+v", repository.upsertedUser)
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

func TestLoginRejectsUnregisteredFirebaseUserWithoutUpsert(t *testing.T) {
	repository := &repository{findUIDErr: errors.New("user not found")}
	service := application.NewService(identityVerifier{identity: domain.Identity{UID: "firebase-id", Email: "ana@example.com"}}, repository)

	_, err := service.Login(context.Background(), "firebase-token")
	if !errors.Is(err, domain.ErrUserNotRegistered) {
		t.Fatalf("Login() error=%v, want ErrUserNotRegistered", err)
	}
	if repository.upsertedUser != (domain.User{}) {
		t.Fatalf("unregistered identity was upserted: %+v", repository.upsertedUser)
	}
	if repository.createdHash != "" {
		t.Fatal("session was created for an unregistered identity")
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

func TestLogoutInvalidatesCurrentUserSession(t *testing.T) {
	repository := &repository{user: domain.User{ID: "user-id", Name: "Ana", Status: "activo"}}
	service := application.NewService(identityVerifier{identity: domain.Identity{UID: "uid", Email: "ana@example.com"}}, repository)

	login, err := service.Login(context.Background(), "firebase-token")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CurrentUser(context.Background(), login.Token); err != nil {
		t.Fatalf("CurrentUser() before logout error=%v", err)
	}
	if err := service.Logout(context.Background(), login.Token); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CurrentUser(context.Background(), login.Token); !errors.Is(err, domain.ErrInvalidSession) {
		t.Fatalf("CurrentUser() after logout error=%v, want ErrInvalidSession", err)
	}
}

func TestRequestPasswordRecoveryDoesNotRevealUnknownEmail(t *testing.T) {
	known := &repository{user: domain.User{ID: "user-id", Email: "ana@example.com"}}
	unknown := &repository{findEmailErr: errors.New("user not found")}

	knownErr := application.NewService(identityVerifier{}, known).RequestPasswordRecovery(context.Background(), "ana@example.com")
	unknownErr := application.NewService(identityVerifier{}, unknown).RequestPasswordRecovery(context.Background(), "missing@example.com")
	if knownErr != nil || unknownErr != nil {
		t.Fatalf("public recovery errors differ: known=%v unknown=%v", knownErr, unknownErr)
	}
	if known.recoveryUserID != "user-id" || known.recoveryHash == "" || known.recoveryHash == "ana@example.com" {
		t.Fatalf("recovery token was not stored as a hash for the matching user: %+v", known)
	}
	if unknown.recoveryHash != "" {
		t.Fatal("unknown email unexpectedly created a recovery token")
	}
}

func TestResetPasswordHashesPasswordAndConsumesTokenOnce(t *testing.T) {
	repository := &repository{recoveryUserID: "user-id"}
	service := application.NewService(identityVerifier{}, repository)

	if err := service.ResetPassword(context.Background(), "recovery-token", "new-password-123"); err != nil {
		t.Fatalf("ResetPassword() error=%v", err)
	}
	if repository.consumedRecoveryHash == "" || repository.consumedRecoveryHash == "recovery-token" {
		t.Fatal("recovery token was not passed to the repository as a hash")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(repository.passwordHash), []byte("new-password-123")); err != nil {
		t.Fatalf("stored password is not a bcrypt hash: %v", err)
	}
	if err := service.ResetPassword(context.Background(), "recovery-token", "new-password-123"); !errors.Is(err, domain.ErrInvalidRecoveryToken) {
		t.Fatalf("second ResetPassword() error=%v, want ErrInvalidRecoveryToken", err)
	}
}

func TestResetPasswordRejectsShortPasswordAndInvalidToken(t *testing.T) {
	t.Run("short password", func(t *testing.T) {
		repository := &repository{recoveryUserID: "user-id"}
		err := application.NewService(identityVerifier{}, repository).ResetPassword(context.Background(), "recovery-token", "short")
		if !errors.Is(err, domain.ErrInvalidRecoveryToken) {
			t.Fatalf("ResetPassword() error=%v, want ErrInvalidRecoveryToken", err)
		}
		if repository.recoveryConsumed {
			t.Fatal("short password consumed the recovery token")
		}
	})

	t.Run("repository rejects token", func(t *testing.T) {
		repository := &repository{recoveryConsumeErr: errors.New("token unavailable")}
		err := application.NewService(identityVerifier{}, repository).ResetPassword(context.Background(), "expired-token", "new-password-123")
		if !errors.Is(err, domain.ErrInvalidRecoveryToken) {
			t.Fatalf("ResetPassword() error=%v, want ErrInvalidRecoveryToken", err)
		}
	})
}

func TestUpdateUserRoleRequiresAdminAndUsesTargetUID(t *testing.T) {
	repository := &repository{user: domain.User{ID: "admin-id", FirebaseUID: "admin-uid", Role: "admin", Status: "activo"}}
	service := application.NewService(identityVerifier{identity: domain.Identity{UID: "admin-uid", Email: "admin@example.com"}}, repository)
	login, err := service.Login(context.Background(), "firebase-token")
	if err != nil {
		t.Fatal(err)
	}
	if err := service.UpdateUserRole(context.Background(), login.Token, "target-uid", "docente"); err != nil {
		t.Fatalf("UpdateUserRole() error=%v", err)
	}
	if repository.updatedUID != "target-uid" || repository.updatedRole != "docente" {
		t.Fatalf("updated target=%q role=%q", repository.updatedUID, repository.updatedRole)
	}
	if err := service.UpdateUserRole(context.Background(), login.Token, "admin-uid", "docente"); !errors.Is(err, domain.ErrSelfRoleChange) {
		t.Fatalf("self role change error=%v", err)
	}
}
