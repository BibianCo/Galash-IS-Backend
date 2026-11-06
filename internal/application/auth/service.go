package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"

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
	StudentCode    string
	BirthDate      string
	Phone          string
	// ClaimID is the id of an imported SGI account returned by MatchImported; empty means a normal sign-up.
	ClaimID string
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
	photos     ports.ProfilePhotoWriter
	imported   ports.ImportedUserRepository
}

// Roles are persisted in their canonical lowercase form.
// "docente" is the contract name; "profesor" stays for existing rows and callers.
var allowedRoles = map[string]bool{"visitante": true, "estudiante": true, "docente": true, "profesor": true, "admin": true}

func NewService(identities ports.IdentityVerifier, users ports.UserRepository) *Service {
	s := &Service{identities: identities, users: users}
	if reader, ok := users.(ports.UserReader); ok {
		s.userReader = reader
	}
	if repository, ok := users.(ports.SessionRepository); ok {
		s.sessions = repository
	}
	if repository, ok := users.(ports.RecoveryTokenRepository); ok {
		s.recovery = repository
	}
	if repository, ok := users.(ports.ProfilePhotoWriter); ok {
		s.photos = repository
	}
	if repository, ok := users.(ports.ImportedUserRepository); ok {
		s.imported = repository
	}
	return s
}

// Self-registration never chooses a role: every new account starts as visitante.
// Other roles are assigned later through UpdateUserRole by an admin.
func (s *Service) Register(ctx context.Context, token string, command RegisterCommand) (RegisterResult, error) {
	identity, err := s.identities.Verify(ctx, token)
	if err != nil {
		return RegisterResult{}, errors.Join(domain.ErrInvalidIdentity, err)
	}

	if identity.Email == "" {
		return RegisterResult{}, domain.ErrIncompleteIdentity
	}
	if command.Name == "" || command.LastName == "" || command.DocumentType == "" || command.DocumentNumber == "" || command.Email == "" || strings.TrimSpace(command.Phone) == "" {
		return RegisterResult{}, domain.ErrInvalidSignup
	}
	if _, err := time.Parse("2006-01-02", strings.TrimSpace(command.BirthDate)); err != nil {
		return RegisterResult{}, domain.ErrInvalidSignup
	}
	if !strings.EqualFold(strings.TrimSpace(command.Email), identity.Email) {
		return RegisterResult{}, domain.ErrInvalidSignup
	}

	user := domain.User{
		FirebaseUID:    identity.UID,
		Name:           command.Name,
		LastName:       command.LastName,
		Email:          strings.TrimSpace(command.Email),
		DocumentType:   command.DocumentType,
		DocumentNumber: command.DocumentNumber,
		StudentCode:    strings.TrimSpace(command.StudentCode),
		BirthDate:      strings.TrimSpace(command.BirthDate),
		Phone:          strings.TrimSpace(command.Phone),
		Status:         "activo",
	}
	var id string
	if claimID := strings.TrimSpace(command.ClaimID); claimID != "" {
		id, err = s.claim(ctx, identity, claimID, user)
	} else {
		id, err = s.users.Upsert(ctx, user)
	}
	if err != nil {
		return RegisterResult{}, err
	}
	return RegisterResult{ID: id, FirebaseUID: identity.UID}, nil
}

// MatchImported lists the unclaimed SGI accounts that match the verified institutional email of the token.
func (s *Service) MatchImported(ctx context.Context, token string) ([]domain.User, error) {
	identity, err := s.identities.Verify(ctx, strings.TrimSpace(token))
	if err != nil {
		return nil, errors.Join(domain.ErrInvalidIdentity, err)
	}
	return s.matchImported(ctx, identity)
}

func (s *Service) matchImported(ctx context.Context, identity domain.Identity) ([]domain.User, error) {
	if s.imported == nil {
		return []domain.User{}, nil
	}
	first, last, ok := InstitutionalNameParts(identity)
	if !ok {
		return []domain.User{}, nil
	}
	users, err := s.imported.ListImported(ctx)
	if err != nil {
		return nil, err
	}
	matches := []domain.User{}
	for _, u := range users {
		tokens := strings.Fields(normalizeName(u.Name + " " + u.LastName))
		if containsToken(tokens, first) && containsToken(tokens, last) {
			matches = append(matches, u)
		}
	}
	return matches, nil
}

