package postgres

import (
	"context"

	"github.com/galash-uptc/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RoleRepository struct {
	db *pgxpool.Pool
}

func NewRoleRepository(db *pgxpool.Pool) *RoleRepository {
	return &RoleRepository{db: db}
}

func (r *RoleRepository) Permissions(ctx context.Context) ([]domain.Permission, error) {
	return r.permissions(ctx, `SELECT id, descripcion FROM permisos ORDER BY id`)
}

func (r *RoleRepository) RolePermissions(ctx context.Context, role string) ([]domain.Permission, error) {
	return r.permissions(ctx, `
		SELECT p.id, p.descripcion FROM permisos p
		JOIN rol_permisos rp ON rp.permiso_id = p.id
		WHERE rp.rol = $1 ORDER BY p.id`, role)
}

func (r *RoleRepository) permissions(ctx context.Context, query string, args ...any) ([]domain.Permission, error) {
	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	items, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Permission, error) {
		var p domain.Permission
		return p, row.Scan(&p.ID, &p.Description)
	})
	if items == nil {
		items = []domain.Permission{}
	}
	return items, err
}

func (r *RoleRepository) SetRolePermissions(ctx context.Context, role string, permissionIDs []string, actorID string) error {
	return pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM rol_permisos WHERE rol = $1`, role); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO rol_permisos (rol, permiso_id) SELECT $1, unnest($2::text[])`, role, permissionIDs); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `INSERT INTO rol_permisos_historial (rol, permisos, actor_id) VALUES ($1, $2, NULLIF($3, '')::uuid)`, role, permissionIDs, actorID)
		return err
	})
}
