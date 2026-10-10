package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
	"golang.org/x/crypto/bcrypt"
)

type RegisterCommand struct {
	Name           string
	LastName       string
	DocumentType   string
	DocumentNumber string
	Email          string
	Role           string
}

type RegisterResult struct {
	ID          string `json:"id"`
	FirebaseUID string `json:"firebase_uid"`
}

type Service struct {
	identities ports.IdentityVerifier
	users      ports.UserRepository
	userReader ports.UserReader
	sessions   ports.SessionRepository
	recovery   ports.RecoveryTokenRepository
}

func NewService(identities ports.IdentityVerifier, users ports.UserRepository) *Service {
	s := &Service{identities: identities, users: users}
	if reader, ok := users.(ports.UserReader); ok { s.userReader = reader }
	if repository, ok := users.(ports.SessionRepository); ok { s.sessions = repository }
	if repository, ok := users.(ports.RecoveryTokenRepository); ok { s.recovery = repository }
	return s
}

func (s *Service) Register(ctx context.Context, token string, command RegisterCommand) (RegisterResult, error) {
	identity, err := s.identities.Verify(ctx, token)
	if err != nil {
		return RegisterResult{}, errors.Join(domain.ErrInvalidIdentity, err)
	}

	if command.Name == "" || command.LastName == "" || command.DocumentType == "" || command.DocumentNumber == "" || command.Email == "" || command.Role == "" || identity.Email == "" {
		return RegisterResult{}, domain.ErrIncompleteIdentity
	}
	if !strings.EqualFold(strings.TrimSpace(command.Email), identity.Email) {
		return RegisterResult{}, domain.ErrIncompleteIdentity
	}
	if command.Role != "estudiante" && command.Role != "profesor" {
		return RegisterResult{}, domain.ErrIncompleteIdentity
	}

	id, err := s.users.Upsert(ctx, domain.User{
		FirebaseUID: identity.UID,
		Name:        command.Name,
		LastName:    command.LastName,
		Email:       strings.TrimSpace(command.Email),
		DocumentType: command.DocumentType,
		DocumentNumber: command.DocumentNumber,
		Role:        command.Role,
		Status:      "activo",
	})
	if err != nil {
		return RegisterResult{}, err
	}
	return RegisterResult{ID: id, FirebaseUID: identity.UID}, nil
}

type LoginResult struct { Token string; User domain.User }

// Login accepts the Firebase ID token produced by the client, verifies it, and
// creates an opaque server-side session. The Firebase token is never stored.
func (s *Service) Login(ctx context.Context, token string) (LoginResult, error) {
	if s.sessions == nil { return LoginResult{}, errors.New("sesiones no configuradas") }
	identity, err := s.identities.Verify(ctx, strings.TrimSpace(token))
	if err != nil || identity.UID == "" || identity.Email == "" { return LoginResult{}, domain.ErrInvalidCredentials }
	id, err := s.users.Upsert(ctx, domain.User{FirebaseUID: identity.UID, Name: identity.Name, Email: identity.Email, Status: "activo"})
	if err != nil { return LoginResult{}, err }
	user := domain.User{ID: id, FirebaseUID: identity.UID, Name: identity.Name, Email: identity.Email, Status: "activo"}
	if s.userReader != nil {
		user, err = s.userReader.FindByID(ctx, id)
		if err != nil { return LoginResult{}, err }
		if user.Status != "" && user.Status != "activo" { return LoginResult{}, domain.ErrInactiveUser }
	}
	plain, err := randomToken()
	if err != nil { return LoginResult{}, err }
	if err := s.sessions.Create(ctx, user.ID, hashToken(plain)); err != nil { return LoginResult{}, err }
	return LoginResult{Token: plain, User: user}, nil
}

func (s *Service) CurrentUser(ctx context.Context, token string) (domain.User, error) {
	if s.sessions == nil || s.userReader == nil { return domain.User{}, domain.ErrInvalidSession }
	id, err := s.sessions.FindUserID(ctx, hashToken(token))
	if err != nil || id == "" { return domain.User{}, domain.ErrInvalidSession }
	user, err := s.userReader.FindByID(ctx, id)
	if err != nil || (user.Status != "" && user.Status != "activo") { return domain.User{}, domain.ErrInvalidSession }
	return user, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if s.sessions == nil { return domain.ErrInvalidSession }
	return s.sessions.Revoke(ctx, hashToken(token))
}

func (s *Service) RequestPasswordRecovery(ctx context.Context, email string) error {
	if s.recovery == nil || s.userReader == nil { return nil }
	user, err := s.userReader.FindByEmail(ctx, strings.TrimSpace(strings.ToLower(email)))
	if err != nil { return nil } // Deliberately indistinguishable to callers.
	plain, err := randomToken(); if err != nil { return err }
	return s.recovery.CreateRecovery(ctx, user.ID, hashToken(plain))
}

func (s *Service) ResetPassword(ctx context.Context, token, password string) error {
	if s.recovery == nil || s.userReader == nil || len(password) < 8 { return domain.ErrInvalidRecoveryToken }
	id, err := s.recovery.Consume(ctx, hashToken(token))
	if err != nil { return domain.ErrInvalidRecoveryToken }
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil { return err }
	return s.userReader.SetPassword(ctx, id, string(hash))
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil { return "", err }
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