// claim recomputes the match server-side: the client never decides which imported account it gets.
func (s *Service) claim(ctx context.Context, identity domain.Identity, claimID string, user domain.User) (string, error) {
	matches, err := s.matchImported(ctx, identity)
	if err != nil {
		return "", err
	}
	for _, m := range matches {
		if m.ID == claimID {
			return s.imported.ClaimImported(ctx, claimID, user)
		}
	}
	return "", domain.ErrClaimNotAllowed
}

var institutionalLocalPart = regexp.MustCompile(`^([a-z]+)\.([a-z]+)(?:[0-9]{2})?$`)

// InstitutionalNameParts extracts first name and last name from a verified primernombre.apellidoNN@uptc.edu.co email.
func InstitutionalNameParts(identity domain.Identity) (first, last string, ok bool) {
	if !identity.EmailVerified {
		return "", "", false
	}
	local, domainPart, found := strings.Cut(strings.ToLower(strings.TrimSpace(identity.Email)), "@")
	if !found || domainPart != "uptc.edu.co" {
		return "", "", false
	}
	parts := institutionalLocalPart.FindStringSubmatch(local)
	if parts == nil {
		return "", "", false
	}
	return parts[1], parts[2], true
}

var nameAccents = strings.NewReplacer("á", "a", "é", "e", "í", "i", "ó", "o", "ú", "u", "ü", "u", "ñ", "n")

func normalizeName(name string) string {
	return nameAccents.Replace(strings.ToLower(name))
}

func containsToken(tokens []string, want string) bool {
	for _, t := range tokens {
		if t == want {
			return true
		}
	}
	return false
}

type LoginResult struct {
	Token string
	User  domain.User
}

// Login accepts the Firebase ID token produced by the client, verifies it, and
// creates an opaque server-side session. The Firebase token is never stored.
func (s *Service) Login(ctx context.Context, token string) (LoginResult, error) {
	if s.sessions == nil {
		return LoginResult{}, errors.New("sesiones no configuradas")
	}
	identity, err := s.identities.Verify(ctx, strings.TrimSpace(token))
	if err != nil || identity.UID == "" || identity.Email == "" {
		return LoginResult{}, domain.ErrInvalidCredentials
	}
	if s.userReader == nil {
		return LoginResult{}, domain.ErrUserNotRegistered
	}
	user, err := s.userReader.FindByFirebaseUID(ctx, identity.UID)
	if err != nil {
		return LoginResult{}, domain.ErrUserNotRegistered
	}
	if user.Status != "" && user.Status != "activo" {
		return LoginResult{}, domain.ErrInactiveUser
	}
	plain, err := randomToken()
	if err != nil {
		return LoginResult{}, err
	}
	if err := s.sessions.Create(ctx, user.ID, hashToken(plain)); err != nil {
		return LoginResult{}, err
	}
	return LoginResult{Token: plain, User: user}, nil
}

func (s *Service) CurrentUser(ctx context.Context, token string) (domain.User, error) {
	if s.sessions == nil || s.userReader == nil {
		return domain.User{}, domain.ErrInvalidSession
	}
	id, err := s.sessions.FindUserID(ctx, hashToken(token))
	if err != nil || id == "" {
		return domain.User{}, domain.ErrInvalidSession
	}
	user, err := s.userReader.FindByID(ctx, id)
	if err != nil || (user.Status != "" && user.Status != "activo") {
		return domain.User{}, domain.ErrInvalidSession
	}
	return user, nil
}

