package postgres

import (
	"context"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"strings"
)

type SeedbedRepository struct{ db *pgxpool.Pool }

func NewSeedbedRepository(db *pgxpool.Pool) *SeedbedRepository { return &SeedbedRepository{db: db} }
func (r *SeedbedRepository) Create(ctx context.Context, s domain.Seedbed) (domain.Seedbed, error) {
	err := r.db.QueryRow(ctx, `INSERT INTO semilleros (nombre,descripcion,linea_investigacion,objetivo,lider_id,email_contacto,estado,visibilidad) VALUES ($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id,creado_en,actualizado_en`, s.Name, s.Description, s.ResearchLine, s.Objective, s.ResponsibleTeacherID, s.ContactEmail, s.Status, s.Visibility).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
	return s, err
}
func scanSeedbed(row pgx.Row) (domain.Seedbed, error) {
	var s domain.Seedbed
	err := row.Scan(&s.ID, &s.Name, &s.Description, &s.ResearchLine, &s.Objective, &s.ResponsibleTeacherID, &s.ContactEmail, &s.Status, &s.Visibility, &s.CreatedAt, &s.UpdatedAt)
	return s, err
}

const seedbedFields = `id,nombre,COALESCE(descripcion,''),COALESCE(linea_investigacion,''),COALESCE(objetivo,''),lider_id,email_contacto,estado,visibilidad,creado_en,actualizado_en`

