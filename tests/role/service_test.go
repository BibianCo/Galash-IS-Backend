package role_test

import (
	"context"
	"errors"
	"testing"

	roleapp "github.com/galash-uptc/backend/internal/application/role"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
)

// memoryRoles is an in-memory RoleRepository with a small permission catalog.
type memoryRoles struct {
	catalog []domain.Permission
	byRole  map[string][]string
	actor   string
}

func newMemoryRoles() *memoryRoles {
	return &memoryRoles{
		catalog: []domain.Permission{{ID: "audit.read", Description: "a"}, {ID: "files.upload", Description: "f"}, {ID: "roles.manage", Description: "r"}},
		byRole:  map[string][]string{"admin": {"audit.read", "files.upload", "roles.manage"}, "estudiante": {"files.upload"}},
	}
}
func (m *memoryRoles) Permissions(context.Context) ([]domain.Permission, error) {
	return m.catalog, nil
}
func (m *memoryRoles) RolePermissions(_ context.Context, role string) ([]domain.Permission, error) {
	items := []domain.Permission{}
	for _, id := range m.byRole[role] {
		items = append(items, domain.Permission{ID: id})
	}
	return items, nil
}
func (m *memoryRoles) SetRolePermissions(_ context.Context, role string, ids []string, actorID string) error {
	m.byRole[role], m.actor = ids, actorID
	return nil
}

// users implements the lookups the role service uses; other methods are unused.
type users struct{ ports.UserReader }

func (users) FindByFirebaseUID(_ context.Context, uid string) (domain.User, error) {
	switch uid {
	case "uid-teacher":
		return domain.User{ID: "teacher", Role: "profesor"}, nil
	case "uid-legacy":
		return domain.User{ID: "legacy", Role: "coordinador"}, nil
	}
	return domain.User{}, errors.New("not found")
}

var admin = domain.User{ID: "admin-id", Role: "admin"}

func TestCatalogHasOnlyTheFourContractRoles(t *testing.T) {
	roles := roleapp.NewService(newMemoryRoles(), users{}).Roles()
	want := map[string]string{"visitante": "Visitante", "estudiante": "Estudiante", "profesor": "Docente", "admin": "Admin"}
	if len(roles) != len(want) {
		t.Fatalf("roles=%+v", roles)
	}
	for _, r := range roles {
		if want[r.ID] != r.Name {
			t.Errorf("role %+v not in catalog", r)
		}
	}
}

func TestRolePermissionsAcceptsContractLabel(t *testing.T) {
	repo := newMemoryRoles()
	repo.byRole["profesor"] = []string{"files.upload"}
	r, items, err := roleapp.NewService(repo, users{}).RolePermissions(context.Background(), "Docente")
	if err != nil || r.ID != "profesor" || len(items) != 1 {
		t.Fatalf("role=%+v items=%+v err=%v", r, items, err)
	}
	if _, _, err := roleapp.NewService(repo, users{}).RolePermissions(context.Background(), "coordinador"); !errors.Is(err, domain.ErrRoleNotFound) {
		t.Fatalf("unknown role err=%v, want ErrRoleNotFound", err)
	}
}

func TestSetRolePermissionsValidatesDeduplicatesAndRecordsActor(t *testing.T) {
	repo := newMemoryRoles()
	s := roleapp.NewService(repo, users{})
	_, items, err := s.SetRolePermissions(context.Background(), admin, "estudiante", []string{"audit.read", " audit.read ", "files.upload"})
	if err != nil || len(items) != 2 || repo.actor != admin.ID {
		t.Fatalf("items=%+v actor=%q err=%v", items, repo.actor, err)
	}
	if _, _, err := s.SetRolePermissions(context.Background(), admin, "estudiante", []string{"no.existe"}); !errors.Is(err, domain.ErrInvalidPermission) {
		t.Fatalf("unknown permission err=%v, want ErrInvalidPermission", err)
	}
	if len(repo.byRole["estudiante"]) != 2 {
		t.Fatalf("rejected request changed permissions: %+v", repo.byRole["estudiante"])
	}
	if _, items, err := s.SetRolePermissions(context.Background(), admin, "estudiante", []string{}); err != nil || len(items) != 0 {
		t.Fatalf("empty list must remove all: items=%+v err=%v", items, err)
	}
}

func TestSetRolePermissionsRejectsActorOwnRole(t *testing.T) {
	repo := newMemoryRoles()
	s := roleapp.NewService(repo, users{})
	if _, _, err := s.SetRolePermissions(context.Background(), admin, "Admin", []string{}); !errors.Is(err, domain.ErrSelfPermissionChange) {
		t.Fatalf("err=%v, want ErrSelfPermissionChange", err)
	}
	if len(repo.byRole["admin"]) != 3 {
		t.Fatalf("admin permissions changed: %+v", repo.byRole["admin"])
	}
}

func TestUserRolesUsesFirebaseUID(t *testing.T) {
	s := roleapp.NewService(newMemoryRoles(), users{})
	roles, err := s.UserRoles(context.Background(), "uid-teacher")
	if err != nil || len(roles) != 1 || roles[0].Name != "Docente" {
		t.Fatalf("roles=%+v err=%v", roles, err)
	}
	if roles, err := s.UserRoles(context.Background(), "uid-legacy"); err != nil || len(roles) != 0 {
		t.Fatalf("role outside catalog must yield no roles: roles=%+v err=%v", roles, err)
	}
	if _, err := s.UserRoles(context.Background(), "missing"); !errors.Is(err, domain.ErrUserNotFound) {
		t.Fatalf("err=%v, want ErrUserNotFound", err)
	}
}

func TestDashboardReturnsRolePermissionsAndExcludesVisitante(t *testing.T) {
	s := roleapp.NewService(newMemoryRoles(), users{})
	d, err := s.Dashboard(context.Background(), domain.User{ID: "ana", Role: "estudiante"})
	if err != nil || d.Role.ID != "estudiante" || len(d.Permissions) != 1 || d.User.ID != "ana" {
		t.Fatalf("dashboard=%+v err=%v", d, err)
	}
	for _, role := range []string{"visitante", "coordinador"} {
		if _, err := s.Dashboard(context.Background(), domain.User{Role: role}); !errors.Is(err, domain.ErrForbidden) {
			t.Errorf("%s: err=%v, want ErrForbidden", role, err)
		}
	}
}
