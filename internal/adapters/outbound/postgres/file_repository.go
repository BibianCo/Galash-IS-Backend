package postgres

import (
	"context"
	"errors"

	"github.com/galash-uptc/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FileRepository struct{ db *pgxpool.Pool }

func NewFileRepository(db *pgxpool.Pool) *FileRepository { return &FileRepository{db: db} }

func scanFile(row pgx.Row, content *[]byte) (domain.FileAttachment, error) {
	var file domain.FileAttachment
	err := row.Scan(&file.ID, &file.EntityType, &file.EntityID, &file.Name, &file.ContentType, &file.Size, &file.UploadedBy, &file.CreatedAt, content)
	if errors.Is(err, pgx.ErrNoRows) {
		return file, domain.ErrFileNotFound
	}
	return file, err
}

func (r *FileRepository) Create(ctx context.Context, file domain.FileAttachment, content []byte) (domain.FileAttachment, error) {
	var ignored []byte
	return scanFile(r.db.QueryRow(ctx, `INSERT INTO archivos_adjuntos(entidad_tipo,entidad_id,nombre,tipo_contenido,tamano,contenido,subido_por) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id,entidad_tipo,entidad_id,nombre,tipo_contenido,tamano,subido_por,creado_en,''::bytea`, file.EntityType, file.EntityID, file.Name, file.ContentType, file.Size, content, file.UploadedBy), &ignored)
}

func (r *FileRepository) Find(ctx context.Context, id, actorID, role string) (domain.FileAttachment, []byte, error) {
	var content []byte
	file, err := scanFile(r.db.QueryRow(ctx, `SELECT a.id,a.entidad_tipo,a.entidad_id,a.nombre,a.tipo_contenido,a.tamano,a.subido_por,a.creado_en,a.contenido FROM archivos_adjuntos a JOIN usuarios u ON u.id=$2 WHERE a.id=$1 AND u.estado='activo' AND (LOWER(u.rol)='admin' OR a.subido_por=$2 OR (a.entidad_tipo='announcement' AND EXISTS (SELECT 1 FROM comunicados c WHERE c.id=a.entidad_id AND c.estado='publicado')) OR (a.entidad_tipo='event' AND EXISTS (SELECT 1 FROM eventos e WHERE e.id=a.entidad_id AND e.estado='publicado')) OR a.entidad_tipo IN ('activity','project','proposal','evidence') AND LOWER(u.rol) IN ('estudiante','docente','admin'))`, id, actorID), &content)
	if err != nil {
		return file, nil, err
	}
	return file, content, nil
}
