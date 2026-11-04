package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type AnnouncementRepository struct{ db *pgxpool.Pool }

func NewAnnouncementRepository(db *pgxpool.Pool) *AnnouncementRepository {
	return &AnnouncementRepository{db: db}
}

const announcementCols = `c.id,c.titulo,c.descripcion,c.adjuntos,c.estado,c.autor_id,COALESCE(u.nombre||' '||COALESCE(u.apellido,''),''),c.creado_en,c.actualizado_en,c.publicado_en,EXISTS(SELECT 1 FROM comunicados_lecturas l WHERE l.comunicado_id=c.id AND l.usuario_id=$1)`

func announcementErr(err error) error {
	var p *pgconn.PgError
	if errors.Is(err, pgx.ErrNoRows) || errors.As(err, &p) && p.Code == "22P02" {
		return domain.ErrAnnouncementNotFound
	}
	return err
}
func scanAnnouncement(row pgx.Row) (domain.Announcement, error) {
	var a domain.Announcement
	var raw []byte
	err := row.Scan(&a.ID, &a.Title, &a.Description, &raw, &a.Status, &a.AuthorID, &a.AuthorName, &a.CreatedAt, &a.UpdatedAt, &a.PublishedAt, &a.ReadConfirmed)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return a, domain.ErrAnnouncementNotFound
		}
		return a, announcementErr(err)
	}
	if err := json.Unmarshal(raw, &a.Attachments); err != nil {
		return a, err
	}
	if a.Attachments == nil {
		a.Attachments = []domain.AnnouncementAttachment{}
	}
	return a, nil
}
func (r *AnnouncementRepository) List(ctx context.Context, userID string, includeAllDrafts bool) ([]domain.Announcement, error) {
	rows, err := r.db.Query(ctx, `SELECT `+announcementCols+` FROM comunicados c JOIN usuarios u ON u.id=c.autor_id WHERE c.estado='publicado' OR c.autor_id=$1 OR ($2 AND c.estado='borrador') ORDER BY c.publicado_en DESC NULLS LAST,c.creado_en DESC`, userID, includeAllDrafts)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Announcement{}
	for rows.Next() {
		a, e := scanAnnouncement(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
func (r *AnnouncementRepository) Find(ctx context.Context, id, userID string, includeAllDrafts bool) (domain.Announcement, error) {
	return scanAnnouncement(r.db.QueryRow(ctx, `SELECT `+announcementCols+` FROM comunicados c JOIN usuarios u ON u.id=c.autor_id WHERE c.id=$2 AND (c.estado='publicado' OR c.autor_id=$1 OR ($3 AND c.estado='borrador'))`, userID, id, includeAllDrafts))
}
func (r *AnnouncementRepository) Create(ctx context.Context, a domain.Announcement) (domain.Announcement, error) {
	raw, _ := json.Marshal(a.Attachments)
	return scanAnnouncement(r.db.QueryRow(ctx, `INSERT INTO comunicados(titulo,descripcion,adjuntos,estado,autor_id) VALUES($2,$3,$4,'borrador',$1) RETURNING id,titulo,descripcion,adjuntos,estado,autor_id,'',creado_en,actualizado_en,publicado_en,false`, a.AuthorID, a.Title, a.Description, raw))
}
func (r *AnnouncementRepository) Update(ctx context.Context, id, title, description string, attachments []domain.AnnouncementAttachment) (domain.Announcement, error) {
	raw, _ := json.Marshal(attachments)
	return scanAnnouncement(r.db.QueryRow(ctx, `UPDATE comunicados SET titulo=$2,descripcion=$3,adjuntos=$4,actualizado_en=now() WHERE id=$1 AND estado='borrador' RETURNING id,titulo,descripcion,adjuntos,estado,autor_id,'',creado_en,actualizado_en,publicado_en,false`, id, title, description, raw))
}
func (r *AnnouncementRepository) Publish(ctx context.Context, id string) (domain.Announcement, error) {
	return scanAnnouncement(r.db.QueryRow(ctx, `UPDATE comunicados SET estado='publicado',publicado_en=now(),actualizado_en=now() WHERE id=$1 AND estado='borrador' RETURNING id,titulo,descripcion,adjuntos,estado,autor_id,'',creado_en,actualizado_en,publicado_en,false`, id))
}
func (r *AnnouncementRepository) ConfirmRead(ctx context.Context, id, userID string) error {
	_, err := r.db.Exec(ctx, `INSERT INTO comunicados_lecturas(comunicado_id,usuario_id) SELECT $1,$2 WHERE EXISTS(SELECT 1 FROM comunicados WHERE id=$1 AND estado='publicado') ON CONFLICT (comunicado_id,usuario_id) DO NOTHING`, id, userID)
	return announcementErr(err)
}
func (r *AnnouncementRepository) ReadStatus(ctx context.Context, id string) ([]domain.AnnouncementReadStatus, error) {
	rows, err := r.db.Query(ctx, `SELECT u.id,u.nombre||' '||COALESCE(u.apellido,''),u.email,CASE WHEN l.confirmado_en IS NULL THEN 'Pendiente' ELSE 'Leído' END,l.confirmado_en FROM usuarios u LEFT JOIN comunicados_lecturas l ON l.usuario_id=u.id AND l.comunicado_id=$1 WHERE lower(u.rol)='docente' AND u.estado='activo' ORDER BY u.nombre,u.apellido`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.AnnouncementReadStatus{}
	for rows.Next() {
		var s domain.AnnouncementReadStatus
		if err := rows.Scan(&s.UserID, &s.Name, &s.Email, &s.Status, &s.ConfirmedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