func (r *SeedbedRepository) Find(ctx context.Context, id string) (domain.Seedbed, error) {
	return scanSeedbed(r.db.QueryRow(ctx, `SELECT `+seedbedFields+` FROM semilleros WHERE id=$1`, id))
}
func (r *SeedbedRepository) Space(ctx context.Context, id string) (domain.SeedbedSpace, error) {
	var space domain.SeedbedSpace
	seedbed, err := r.Find(ctx, id)
	if err != nil {
		return space, err
	}
	space.Seedbed = seedbed
	space.Purpose = seedbed.Objective
	if err := r.db.QueryRow(ctx, `SELECT id,nombre,COALESCE(apellido,''),email FROM usuarios WHERE id=$1`, seedbed.ResponsibleTeacherID).
		Scan(&space.Responsible.ID, &space.Responsible.Name, &space.Responsible.LastName, &space.Responsible.Email); err != nil {
		return domain.SeedbedSpace{}, err
	}
	space.Activities = []domain.ActivitySummary{}
	activityRows, err := r.db.Query(ctx, `SELECT id,titulo,COALESCE(descripcion,''),estado,fecha FROM actividades WHERE semillero_id=$1 ORDER BY fecha NULLS LAST,titulo`, id)
	if err != nil {
		return domain.SeedbedSpace{}, err
	}
	for activityRows.Next() {
		var activity domain.ActivitySummary
		if err := activityRows.Scan(&activity.ID, &activity.Title, &activity.Description, &activity.Status, &activity.Date); err != nil {
			activityRows.Close()
			return domain.SeedbedSpace{}, err
		}
		space.Activities = append(space.Activities, activity)
	}
	if err := activityRows.Err(); err != nil {
		activityRows.Close()
		return domain.SeedbedSpace{}, err
	}
	activityRows.Close()
	space.Projects = []domain.ProjectSummary{}
	projectRows, err := r.db.Query(ctx, `SELECT id,titulo,COALESCE(resumen,''),COALESCE(area_investigacion,''),fecha_inicio,fecha_fin FROM proyectos WHERE semillero_id=$1 ORDER BY fecha_inicio NULLS LAST,titulo`, id)
	if err != nil {
		return domain.SeedbedSpace{}, err
	}
	for projectRows.Next() {
		var project domain.ProjectSummary
		if err := projectRows.Scan(&project.ID, &project.Title, &project.Summary, &project.ResearchArea, &project.StartDate, &project.EndDate); err != nil {
			projectRows.Close()
			return domain.SeedbedSpace{}, err
		}
		space.Projects = append(space.Projects, project)
	}
	if err := projectRows.Err(); err != nil {
		projectRows.Close()
		return domain.SeedbedSpace{}, err
	}
	projectRows.Close()
	return space, nil
}
func (r *SeedbedRepository) List(ctx context.Context, q, line string, page, size int, private bool) ([]domain.Seedbed, error) {
	where := `WHERE estado='Publicado' AND visibilidad='Publicado'`
	args := []any{}
	if private {
		where = `WHERE TRUE`
	}
	if strings.TrimSpace(q) != "" {
		args = append(args, "%"+strings.ToLower(strings.TrimSpace(q))+"%")
		where += ` AND lower(nombre||' '||COALESCE(descripcion,'')) LIKE $` + itoa(len(args))
	}
	if strings.TrimSpace(line) != "" {
		args = append(args, line)
		where += ` AND linea_investigacion=$` + itoa(len(args))
	}
	args = append(args, size, (page-1)*size)
	rows, err := r.db.Query(ctx, `SELECT `+seedbedFields+` FROM semilleros `+where+` ORDER BY nombre LIMIT $`+itoa(len(args)-1)+` OFFSET $`+itoa(len(args)), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Seedbed{}
	for rows.Next() {
		s, err := scanSeedbed(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
func itoa(i int) string {
	if i == 1 {
		return "1"
	}
	if i == 2 {
		return "2"
	}
	if i == 3 {
		return "3"
	}
	if i == 4 {
		return "4"
	}
	if i == 5 {
		return "5"
	}
	return "6"
}
func (r *SeedbedRepository) Update(ctx context.Context, id string, s domain.Seedbed) (domain.Seedbed, error) {
	return scanSeedbed(r.db.QueryRow(ctx, `UPDATE semilleros SET nombre=$1,descripcion=$2,linea_investigacion=$3,objetivo=$4,lider_id=$5,email_contacto=$6,actualizado_en=now() WHERE id=$7 RETURNING `+seedbedFields, s.Name, s.Description, s.ResearchLine, s.Objective, s.ResponsibleTeacherID, s.ContactEmail, id))
}
func (r *SeedbedRepository) SetVisibility(ctx context.Context, id, status string) (domain.Seedbed, error) {
	return scanSeedbed(r.db.QueryRow(ctx, `UPDATE semilleros SET estado=$1,visibilidad=$1,actualizado_en=now() WHERE id=$2 RETURNING `+seedbedFields, status, id))
}
func (r *SeedbedRepository) ListMembers(ctx context.Context, id string, history bool) ([]domain.SeedbedMember, error) {
	where := `AND sm.retirado_en IS NULL`
	if history {
		where = ``
	}
	rows, err := r.db.Query(ctx, `SELECT sm.usuario_id,u.nombre,COALESCE(u.apellido,''),u.rol,sm.ingreso_en,sm.retirado_en FROM semillero_miembros sm JOIN usuarios u ON u.id=sm.usuario_id WHERE sm.semillero_id=$1 `+where+` ORDER BY u.nombre`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.SeedbedMember{}
	for rows.Next() {
		var m domain.SeedbedMember
		if err := rows.Scan(&m.UserID, &m.Name, &m.LastName, &m.Role, &m.JoinedAt, &m.LeftAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}
func (r *SeedbedRepository) AddMember(ctx context.Context, sid, uid string) error {
	_, err := r.db.Exec(ctx, `INSERT INTO semillero_miembros(semillero_id,usuario_id) VALUES($1,$2) ON CONFLICT (semillero_id,usuario_id) DO UPDATE SET retirado_en=NULL`, sid, uid)
	return err
}
func (r *SeedbedRepository) RemoveMember(ctx context.Context, sid, uid string) error {
	res, err := r.db.Exec(ctx, `UPDATE semillero_miembros SET retirado_en=now() WHERE semillero_id=$1 AND usuario_id=$2 AND retirado_en IS NULL`, sid, uid)
	if err == nil && res.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return err
}
