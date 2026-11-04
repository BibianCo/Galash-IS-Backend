package postgres

import (
	"context"
	"errors"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type OfficialLinkRepository struct{ db *pgxpool.Pool }

func NewOfficialLinkRepository(db *pgxpool.Pool) *OfficialLinkRepository {
	return &OfficialLinkRepository{db: db}
}

const linkCols = `e.id,e.nombre,e.descripcion,e.url,e.categoria,e.visible,e.orden,e.estado,e.actualizado_por,e.actualizado_en,e.creado_en`
const linkReturningCols = `id,nombre,descripcion,url,categoria,visible,orden,estado,actualizado_por,actualizado_en,creado_en`

func linkError(err error) error {
	var pgErr *pgconn.PgError
	if errors.Is(err, pgx.ErrNoRows) || errors.As(err, &pgErr) && pgErr.Code == "22P02" {
		return domain.ErrOfficialLinkNotFound
	}
	return err
}
func scanLink(row pgx.Row) (domain.OfficialLink, error) {
	var l domain.OfficialLink
	err := row.Scan(&l.ID, &l.Name, &l.Description, &l.URL, &l.Category, &l.Visible, &l.Order, &l.Status, &l.UpdatedBy, &l.UpdatedAt, &l.CreatedAt)
	return l, linkError(err)
}
func (r *OfficialLinkRepository) ListVisible(ctx context.Context, category string, includeAll bool) ([]domain.OfficialLink, error) {
	rows, err := r.db.Query(ctx, `SELECT `+linkCols+` FROM enlaces_oficiales e WHERE ($1 OR (e.visible=true AND e.estado='activo')) AND ($2='' OR e.categoria=$2) ORDER BY e.orden,e.nombre`, includeAll, category)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.OfficialLink{}
	for rows.Next() {
		var l domain.OfficialLink
		if err := rows.Scan(&l.ID, &l.Name, &l.Description, &l.URL, &l.Category, &l.Visible, &l.Order, &l.Status, &l.UpdatedBy, &l.UpdatedAt, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
func (r *OfficialLinkRepository) Find(ctx context.Context, id string) (domain.OfficialLink, error) {
	return scanLink(r.db.QueryRow(ctx, `SELECT `+linkCols+` FROM enlaces_oficiales e WHERE e.id=$1`, id))
}
func (r *OfficialLinkRepository) Create(ctx context.Context, l domain.OfficialLink) (domain.OfficialLink, error) {
	return scanLink(r.db.QueryRow(ctx, `INSERT INTO enlaces_oficiales(nombre,descripcion,url,categoria,visible,orden,estado,actualizado_por) VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING `+linkReturningCols, l.Name, l.Description, l.URL, l.Category, l.Visible, l.Order, l.Status, l.UpdatedBy))
}
func (r *OfficialLinkRepository) Update(ctx context.Context, id, actor string, l domain.OfficialLink) (domain.OfficialLink, error) {
	return scanLink(r.db.QueryRow(ctx, `UPDATE enlaces_oficiales SET nombre=$2,descripcion=$3,url=$4,categoria=$5,visible=$6,orden=$7,actualizado_por=$8,actualizado_en=now() WHERE id=$1 RETURNING `+linkReturningCols, id, l.Name, l.Description, l.URL, l.Category, l.Visible, l.Order, actor))
}
func (r *OfficialLinkRepository) SetStatus(ctx context.Context, id, actor, status string) (domain.OfficialLink, error) {
	return scanLink(r.db.QueryRow(ctx, `UPDATE enlaces_oficiales SET estado=$2,visible=CASE WHEN $2='activo' THEN true ELSE false END,actualizado_por=$3,actualizado_en=now() WHERE id=$1 RETURNING `+linkReturningCols, id, status, actor))
}
