package postgres

import (
	"context"
	"errors"

	"github.com/galash-uptc/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type FileRepository struct{ db *pgxpool.Pool }

func NewFileRepository(db *pgxpool.Pool) *FileRepository { return &FileRepository{db: db} }

func scanFile(row pgx.Row, content *[]byte) (domain.FileAttachment, error) {
	var file domain.FileAttachment
	err := row.Scan(&file.ID, &file.EntityType, &file.EntityID, &file.Name, &file.ContentType, &file.Size, &file.UploadedBy, &file.CreatedAt, content)
	var pgErr *pgconn.PgError
	if errors.Is(err, pgx.ErrNoRows) || errors.As(err, &pgErr) && pgErr.Code == "22P02" {
		return file, domain.ErrFileNotFound
	}
	return file, err
}

// entityAccess mirrors each entity's read rule (D032): Admin always; announcements
// and events when published or owned; activities/evidence and projects are
// publicly readable; proposals only for their author or teachers. It expects
// u (active actor), $2 actor id and t/e as entity type and id.
const entityAccess = `(LOWER(u.rol)='admin' AND (t IN ('activity','evidence') AND EXISTS (SELECT 1 FROM actividades x WHERE x.id=e)
	OR t='project' AND EXISTS (SELECT 1 FROM proyectos x WHERE x.id=e)
	OR t='proposal' AND EXISTS (SELECT 1 FROM propuestas_investigacion x WHERE x.id=e)
	OR t='announcement' AND EXISTS (SELECT 1 FROM comunicados x WHERE x.id=e)
	OR t='event' AND EXISTS (SELECT 1 FROM eventos x WHERE x.id=e))
	OR t='announcement' AND EXISTS (SELECT 1 FROM comunicados x WHERE x.id=e AND (x.estado='publicado' OR x.autor_id=$2))
	OR t='event' AND EXISTS (SELECT 1 FROM eventos x WHERE x.id=e AND (x.estado='publicado' OR x.organizador_id=$2))
	OR t IN ('activity','evidence') AND EXISTS (SELECT 1 FROM actividades x WHERE x.id=e)
	OR t='project' AND EXISTS (SELECT 1 FROM proyectos x WHERE x.id=e)
	OR t='proposal' AND EXISTS (SELECT 1 FROM propuestas_investigacion x WHERE x.id=e AND (x.autor_id=$2 OR LOWER(u.rol) IN ('docente','profesor'))))`

// Create stores the file only when the uploader can access the target entity.
func (r *FileRepository) Create(ctx context.Context, file domain.FileAttachment, content []byte) (domain.FileAttachment, error) {
	var ignored []byte
	created, err := scanFile(r.db.QueryRow(ctx, `INSERT INTO archivos_adjuntos(entidad_tipo,entidad_id,nombre,tipo_contenido,tamano,contenido,subido_por)
		SELECT $3,$4::uuid,$1,$5,$6,$7,$2 FROM usuarios u, (SELECT $3::text AS t, $4::uuid AS e) target
		WHERE u.id=$2 AND u.estado='activo' AND `+entityAccess+`
		RETURNING id,entidad_tipo,entidad_id,nombre,tipo_contenido,tamano,subido_por,creado_en,''::bytea`,
		file.Name, file.UploadedBy, file.EntityType, file.EntityID, file.ContentType, file.Size, content), &ignored)
	if errors.Is(err, domain.ErrFileNotFound) {
		return created, domain.ErrForbidden
	}
	return created, err
}

func (r *FileRepository) Find(ctx context.Context, id, actorID, role string) (domain.FileAttachment, []byte, error) {
	var content []byte
	file, err := scanFile(r.db.QueryRow(ctx, `SELECT a.id,a.entidad_tipo,a.entidad_id,a.nombre,a.tipo_contenido,a.tamano,a.subido_por,a.creado_en,a.contenido
		FROM archivos_adjuntos a JOIN usuarios u ON u.id=$2, LATERAL (SELECT a.entidad_tipo AS t, a.entidad_id AS e) target
		WHERE a.id=$1 AND u.estado='activo' AND (a.subido_por=$2 OR `+entityAccess+`)`, id, actorID), &content)
	if err != nil {
		return file, nil, err
	}
	return file, content, nil
}