// SetProfilePhoto stores the image link on the session's own user; the client never picks the user.
func (s *Service) SetProfilePhoto(ctx context.Context, token, photoURL string) (string, error) {
	user, err := s.CurrentUser(ctx, token)
	if err != nil {
		return "", err
	}
	photoURL = strings.TrimSpace(photoURL)
	u, err := url.ParseRequestURI(photoURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || len(photoURL) > 2048 {
		return "", domain.ErrInvalidPhotoURL
	}
	if s.photos == nil {
		return "", errors.New("repositorio de fotos no configurado")
	}
	if err := s.photos.SetPhotoURL(ctx, user.ID, photoURL); err != nil {
		return "", err
	}
	return photoURL, nil
}

func (s *Service) SearchUsers(ctx context.Context, query string, limit int) ([]domain.User, error) {
	return s.searchUsers(ctx, query, limit, false)
}

func (s *Service) searchUsers(ctx context.Context, query string, limit int, includeInactive bool) ([]domain.User, error) {
	if s.userReader == nil || (!includeInactive && strings.TrimSpace(query) == "") {
		return nil, domain.ErrIncompleteIdentity
	}
	// An unfiltered admin request returns the complete account list (M11).
	if includeInactive && strings.TrimSpace(query) == "" {
		return s.userReader.Search(ctx, "", 0, true)
	}
	if limit < 5 {
		limit = 5
	}
	if limit > 50 {
		limit = 50
	}
	return s.userReader.Search(ctx, strings.TrimSpace(query), limit, includeInactive)
}

func (s *Service) SearchAdminUsers(ctx context.Context, token, query string, limit int) ([]domain.User, error) {
	actor, err := s.CurrentUser(ctx, token)
	if err != nil {
		return nil, domain.ErrInvalidSession
	}
	if !strings.EqualFold(strings.TrimSpace(actor.Role), "admin") {
		return nil, domain.ErrForbidden
	}
	// Admins must see inactive accounts to reactivate them (contract M11).
	return s.searchUsers(ctx, query, limit, true)
}

func (s *Service) UpdateUserRole(ctx context.Context, token, targetUID, role string) error {
	if s.userReader == nil {
		return domain.ErrInvalidSession
	}
	actor, err := s.CurrentUser(ctx, token)
	if err != nil {
		return domain.ErrInvalidSession
	}
	if !strings.EqualFold(strings.TrimSpace(actor.Role), "admin") {
		return domain.ErrForbidden
	}
	targetUID = strings.TrimSpace(targetUID)
	role = strings.ToLower(strings.TrimSpace(role))
	if targetUID == "" || !allowedRoles[role] {
		if targetUID == "" {
			return domain.ErrIncompleteIdentity
		}
		return domain.ErrInvalidRole
	}
	if actor.FirebaseUID != "" && actor.FirebaseUID == targetUID {
		return domain.ErrSelfRoleChange
	}
	return s.userReader.UpdateRole(ctx, targetUID, role)
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if s.sessions == nil {
		return domain.ErrInvalidSession
	}
	return s.sessions.Revoke(ctx, hashToken(token))
}

func (s *Service) RequestPasswordRecovery(ctx context.Context, email string) error {
	if s.recovery == nil || s.userReader == nil {
		return nil
	}
	user, err := s.userReader.FindByEmail(ctx, strings.TrimSpace(strings.ToLower(email)))
	if err != nil {
		return nil
	} // Deliberately indistinguishable to callers.
	plain, err := randomToken()
	if err != nil {
		return err
	}
	return s.recovery.CreateRecovery(ctx, user.ID, hashToken(plain))
}

func (s *Service) ResetPassword(ctx context.Context, token, password string) error {
	if s.recovery == nil || s.userReader == nil || len(password) < 8 {
		return domain.ErrInvalidRecoveryToken
	}
	id, err := s.recovery.Consume(ctx, hashToken(token))
	if err != nil {
		return domain.ErrInvalidRecoveryToken
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	// A password reset ends every session that was opened with the old credentials.
	// Revoke first: if this fails, the password is unchanged and nothing is half-applied.
	if s.sessions != nil {
		if err := s.sessions.RevokeAllForUser(ctx, id); err != nil {
			return err
		}
	}
	return s.userReader.SetPassword(ctx, id, string(hash))
}

// PurgeExpiredCredentials deletes sessions and recovery tokens that can no longer be used.
func (s *Service) PurgeExpiredCredentials(ctx context.Context) error {
	if s.sessions != nil {
		if err := s.sessions.DeleteExpiredSessions(ctx); err != nil {
			return err
		}
	}
	if s.recovery != nil {
		return s.recovery.DeleteExpiredRecoveries(ctx)
	}
	return nil
}

func randomToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
