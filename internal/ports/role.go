package ports

import (
	"context"

	"github.com/galash-uptc/backend/internal/domain"
)

type RoleRepository interface {
	Permissions(ctx context.Context) ([]domain.Permission, error)
	RolePermissions(ctx context.Context, role string) ([]domain.Permission, error)
	// SetRolePermissions replaces the role's permissions and records actor and date.
	SetRolePermissions(ctx context.Context, role string, permissionIDs []string, actorID string) error
}
