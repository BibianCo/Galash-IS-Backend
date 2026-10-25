package postgres

import (
	"context"
	"errors"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ContentPageRepository struct{ db *pgxpool.Pool }

func NewContentPageRepository(db *pgxpool.Pool) *ContentPageRepository {
	return &ContentPageRepository{db: db}
}

const pageCols = `p.id,p.slug,p.titulo,p.contenido,p.estado,p.autor_id,COALESCE(u.nombre||' '||COALESCE(u.apellido,''),''),p.creado_en,p.actualizado_en,p.publicado_en`
const pageReturningCols = `id,slug,titulo,contenido,estado,autor_id,COALESCE((SELECT u.nombre||' '||COALESCE(u.apellido,'') FROM usuarios u WHERE u.id=autor_id),''),creado_en,actualizado_en,publicado_en`

func scanPage(row pgx.Row) (domain.ContentPage, error) {
	var p domain.ContentPage
	err := row.Scan(&p.ID, &p.Slug, &p.Title, &p.Content, &p.Status, &p.AuthorID, &p.AuthorName, &p.CreatedAt, &p.UpdatedAt, &p.PublishedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return p, domain.ErrContentNotFound
	}
	return p, err
}
func (r *ContentPageRepository) List(ctx context.Context, drafts bool) ([]domain.ContentPage, error) {
	status := "publicado"
	if drafts {
		status = ""
	}
	rows, err := r.db.Query(ctx, `SELECT `+pageCols+` FROM contenido_paginas p JOIN usuarios u ON u.id=p.autor_id WHERE ($1='' OR p.estado=$1) ORDER BY p.slug`, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.ContentPage{}
	for rows.Next() {
		p, e := scanPage(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, p)
	}
	return out, rows.Err()
}
func (r *ContentPageRepository) FindBySlug(ctx context.Context, slug string) (domain.ContentPage, error) {
	return scanPage(r.db.QueryRow(ctx, `SELECT `+pageCols+` FROM contenido_paginas p JOIN usuarios u ON u.id=p.autor_id WHERE p.slug=$1 AND p.estado='publicado'`, slug))
}
func (r *ContentPageRepository) Find(ctx context.Context, id string) (domain.ContentPage, error) {
	return scanPage(r.db.QueryRow(ctx, `SELECT `+pageCols+` FROM contenido_paginas p JOIN usuarios u ON u.id=p.autor_id WHERE p.id=$1`, id))
}
func (r *ContentPageRepository) Create(ctx context.Context, p domain.ContentPage) (domain.ContentPage, error) {
	return scanPage(r.db.QueryRow(ctx, `INSERT INTO contenido_paginas(slug,titulo,contenido,estado,autor_id) VALUES($1,$2,$3,$4,$5) RETURNING `+pageReturningCols, p.Slug, p.Title, p.Content, p.Status, p.AuthorID))
}
func (r *ContentPageRepository) Update(ctx context.Context, id, title, content, slug string) (domain.ContentPage, error) {
	return scanPage(r.db.QueryRow(ctx, `UPDATE contenido_paginas SET slug=$2,titulo=$3,contenido=$4,actualizado_en=now() WHERE id=$1 RETURNING `+pageReturningCols, id, slug, title, content))
}
func (r *ContentPageRepository) Publish(ctx context.Context, id string) (domain.ContentPage, error) {
	return scanPage(r.db.QueryRow(ctx, `UPDATE contenido_paginas SET estado='publicado',publicado_en=now(),actualizado_en=now() WHERE id=$1 RETURNING `+pageReturningCols, id))
}
