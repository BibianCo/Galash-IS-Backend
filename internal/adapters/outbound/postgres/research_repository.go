package postgres

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/galash-uptc/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ResearchRepository struct{ db *pgxpool.Pool }

func NewResearchRepository(db *pgxpool.Pool) *ResearchRepository { return &ResearchRepository{db} }

func act(row pgx.Row) (domain.Activity, error) {
	var a domain.Activity
	e := row.Scan(&a.ID, &a.SeedbedID, &a.Title, &a.Description, &a.Status, &a.Date, &a.ResponsibleIDs, &a.Evidence, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(e, pgx.ErrNoRows) {
		e = domain.ErrResearchNotFound
	}
	return a, e
}

func (r *ResearchRepository) ListActivities(c context.Context, s, q, st string) (out []domain.Activity, e error) {
	rows, e := r.db.Query(c, `SELECT id,semillero_id,titulo,COALESCE(descripcion,''),estado,fecha,responsables,evidencias,creado_en,actualizado_en FROM actividades WHERE semillero_id=$1 AND ($2='' OR estado=$2) AND ($3='' OR fecha::text LIKE '%'||$3||'%') ORDER BY fecha NULLS LAST,titulo`, s, st, q)
	if e != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var a domain.Activity
		if e = rows.Scan(&a.ID, &a.SeedbedID, &a.Title, &a.Description, &a.Status, &a.Date, &a.ResponsibleIDs, &a.Evidence, &a.CreatedAt, &a.UpdatedAt); e != nil {
			return
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *ResearchRepository) CreateActivity(c context.Context, a domain.Activity) (domain.Activity, error) {
	return act(r.db.QueryRow(c, `INSERT INTO actividades(semillero_id,titulo,descripcion,estado,fecha,responsables,evidencias) VALUES($1,$2,$3,$4,$5,COALESCE($6::text[],'{}'),COALESCE($7::text[],'{}')) RETURNING id,semillero_id,titulo,COALESCE(descripcion,''),estado,fecha,responsables,evidencias,creado_en,actualizado_en`, a.SeedbedID, a.Title, a.Description, a.Status, a.Date, a.ResponsibleIDs, a.Evidence))
}

func (r *ResearchRepository) FindActivity(c context.Context, id string) (domain.Activity, error) {
	return act(r.db.QueryRow(c, `SELECT id,semillero_id,titulo,COALESCE(descripcion,''),estado,fecha,responsables,evidencias,creado_en,actualizado_en FROM actividades WHERE id=$1`, id))
}

func (r *ResearchRepository) UpdateActivity(c context.Context, id string, a domain.Activity) (domain.Activity, error) {
	return act(r.db.QueryRow(c, `UPDATE actividades SET titulo=$2,descripcion=$3,estado=$4,fecha=$5,responsables=COALESCE($6::text[],'{}'),evidencias=COALESCE($7::text[],'{}'),actualizado_en=now() WHERE id=$1 RETURNING id,semillero_id,titulo,COALESCE(descripcion,''),estado,fecha,responsables,evidencias,creado_en,actualizado_en`, id, a.Title, a.Description, a.Status, a.Date, a.ResponsibleIDs, a.Evidence))
}

func proj(row pgx.Row) (domain.Project, error) {
	var p domain.Project
	e := row.Scan(&p.ID, &p.SeedbedID, &p.Title, &p.Summary, &p.ResearchLine, &p.Status, &p.ResponsibleIDs, &p.MemberIDs, &p.StartDate, &p.EndDate, &p.Progress, &p.CreatedBy, &p.UpdatedAt)
	if errors.Is(e, pgx.ErrNoRows) {
		e = domain.ErrResearchNotFound
	}
	return p, e
}

const projectSQL = `id,semillero_id,titulo,COALESCE(resumen,''),COALESCE(area_investigacion,''),estado,responsables,integrantes,fecha_inicio,fecha_fin,avance,COALESCE(creado_por::text,''),actualizado_en`

func (r *ResearchRepository) ListProjects(c context.Context, s, st, period string) (out []domain.Project, e error) {
	out = make([]domain.Project, 0)
	rows, e := r.db.Query(c, `SELECT `+projectSQL+` FROM proyectos WHERE ($1::text='' OR semillero_id::text=$1) AND ($2='' OR estado=$2) AND ($3='' OR EXTRACT(YEAR FROM fecha_inicio)::text=$3) ORDER BY fecha_inicio NULLS LAST,titulo`, s, st, period)
	if e != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		p, err := proj(rows)
		if err != nil {
			return out, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *ResearchRepository) CreateProject(c context.Context, p domain.Project) (domain.Project, error) {
	return proj(r.db.QueryRow(c, `INSERT INTO proyectos(semillero_id,titulo,resumen,area_investigacion,estado,responsables,integrantes,fecha_inicio,fecha_fin,avance,creado_por) VALUES($1,$2,$3,$4,$5,COALESCE($6::text[],'{}'),COALESCE($7::text[],'{}'),$8,$9,$10,$11) RETURNING `+projectSQL, p.SeedbedID, p.Title, p.Summary, p.ResearchLine, p.Status, p.ResponsibleIDs, p.MemberIDs, p.StartDate, p.EndDate, p.Progress, p.CreatedBy))
}

func (r *ResearchRepository) FindProject(c context.Context, id string) (domain.Project, error) {
	return proj(r.db.QueryRow(c, `SELECT `+projectSQL+` FROM proyectos WHERE id=$1`, id))
}

func projectHistory(c context.Context, tx pgx.Tx, id, actor, field, value string) error {
	_, e := tx.Exec(c, `INSERT INTO proyecto_historial(proyecto_id,autor_id,campo,valor) VALUES($1,$2,$3,$4)`, id, actor, field, value)
	return e
}

func (r *ResearchRepository) UpdateProject(c context.Context, id string, p domain.Project, actor string, changes []domain.ProjectHistory) (domain.Project, error) {
	tx, e := r.db.Begin(c)
	if e != nil {
		return p, e
	}
	defer tx.Rollback(c)
	out, e := proj(tx.QueryRow(c, `UPDATE proyectos SET titulo=$2,resumen=$3,area_investigacion=$4,estado=$5,responsables=COALESCE($6::text[],'{}'),integrantes=COALESCE($7::text[],'{}'),fecha_inicio=$8,fecha_fin=$9,avance=$10,actualizado_en=now() WHERE id=$1 RETURNING `+projectSQL, id, p.Title, p.Summary, p.ResearchLine, p.Status, p.ResponsibleIDs, p.MemberIDs, p.StartDate, p.EndDate, p.Progress))
	for _, h := range changes {
		if e == nil {
			e = projectHistory(c, tx, id, actor, h.Field, h.Value)
		}
	}
	if e == nil {
		e = tx.Commit(c)
	}
	return out, e
}

func (r *ResearchRepository) UpdateProgress(c context.Context, id string, n int, a string) (domain.Project, error) {
	tx, e := r.db.Begin(c)
	if e != nil {
		return domain.Project{}, e
	}
	defer tx.Rollback(c)
	p, e := proj(tx.QueryRow(c, `UPDATE proyectos SET avance=$2,actualizado_en=now() WHERE id=$1 RETURNING `+projectSQL, id, n))
	if e == nil {
		e = projectHistory(c, tx, id, a, "avance", strconv.Itoa(n))
	}
	if e == nil {
		e = tx.Commit(c)
	}
	return p, e
}

func (r *ResearchRepository) ProjectProgress(c context.Context, s, period string) ([]domain.Project, error) {
	return r.ListProjects(c, s, "", period)
}

func (r *ResearchRepository) ProjectHistory(c context.Context, id string) (out []domain.ProjectHistory, e error) {
	out = make([]domain.ProjectHistory, 0)
	rows, e := r.db.Query(c, `SELECT id,proyecto_id,COALESCE(autor_id::text,''),campo,valor,creado_en FROM proyecto_historial WHERE proyecto_id=$1 ORDER BY creado_en DESC`, id)
	if e != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var h domain.ProjectHistory
		if e = rows.Scan(&h.ID, &h.ProjectID, &h.AuthorID, &h.Field, &h.Value, &h.CreatedAt); e != nil {
			return
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func prop(row pgx.Row) (domain.ResearchProposal, error) {
	var p domain.ResearchProposal
	e := row.Scan(&p.ID, &p.SeedbedID, &p.Title, &p.Problem, &p.Objectives, &p.Methodology, &p.Status, &p.ResponsibleIDs, &p.Attachments, &p.AuthorID, &p.Observations, &p.ProjectID, &p.CreatedAt, &p.UpdatedAt)
	if errors.Is(e, pgx.ErrNoRows) {
		e = domain.ErrResearchNotFound
	}
	return p, e
}

const propSQL = `id,COALESCE(semillero_id::text,''),titulo,problema,objetivos,metodologia,estado,responsables,anexos,autor_id,COALESCE(observaciones,''),proyecto_id::text,creado_en,actualizado_en`

func (r *ResearchRepository) ListProposals(c context.Context, s string) (out []domain.ResearchProposal, e error) {
	out = make([]domain.ResearchProposal, 0)
	rows, e := r.db.Query(c, `SELECT `+propSQL+` FROM propuestas_investigacion WHERE $1::text='' OR autor_id::text=$1 OR $1=ANY(responsables)`, s)
	if e != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		p, err := prop(rows)
		if err != nil {
			return out, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func proposalHistory(c context.Context, tx pgx.Tx, id, actor, status, note string) error {
	_, e := tx.Exec(c, `INSERT INTO propuesta_historial(propuesta_id,autor_id,estado,observacion) VALUES($1,$2,$3,NULLIF($4,''))`, id, actor, status, note)
	return e
}

func (r *ResearchRepository) CreateProposal(c context.Context, p domain.ResearchProposal) (domain.ResearchProposal, error) {
	tx, e := r.db.Begin(c)
	if e != nil {
		return p, e
	}
	defer tx.Rollback(c)
	out, e := prop(tx.QueryRow(c, `INSERT INTO propuestas_investigacion(semillero_id,titulo,problema,objetivos,metodologia,estado,responsables,anexos,autor_id) VALUES(NULLIF($1,'')::uuid,$2,$3,$4,$5,$6,COALESCE($7::text[],'{}'),COALESCE($8::text[],'{}'),$9) RETURNING `+propSQL, p.SeedbedID, p.Title, p.Problem, p.Objectives, p.Methodology, p.Status, p.ResponsibleIDs, p.Attachments, p.AuthorID))
	if e == nil {
		e = proposalHistory(c, tx, out.ID, p.AuthorID, out.Status, "")
	}
	if e == nil {
		e = tx.Commit(c)
	}
	return out, e
}

func (r *ResearchRepository) FindProposal(c context.Context, id string) (domain.ResearchProposal, error) {
	return prop(r.db.QueryRow(c, `SELECT `+propSQL+` FROM propuestas_investigacion WHERE id=$1`, id))
}

func (r *ResearchRepository) UpdateProposal(c context.Context, id string, p domain.ResearchProposal, actor string) (domain.ResearchProposal, error) {
	tx, e := r.db.Begin(c)
	if e != nil {
		return p, e
	}
	defer tx.Rollback(c)
	out, e := prop(tx.QueryRow(c, `UPDATE propuestas_investigacion SET titulo=$2,problema=$3,objetivos=$4,metodologia=$5,estado=$6,responsables=COALESCE($7::text[],'{}'),anexos=COALESCE($8::text[],'{}'),observaciones=$9,actualizado_en=now() WHERE id=$1 RETURNING `+propSQL, id, p.Title, p.Problem, p.Objectives, p.Methodology, p.Status, p.ResponsibleIDs, p.Attachments, p.Observations))
	if e == nil {
		e = proposalHistory(c, tx, id, actor, out.Status, out.Observations)
	}
	if e == nil {
		e = tx.Commit(c)
	}
	return out, e
}

func (r *ResearchRepository) ProposalHistory(c context.Context, id string) (out []domain.ProposalHistory, e error) {
	out = make([]domain.ProposalHistory, 0)
	rows, e := r.db.Query(c, `SELECT id,propuesta_id,COALESCE(autor_id::text,''),COALESCE(estado,''),COALESCE(observacion,''),creado_en FROM propuesta_historial WHERE propuesta_id=$1 ORDER BY creado_en`, id)
	if e != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var h domain.ProposalHistory
		if e = rows.Scan(&h.ID, &h.ProposalID, &h.AuthorID, &h.Status, &h.Observation, &h.CreatedAt); e != nil {
			return
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// ConvertProposal creates the Radicado project and links both records; the proposal and its history stay intact.
func (r *ResearchRepository) ConvertProposal(c context.Context, id, actor string) (domain.Project, error) {
	tx, e := r.db.Begin(c)
	if e != nil {
		return domain.Project{}, e
	}
	defer tx.Rollback(c)
	p, e := proj(tx.QueryRow(c, `INSERT INTO proyectos(semillero_id,titulo,resumen,area_investigacion,estado,responsables,integrantes,avance,creado_por) SELECT semillero_id,titulo,problema,'','Radicado',responsables,'{}',0,autor_id FROM propuestas_investigacion WHERE id=$1 RETURNING `+projectSQL, id))
	if e == nil {
		_, e = tx.Exec(c, `UPDATE propuestas_investigacion SET estado='Convertida',proyecto_id=$2,actualizado_en=now() WHERE id=$1`, id, p.ID)
	}
	if e == nil {
		e = proposalHistory(c, tx, id, actor, "Convertida", "")
	}
	if e == nil {
		e = projectHistory(c, tx, p.ID, actor, "propuesta", id)
	}
	if e == nil {
		e = tx.Commit(c)
	}
	return p, e
}

func review(row pgx.Row) (domain.SeedbedReview, error) {
	var v domain.SeedbedReview
	e := row.Scan(&v.ID, &v.SeedbedID, &v.AuthorID, &v.Text, &v.Status, &v.ModerationNote, &v.CreatedAt, &v.ModeratedAt)
	if errors.Is(e, pgx.ErrNoRows) {
		e = domain.ErrResearchNotFound
	}
	return v, e
}

func (r *ResearchRepository) CreateReview(c context.Context, v domain.SeedbedReview) (domain.SeedbedReview, error) {
	return review(r.db.QueryRow(c, `INSERT INTO semillero_resenas(semillero_id,autor_id,texto) VALUES($1,$2,$3) RETURNING id,semillero_id,autor_id,texto,estado,COALESCE(nota_moderacion,''),creado_en,moderado_en`, v.SeedbedID, v.AuthorID, v.Text))
}

func (r *ResearchRepository) ListReviews(c context.Context, s string) (out []domain.SeedbedReview, e error) {
	rows, e := r.db.Query(c, `SELECT id,semillero_id,autor_id,texto,estado,COALESCE(nota_moderacion,''),creado_en,moderado_en FROM semillero_resenas WHERE $1='' OR estado=$1 ORDER BY creado_en`, s)
	if e != nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var v domain.SeedbedReview
		if e = rows.Scan(&v.ID, &v.SeedbedID, &v.AuthorID, &v.Text, &v.Status, &v.ModerationNote, &v.CreatedAt, &v.ModeratedAt); e != nil {
			return
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *ResearchRepository) ModerateReview(c context.Context, id, status, note string) (domain.SeedbedReview, error) {
	return review(r.db.QueryRow(c, `UPDATE semillero_resenas SET estado=$2,nota_moderacion=$3,moderado_en=$4 WHERE id=$1 RETURNING id,semillero_id,autor_id,texto,estado,COALESCE(nota_moderacion,''),creado_en,moderado_en`, id, status, note, time.Now().UTC()))
}
