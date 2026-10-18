package ports

import (
	"context"

	"github.com/galash-uptc/backend/internal/domain"
)

type IdentityVerifier interface {
	Verify(ctx context.Context, token string) (domain.Identity, error)
}

type UserRepository interface {
	Upsert(ctx context.Context, user domain.User) (string, error)
}

type UserReader interface {
	FindByID(ctx context.Context, id string) (domain.User, error)
	FindByEmail(ctx context.Context, email string) (domain.User, error)
	FindByFirebaseUID(ctx context.Context, uid string) (domain.User, error)
	SetPassword(ctx context.Context, id string, passwordHash string) error
	Search(ctx context.Context, query string, limit int) ([]domain.User, error)
	UpdateRole(ctx context.Context, firebaseUID string, role string) error
}

type SessionRepository interface {
	Create(ctx context.Context, userID string, tokenHash string) error
	FindUserID(ctx context.Context, tokenHash string) (string, error)
	Revoke(ctx context.Context, tokenHash string) error
	RevokeAllForUser(ctx context.Context, userID string) error
	DeleteExpiredSessions(ctx context.Context) error
}

type RecoveryTokenRepository interface {
	CreateRecovery(ctx context.Context, userID string, tokenHash string) error
	Consume(ctx context.Context, tokenHash string) (string, error)
	DeleteExpiredRecoveries(ctx context.Context) error
}

type HealthChecker interface {
	Ping(ctx context.Context) error
}

type SeedbedRepository interface {
	Create(ctx context.Context, seedbed domain.Seedbed) (domain.Seedbed, error)
	List(ctx context.Context, query, researchLine string, page, pageSize int, includePrivate bool) ([]domain.Seedbed, error)
	Find(ctx context.Context, id string) (domain.Seedbed, error)
	Space(ctx context.Context, id string) (domain.SeedbedSpace, error)
	Update(ctx context.Context, id string, seedbed domain.Seedbed) (domain.Seedbed, error)
	SetVisibility(ctx context.Context, id, status string) (domain.Seedbed, error)
	ListMembers(ctx context.Context, id string, includeHistory bool) ([]domain.SeedbedMember, error)
	AddMember(ctx context.Context, seedbedID, userID string) error
	RemoveMember(ctx context.Context, seedbedID, userID string) error
}
