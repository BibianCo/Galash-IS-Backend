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
	SetPassword(ctx context.Context, id string, passwordHash string) error
}

type SessionRepository interface {
	Create(ctx context.Context, userID string, tokenHash string) error
	FindUserID(ctx context.Context, tokenHash string) (string, error)
	Revoke(ctx context.Context, tokenHash string) error
}

type RecoveryTokenRepository interface {
	CreateRecovery(ctx context.Context, userID string, tokenHash string) error
	Consume(ctx context.Context, tokenHash string) (string, error)
}

type HealthChecker interface {
	Ping(ctx context.Context) error
}
