package role

import (
	"context"
	"strings"

	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
)

type Service struct {
	repo  ports.RoleRepository
	users ports.UserReader
}

func NewService(repo ports.RoleRepository, users ports.UserReader) *Service {
	return &Service{repo: repo, users: users}
}

func (s *Service) Roles() []domain.Role { return domain.Roles }

func (s *Service) Permissions(ctx context.Context) ([]domain.Permission, error) {
	return s.repo.Permissions(ctx)
}

func (s *Service) RolePermissions(ctx context.Context, roleID string) (domain.Role, []domain.Permission, error) {
	r, ok := domain.FindRole(roleID)
	if !ok {
		return domain.Role{}, nil, domain.ErrRoleNotFound
	}
	items, err := s.repo.RolePermissions(ctx, r.ID)
	return r, items, err
}

// SetRolePermissions replaces the permissions of roleID. The actor cannot change
// the role they hold, so no one grants privileges to themselves.
func (s *Service) SetRolePermissions(ctx context.Context, actor domain.User, roleID string, permissionIDs []string) (domain.Role, []domain.Permission, error) {
	r, ok := domain.FindRole(roleID)
	if !ok {
		return domain.Role{}, nil, domain.ErrRoleNotFound
	}
	if own, ok := domain.FindRole(actor.Role); ok && own.ID == r.ID {
		return domain.Role{}, nil, domain.ErrSelfPermissionChange
	}
	catalog, err := s.repo.Permissions(ctx)
	if err != nil {
		return domain.Role{}, nil, err
	}
	known := map[string]bool{}
	for _, p := range catalog {
		known[p.ID] = true
	}
	ids := []string{}
	seen := map[string]bool{}
	for _, id := range permissionIDs {
		id = strings.TrimSpace(id)
		if !known[id] {
			return domain.Role{}, nil, domain.ErrInvalidPermission
		}
		if !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	if err := s.repo.SetRolePermissions(ctx, r.ID, ids, actor.ID); err != nil {
		return domain.Role{}, nil, err
	}
	return s.RolePermissions(ctx, r.ID)
}

// UserRoles looks the user up by Firebase UID, the same identifier PUT /users/{userId}/roles uses.
func (s *Service) UserRoles(ctx context.Context, firebaseUID string) ([]domain.Role, error) {
	u, err := s.users.FindByFirebaseUID(ctx, strings.TrimSpace(firebaseUID))
	if err != nil {
		return nil, domain.ErrUserNotFound
	}
	roles := []domain.Role{}
	if r, ok := domain.FindRole(u.Role); ok {
		roles = append(roles, r)
	}
	return roles, nil
}

// Dashboard is only available to Estudiante, Docente and Admin.
func (s *Service) Dashboard(ctx context.Context, user domain.User) (domain.Dashboard, error) {
	r, ok := domain.FindRole(user.Role)
	if !ok || r.ID == "visitante" {
		return domain.Dashboard{}, domain.ErrForbidden
	}
	items, err := s.repo.RolePermissions(ctx, r.ID)
	if err != nil {
		return domain.Dashboard{}, err
	}
	return domain.Dashboard{User: user, Role: r, Permissions: items}, nil
}
