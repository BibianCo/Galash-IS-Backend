package postgres

import (
	"context"
	"errors"
	"strings"

	"github.com/galash-uptc/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RegistrationRepository struct{ db *pgxpool.Pool }

func NewRegistrationRepository(db *pgxpool.Pool) *RegistrationRepository { return &RegistrationRepository{db: db} }

const registrationFields = `id,COALESCE(usuario_id::text,''),COALESCE(semillero_id::text,''),nombre,cc,cod,telefono,fecha_nacimiento::text,lugar_nacimiento,direccion_residencia,link_cvlac,link_orcid,maximo_nivel_estudio,titulo_maximo_nivel_estudio,titulo_pregrado,vinculacion,categoria_actual_investigador,convocatoria_minciencias,estado,COALESCE(observacion,''),fecha_radicacion,actualizado_en,COALESCE(responsable_id::text,'')`

func scanRegistration(row pgx.Row) (domain.RegistrationRequest, error) {
	var result domain.RegistrationRequest
	err := row.Scan(&result.ID, &result.UserID, &result.SeedbedID, &result.Name, &result.DocumentNumber, &result.StudentCode, &result.Phone, &result.BirthDate, &result.BirthPlace, &result.Address, &result.CvLAC, &result.ORCID, &result.HighestEducation, &result.HighestDegree, &result.UndergraduateDegree, &result.Affiliation, &result.ResearcherCategory, &result.MinCienciasCall, &result.Status, &result.Observation, &result.CreatedAt, &result.UpdatedAt, &result.ResponsibleID)
	return result, err
}

func (r *RegistrationRepository) CreateRegistration(ctx context.Context, q domain.RegistrationRequest, tokenHash string) (domain.RegistrationRequest, error) {
	row := r.db.QueryRow(ctx, `INSERT INTO solicitudes_registro (usuario_id,semillero_id,nombre,cc,cod,telefono,fecha_nacimiento,lugar_nacimiento,direccion_residencia,link_cvlac,link_orcid,maximo_nivel_estudio,titulo_maximo_nivel_estudio,titulo_pregrado,vinculacion,categoria_actual_investigador,convocatoria_minciencias,estado,token_hash) VALUES (NULLIF($1,'')::uuid,NULLIF($2,'')::uuid,$3,$4,$5,$6,$7::date,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19) RETURNING `+registrationFields, q.UserID, q.SeedbedID, q.Name, q.DocumentNumber, q.StudentCode, q.Phone, q.BirthDate, q.BirthPlace, q.Address, q.CvLAC, q.ORCID, q.HighestEducation, q.HighestDegree, q.UndergraduateDegree, q.Affiliation, q.ResearcherCategory, q.MinCienciasCall, domain.RegistrationFiled, tokenHash)
	created, err := scanRegistration(row)
	if err != nil {
		if strings.Contains(err.Error(), "semillero") {
			return domain.RegistrationRequest{}, domain.ErrSeedbedNotFound
		}
		return domain.RegistrationRequest{}, err
	}
	_, err = r.db.Exec(ctx, `INSERT INTO seguimiento_solicitud (solicitud_id,estado,comentario,responsable_id) VALUES ($1,$2,'Solicitud radicada',NULL)`, created.ID, created.Status)
	return created, err
}

func (r *RegistrationRepository) GetRegistrationStatus(ctx context.Context, id, tokenHash, userID string) (domain.RegistrationRequest, error) {
	var row pgx.Row
	if userID != "" {
		row = r.db.QueryRow(ctx, `SELECT `+registrationFields+` FROM solicitudes_registro WHERE id=$1 AND usuario_id=$2`, id, userID)
	} else {
		row = r.db.QueryRow(ctx, `SELECT `+registrationFields+` FROM solicitudes_registro WHERE id=$1 AND token_hash=$2`, id, tokenHash)
	}
	result, err := scanRegistration(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.RegistrationRequest{}, domain.ErrRegistrationNotFound
	}
	return result, err
}

func (r *RegistrationRepository) ListRegistrations(ctx context.Context, status, from, to, seedbedID, responsibleID string) ([]domain.RegistrationRequest, error) {
	where, args := []string{"TRUE"}, []any{}
	add := func(value, clause string) { if strings.TrimSpace(value) != "" { args = append(args, value); where = append(where, clause+itoa(len(args))) } }
	add(status, "estado=$")
	add(seedbedID, "semillero_id=$")
	add(responsibleID, "responsable_id=$")
	if from != "" { args = append(args, from); where = append(where, "fecha_radicacion >= $"+itoa(len(args))+"::date") }
	if to != "" { args = append(args, to); where = append(where, "fecha_radicacion < ($"+itoa(len(args))+"::date + interval '1 day')") }
	rows, err := r.db.Query(ctx, `SELECT `+registrationFields+` FROM solicitudes_registro WHERE `+strings.Join(where, " AND ")+` ORDER BY fecha_radicacion DESC`, args...)
	if err != nil { return nil, err }
	defer rows.Close()
	result := []domain.RegistrationRequest{}
	for rows.Next() { item, err := scanRegistration(rows); if err != nil { return nil, err }; result = append(result, item) }
	return result, rows.Err()
}

func (r *RegistrationRepository) GetRegistrationDetail(ctx context.Context, id string) (domain.RegistrationDetail, error) {
	request, err := scanRegistration(r.db.QueryRow(ctx, `SELECT `+registrationFields+` FROM solicitudes_registro WHERE id=$1`, id))
	if errors.Is(err, pgx.ErrNoRows) { return domain.RegistrationDetail{}, domain.ErrRegistrationNotFound }
	if err != nil { return domain.RegistrationDetail{}, err }
	rows, err := r.db.Query(ctx, `SELECT estado,COALESCE(comentario,''),COALESCE(responsable_id::text,''),fecha FROM seguimiento_solicitud WHERE solicitud_id=$1 ORDER BY fecha`, id)
	if err != nil { return domain.RegistrationDetail{}, err }
	defer rows.Close()
	detail := domain.RegistrationDetail{RegistrationRequest: request, History: []domain.RegistrationHistory{}}
	for rows.Next() { var item domain.RegistrationHistory; if err := rows.Scan(&item.Status, &item.Observation, &item.Responsible, &item.At); err != nil { return domain.RegistrationDetail{}, err }; detail.History = append(detail.History, item) }
	return detail, rows.Err()
}

func (r *RegistrationRepository) UpdateRegistrationStatus(ctx context.Context, id, status, observation, responsibleID string) (domain.RegistrationRequest, error) {
	var current string
	if err := r.db.QueryRow(ctx, `SELECT estado FROM solicitudes_registro WHERE id=$1`, id).Scan(&current); errors.Is(err, pgx.ErrNoRows) { return domain.RegistrationRequest{}, domain.ErrRegistrationNotFound } else if err != nil { return domain.RegistrationRequest{}, err }
	if !allowedTransition(current, status) { return domain.RegistrationRequest{}, domain.ErrInvalidRegistrationTransition }
	request, err := scanRegistration(r.db.QueryRow(ctx, `UPDATE solicitudes_registro SET estado=$1,observacion=$2,responsable_id=NULLIF($3,'')::uuid,actualizado_en=now() WHERE id=$4 RETURNING `+registrationFields, status, observation, responsibleID, id))
	if err != nil { return domain.RegistrationRequest{}, err }
	if _, err = r.db.Exec(ctx, `INSERT INTO seguimiento_solicitud (solicitud_id,estado,comentario,responsable_id) VALUES ($1,$2,$3,NULLIF($4,'')::uuid)`, id, status, observation, responsibleID); err != nil { return domain.RegistrationRequest{}, err }
	if status == domain.RegistrationApproved { _, err = r.db.Exec(ctx, `UPDATE usuarios SET rol='estudiante' WHERE estado='activo' AND (id=NULLIF($1,'')::uuid OR dni=$2)`, request.UserID, request.DocumentNumber) }
	if err == nil && request.UserID != "" && (status == domain.RegistrationApproved || status == domain.RegistrationRejected || status == domain.RegistrationPending) { _, err = r.db.Exec(ctx, `INSERT INTO notificaciones (usuario_id,tipo,mensaje) VALUES ($1,'registration-request',$2)`, request.UserID, "La solicitud "+request.ID+" cambió al estado "+status+". "+observation) }
	return request, err
}

func allowedTransition(from, to string) bool {
	if from == domain.RegistrationFiled { return to == domain.RegistrationReview || to == domain.RegistrationPending }
	if from == domain.RegistrationPending { return to == domain.RegistrationReview }
	if from == domain.RegistrationReview { return to == domain.RegistrationApproved || to == domain.RegistrationRejected }
	return false
}
