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

type InternalCalendarRepository struct{ db *pgxpool.Pool }

func NewInternalCalendarRepository(db *pgxpool.Pool) *InternalCalendarRepository {
	return &InternalCalendarRepository{db: db}
}

const calendarColumns = `c.id,c.semillero_id,c.tipo,c.titulo,COALESCE(c.descripcion,''),c.inicia_en,c.termina_en,c.organizador_id,c.estado,c.creado_en,c.actualizado_en`
const calendarReturningColumns = `id,semillero_id,tipo,titulo,COALESCE(descripcion,''),inicia_en,termina_en,organizador_id,estado,creado_en,actualizado_en`

func calendarErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.Is(err, pgx.ErrNoRows) || errors.As(err, &pgErr) && pgErr.Code == "22P02" {
		return domain.ErrInternalCalendarNotFound
	}
	return err
}
func scanCalendar(row pgx.Row) (domain.InternalCalendarEvent, error) {
	var e domain.InternalCalendarEvent
	err := row.Scan(&e.ID, &e.SeedbedID, &e.Type, &e.Title, &e.Description, &e.StartAt, &e.EndAt, &e.OrganizerID, &e.Status, &e.CreatedAt, &e.UpdatedAt)
	return e, calendarErr(err)
}
func (r *InternalCalendarRepository) allowed(actorID, role, alias string) (string, []any) {
	if role == "admin" || role == "Admin" {
		return "TRUE", []any{actorID}
	}
	return `EXISTS (SELECT 1 FROM semillero_miembros sm WHERE sm.semillero_id=` + alias + `.semillero_id AND sm.usuario_id=$1 AND sm.retirado_en IS NULL) OR EXISTS (SELECT 1 FROM semilleros s WHERE s.id=` + alias + `.semillero_id AND s.lider_id=$1)`, []any{actorID}
}
func (r *InternalCalendarRepository) List(ctx context.Context, actorID, role, seedbedID string, from, to *time.Time) ([]domain.InternalCalendarEvent, error) {
	access, args := r.allowed(actorID, role, "c")
	args = append(args, seedbedID, from, to)
	rows, err := r.db.Query(ctx, `SELECT `+calendarColumns+` FROM cronograma_interno c WHERE (`+access+`) AND ($`+calendarItoa(len(args)-2)+`='' OR c.semillero_id=$`+calendarItoa(len(args)-2)+`) AND ($`+calendarItoa(len(args)-1)+`::timestamptz IS NULL OR c.inicia_en >= $`+calendarItoa(len(args)-1)+`) AND ($`+calendarItoa(len(args))+`::timestamptz IS NULL OR c.inicia_en <= $`+calendarItoa(len(args))+`) ORDER BY c.inicia_en`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.InternalCalendarEvent{}
	for rows.Next() {
		var e domain.InternalCalendarEvent
		if err := rows.Scan(&e.ID, &e.SeedbedID, &e.Type, &e.Title, &e.Description, &e.StartAt, &e.EndAt, &e.OrganizerID, &e.Status, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		if err := r.loadParticipants(ctx, &e); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// itoa is deliberately local to keep the SQL argument construction explicit.
func calendarItoa(n int) string {
	if n == 1 {
		return "1"
	}
	if n == 2 {
		return "2"
	}
	if n == 3 {
		return "3"
	}
	if n == 4 {
		return "4"
	}
	if n == 5 {
		return "5"
	}
	return "6"
}
func (r *InternalCalendarRepository) Find(ctx context.Context, actorID, role, id string) (domain.InternalCalendarEvent, error) {
	access, _ := r.allowed(actorID, role, "c")
	e, err := scanCalendar(r.db.QueryRow(ctx, `SELECT `+calendarColumns+` FROM cronograma_interno c WHERE c.id=$2 AND (`+access+`)`, actorID, id))
	if err != nil {
		return e, err
	}
	if err := r.loadParticipants(ctx, &e); err != nil {
		return e, err
	}
	return e, nil
}
func (r *InternalCalendarRepository) loadParticipants(ctx context.Context, e *domain.InternalCalendarEvent) error {
	rows, err := r.db.Query(ctx, `SELECT u.id,u.nombre,COALESCE(u.apellido,''),u.rol FROM cronograma_interno_participantes p JOIN usuarios u ON u.id=p.usuario_id WHERE p.evento_id=$1 ORDER BY u.nombre,u.apellido`, e.ID)
	if err != nil {
		return err
	}
	defer rows.Close()
	e.ParticipantIDs = []string{}
	e.Participants = []domain.InternalCalendarPerson{}
	for rows.Next() {
		var p domain.InternalCalendarPerson
		if err := rows.Scan(&p.ID, &p.Name, &p.LastName, &p.Role); err != nil {
			return err
		}
		e.ParticipantIDs = append(e.ParticipantIDs, p.ID)
		e.Participants = append(e.Participants, p)
	}
	return rows.Err()
}
func (r *InternalCalendarRepository) Create(ctx context.Context, e domain.InternalCalendarEvent) (domain.InternalCalendarEvent, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return e, err
	}
	defer tx.Rollback(ctx)
	row := tx.QueryRow(ctx, `INSERT INTO cronograma_interno(semillero_id,tipo,titulo,descripcion,inicia_en,termina_en,organizador_id) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING `+calendarReturningColumns, e.SeedbedID, e.Type, e.Title, e.Description, e.StartAt, e.EndAt, e.OrganizerID)
	var out domain.InternalCalendarEvent
	if err = row.Scan(&out.ID, &out.SeedbedID, &out.Type, &out.Title, &out.Description, &out.StartAt, &out.EndAt, &out.OrganizerID, &out.Status, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return e, err
	}
	for _, id := range e.ParticipantIDs {
		var ok bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM usuarios u JOIN semillero_miembros sm ON sm.usuario_id=u.id WHERE u.id=$1 AND u.estado='activo' AND sm.semillero_id=$2 AND sm.retirado_en IS NULL)`, id, e.SeedbedID).Scan(&ok); err != nil {
			return e, err
		}
		if !ok {
			return e, domain.ErrInvalidInternalCalendar
		}
		if _, err = tx.Exec(ctx, `INSERT INTO cronograma_interno_participantes(evento_id,usuario_id) VALUES($1,$2)`, out.ID, id); err != nil {
			return e, err
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO cronograma_interno_historial(evento_id,accion,actor_id) VALUES($1,'creado',$2)`, out.ID, e.OrganizerID); err != nil {
		return e, err
	}
	if err = tx.Commit(ctx); err != nil {
		return e, err
	}
	out.ParticipantIDs = e.ParticipantIDs
	return out, nil
}
func (r *InternalCalendarRepository) Update(ctx context.Context, e domain.InternalCalendarEvent) (domain.InternalCalendarEvent, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return e, err
	}
	defer tx.Rollback(ctx)
	row := tx.QueryRow(ctx, `UPDATE cronograma_interno SET tipo=$2,titulo=$3,descripcion=$4,inicia_en=$5,termina_en=$6,actualizado_en=now() WHERE id=$1 RETURNING `+calendarReturningColumns, e.ID, e.Type, e.Title, e.Description, e.StartAt, e.EndAt)
	var out domain.InternalCalendarEvent
	if err = row.Scan(&out.ID, &out.SeedbedID, &out.Type, &out.Title, &out.Description, &out.StartAt, &out.EndAt, &out.OrganizerID, &out.Status, &out.CreatedAt, &out.UpdatedAt); err != nil {
		return e, calendarErr(err)
	}
	if _, err = tx.Exec(ctx, `DELETE FROM cronograma_interno_participantes WHERE evento_id=$1`, e.ID); err != nil {
		return e, err
	}
	for _, id := range e.ParticipantIDs {
		var ok bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM usuarios u JOIN semillero_miembros sm ON sm.usuario_id=u.id WHERE u.id=$1 AND u.estado='activo' AND sm.semillero_id=$2 AND sm.retirado_en IS NULL)`, id, e.SeedbedID).Scan(&ok); err != nil || !ok {
			if err != nil {
				return e, err
			}
			return e, domain.ErrInvalidInternalCalendar
		}
		if _, err = tx.Exec(ctx, `INSERT INTO cronograma_interno_participantes(evento_id,usuario_id) VALUES($1,$2)`, e.ID, id); err != nil {
			return e, err
		}
	}
	if _, err = tx.Exec(ctx, `INSERT INTO cronograma_interno_historial(evento_id,accion,actor_id) VALUES($1,'actualizado',$2)`, e.ID, e.OrganizerID); err != nil {
		return e, err
	}
	if err = tx.Commit(ctx); err != nil {
		return e, err
	}
	out.ParticipantIDs = e.ParticipantIDs
	return out, nil
}
func (r *InternalCalendarRepository) CanManage(ctx context.Context, e domain.InternalCalendarEvent, actorID, role string) (bool, error) {
	if role == "admin" || role == "Admin" {
		return true, nil
	}
	var ok bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM semilleros WHERE id=$1 AND lider_id=$2) OR $2=$3`, e.SeedbedID, actorID, e.OrganizerID).Scan(&ok)
	return ok, err
}
func (r *InternalCalendarRepository) HasConflict(ctx context.Context, e domain.InternalCalendarEvent, exclude string) (bool, error) {
	var ok bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM cronograma_interno c JOIN cronograma_interno_participantes p ON p.evento_id=c.id WHERE c.id<>NULLIF($1,'')::uuid AND c.estado='activo' AND c.inicia_en<$3 AND c.termina_en>$2 AND p.usuario_id=ANY($4::uuid[]))`, exclude, e.StartAt, e.EndAt, e.ParticipantIDs).Scan(&ok)
	return ok, err
}
func (r *InternalCalendarRepository) Cancel(ctx context.Context, id, actorID string) (domain.InternalCalendarEvent, error) {
	var e domain.InternalCalendarEvent
	err := r.db.QueryRow(ctx, `UPDATE cronograma_interno SET estado='cancelado',actualizado_en=now() WHERE id=$1 RETURNING `+calendarReturningColumns, id).Scan(&e.ID, &e.SeedbedID, &e.Type, &e.Title, &e.Description, &e.StartAt, &e.EndAt, &e.OrganizerID, &e.Status, &e.CreatedAt, &e.UpdatedAt)
	if err != nil {
		return e, calendarErr(err)
	}
	_, err = r.db.Exec(ctx, `INSERT INTO cronograma_interno_historial(evento_id,accion,actor_id) VALUES($1,'cancelado',$2)`, id, actorID)
	return e, err
}
