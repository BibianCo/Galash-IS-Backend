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

type HealthChecker interface {
	Ping(ctx context.Context) error
}
