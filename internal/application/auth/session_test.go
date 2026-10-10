package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/galash-uptc/backend/internal/domain"
)

type sessionIdentityVerifier struct {
	identity domain.Identity
	err      error
}

func (v sessionIdentityVerifier) Verify(context.Context, string) (domain.Identity, error) {
	return v.identity, v.err
}

type sessionRepositoryStub struct {
	user           domain.User
	upserted       domain.User
	findUserErr    error
	createdHash    string
	createdUser    string
	revokedHash    string
	sessionID      string
	findSessionErr error
}

func (r *sessionRepositoryStub) Upsert(_ context.Context, user domain.User) (string, error) {
	r.upserted = user
	return r.user.ID, nil
}

func (r *sessionRepositoryStub) FindByID(context.Context, string) (domain.User, error) {
	if r.findUserErr != nil {
		return domain.User{}, r.findUserErr
	}
	return r.user, nil
}

func (r *sessionRepositoryStub) FindByEmail(context.Context, string) (domain.User, error) {
	return r.user, nil
}

func (r *sessionRepositoryStub) SetPassword(context.Context, string, string) error { return nil }

func (r *sessionRepositoryStub) Create(_ context.Context, userID, tokenHash string) error {
	r.createdUser = userID
	r.createdHash = tokenHash
	return nil
}

func (r *sessionRepositoryStub) FindUserID(_ context.Context, tokenHash string) (string, error) {
	if r.findSessionErr != nil {
		return "", r.findSessionErr
	}
	if tokenHash != r.createdHash {
		return "", errors.New("session not found")
	}
	return r.sessionID, nil
}

func (r *sessionRepositoryStub) Revoke(_ context.Context, tokenHash string) error {
	r.revokedHash = tokenHash
	return nil
}

func TestLoginCreatesOpaqueSession(t *testing.T) {
	repository := &sessionRepositoryStub{
		user:      domain.User{ID: "user-id", Name: "Ana", Email: "ana@example.com", Status: "activo"},
		sessionID: "user-id",
	}
	service := NewService(sessionIdentityVerifier{identity: domain.Identity{UID: "firebase-id", Email: "ana@example.com"}}, repository)

	result, err := service.Login(context.Background(), "firebase-token")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if result.Token == "" {
		t.Fatal("Login() returned empty session token")
	}
	if repository.createdUser != "user-id" || repository.createdHash != hashToken(result.Token) {
		t.Fatalf("unexpected session creation: user=%q hash=%q", repository.createdUser, repository.createdHash)
	}
	if repository.createdHash == result.Token {
		t.Fatal("Login() stored the plain session token")
	}
}

func TestLoginRejectsInvalidIdentityWithoutCreatingSession(t *testing.T) {
	repository := &sessionRepositoryStub{user: domain.User{ID: "user-id"}}
	service := NewService(sessionIdentityVerifier{err: errors.New("invalid token")}, repository)

	_, err := service.Login(context.Background(), "bad-token")
	if !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want ErrInvalidCredentials", err)
	}
	if repository.createdHash != "" {
		t.Fatal("Login() created a session for invalid credentials")
	}
}

func TestLoginRejectsInactiveUser(t *testing.T) {
	repository := &sessionRepositoryStub{
		user: domain.User{ID: "user-id", Email: "ana@example.com", Status: "inactivo"},
	}
	service := NewService(sessionIdentityVerifier{identity: domain.Identity{UID: "firebase-id", Email: "ana@example.com"}}, repository)

	_, err := service.Login(context.Background(), "firebase-token")
	if !errors.Is(err, domain.ErrInactiveUser) {
		t.Fatalf("Login() error = %v, want ErrInactiveUser", err)
	}
	if repository.createdHash != "" {
		t.Fatal("Login() created a session for inactive user")
	}
}

func TestCurrentUserReturnsActiveUserForValidSession(t *testing.T) {
	repository := &sessionRepositoryStub{
		user:        domain.User{ID: "user-id", Name: "Ana", Status: "activo"},
		sessionID:   "user-id",
		createdHash: hashToken("session-token"),
	}
	service := NewService(sessionIdentityVerifier{}, repository)

	user, err := service.CurrentUser(context.Background(), "session-token")
	if err != nil {
		t.Fatalf("CurrentUser() error = %v", err)
	}
	if user.ID != "user-id" || user.Name != "Ana" {
		t.Fatalf("CurrentUser() = %+v", user)
	}
}

func TestCurrentUserRejectsInvalidSession(t *testing.T) {
	repository := &sessionRepositoryStub{
		user:           domain.User{ID: "user-id", Status: "activo"},
		findSessionErr: errors.New("expired session"),
	}
	service := NewService(sessionIdentityVerifier{}, repository)

	_, err := service.CurrentUser(context.Background(), "expired-token")
	if !errors.Is(err, domain.ErrInvalidSession) {
		t.Fatalf("CurrentUser() error = %v, want ErrInvalidSession", err)
	}
}

func TestLogoutRevokesHashedSession(t *testing.T) {
	repository := &sessionRepositoryStub{}
	service := NewService(sessionIdentityVerifier{}, repository)

	if err := service.Logout(context.Background(), "session-token"); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if repository.revokedHash != hashToken("session-token") {
		t.Fatalf("Logout() revoked hash = %q, want %q", repository.revokedHash, hashToken("session-token"))
	}
}
