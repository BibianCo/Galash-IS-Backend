package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/galash-uptc/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EventRepository struct{ db *pgxpool.Pool }

func NewEventRepository(db *pgxpool.Pool) *EventRepository { return &EventRepository{db: db} }

const eventSQL = `e.id,e.titulo,COALESCE(e.descripcion,''),e.tipo,e.inicia_en,e.termina_en,e.organizador_id,
 COALESCE(u.nombre||' '||COALESCE(u.apellido,''),''),COALESCE(e.lugar,''),COALESCE(e.enlace,''),COALESCE(e.imagen_url,''),COALESCE(e.enlace_inscripcion,''),e.estado,e.creado_en,e.actualizado_en`
const eventReturningSQL = `id,titulo,COALESCE(descripcion,''),tipo,inicia_en,termina_en,organizador_id,
 COALESCE((SELECT u.nombre||' '||COALESCE(u.apellido,'') FROM usuarios u WHERE u.id=organizador_id),''),COALESCE(lugar,''),COALESCE(enlace,''),COALESCE(imagen_url,''),COALESCE(enlace_inscripcion,''),estado,creado_en,actualizado_en`

func eventErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.Is(err, pgx.ErrNoRows) || errors.As(err, &pgErr) && pgErr.Code == "22P02" {
		return domain.ErrEventNotFound
	}
	return err
}
func scanEvent(row pgx.Row) (domain.Event, error) {
	var e domain.Event
	err := row.Scan(&e.ID, &e.Title, &e.Description, &e.Type, &e.StartAt, &e.EndAt, &e.OrganizerID, &e.OrganizerName, &e.Venue, &e.MeetingURL, &e.ImageURL, &e.RegistrationURL, &e.Status, &e.CreatedAt, &e.UpdatedAt)
	return e, eventErr(err)
}
func (r *EventRepository) List(ctx context.Context, typ, status string, from, to *time.Time) ([]domain.Event, error) {
	rows, err := r.db.Query(ctx, `SELECT `+eventSQL+` FROM eventos e JOIN usuarios u ON u.id=e.organizador_id WHERE ($1='' OR e.tipo=$1) AND ($2='' OR e.estado=$2) AND ($3::timestamptz IS NULL OR e.inicia_en >= $3) AND ($4::timestamptz IS NULL OR e.inicia_en <= $4) ORDER BY e.inicia_en`, typ, status, from, to)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]domain.Event, 0)
	for rows.Next() {
		e, scanErr := scanEvent(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
func (r *EventRepository) Find(ctx context.Context, id string) (domain.Event, error) {
	return scanEvent(r.db.QueryRow(ctx, `SELECT `+eventSQL+` FROM eventos e JOIN usuarios u ON u.id=e.organizador_id WHERE e.id=$1`, id))
}
func (r *EventRepository) Create(ctx context.Context, e domain.Event) (domain.Event, error) {
	return scanEvent(r.db.QueryRow(ctx, `INSERT INTO eventos(titulo,descripcion,tipo,inicia_en,termina_en,organizador_id,lugar,enlace,imagen_url,enlace_inscripcion,estado) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING `+eventReturningSQL, e.Title, e.Description, e.Type, e.StartAt, e.EndAt, e.OrganizerID, e.Venue, e.MeetingURL, e.ImageURL, e.RegistrationURL, e.Status))
}
func (r *EventRepository) Update(ctx context.Context, id string, e domain.Event) (domain.Event, error) {
	return scanEvent(r.db.QueryRow(ctx, `UPDATE eventos SET titulo=$2,descripcion=$3,tipo=$4,inicia_en=$5,termina_en=$6,lugar=$7,enlace=$8,imagen_url=$9,enlace_inscripcion=$10,actualizado_en=now() WHERE id=$1 RETURNING `+eventReturningSQL, id, e.Title, e.Description, e.Type, e.StartAt, e.EndAt, e.Venue, e.MeetingURL, e.ImageURL, e.RegistrationURL))
}
func (r *EventRepository) SetStatus(ctx context.Context, id, status string) (domain.Event, error) {
	return scanEvent(r.db.QueryRow(ctx, `UPDATE eventos SET estado=$2,actualizado_en=now() WHERE id=$1 RETURNING `+eventReturningSQL, id, status))
}

func (r *EventRepository) HasConflict(ctx context.Context, e domain.Event, excludeID string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(
		SELECT 1 FROM eventos
		WHERE ($1 = '' OR id::text <> $1)
		  AND estado <> 'cancelado'
		  AND NULLIF(BTRIM(lugar), '') IS NOT NULL
		  AND BTRIM(lugar) = BTRIM($2)
		  AND inicia_en < COALESCE($4::timestamptz, date_trunc('day', $3::timestamptz) + interval '1 day')
		  AND $3::timestamptz < COALESCE(termina_en, date_trunc('day', inicia_en) + interval '1 day')
	)`, excludeID, e.Venue, e.StartAt, e.EndAt).Scan(&exists)
	return exists, err
}
