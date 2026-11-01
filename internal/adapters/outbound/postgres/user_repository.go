package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	memberapp "github.com/galash-uptc/backend/internal/application/member"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	db *pgxpool.Pool
}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Upsert(ctx context.Context, user domain.User) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `
		INSERT INTO usuarios (firebase_uid, nombre, apellido, tipo_dni, dni, email, codigo_estudiantil, fecha_nacimiento, telefono, rol)
		VALUES (
			$1,
			$2,
			NULLIF($3, ''),
			NULLIF($4, ''),
			NULLIF($5, ''),
			$6,
			NULLIF($7, ''),
			NULLIF($8, '')::date,
			NULLIF($9, ''),
			'visitante'
		)
		ON CONFLICT (firebase_uid) DO UPDATE SET
			nombre = COALESCE(NULLIF(EXCLUDED.nombre, ''), usuarios.nombre),
			apellido = COALESCE(NULLIF(EXCLUDED.apellido, ''), usuarios.apellido),
			tipo_dni = COALESCE(NULLIF(EXCLUDED.tipo_dni, ''), usuarios.tipo_dni),
			dni = COALESCE(NULLIF(EXCLUDED.dni, ''), usuarios.dni),
			email = EXCLUDED.email,
			codigo_estudiantil = COALESCE(EXCLUDED.codigo_estudiantil, usuarios.codigo_estudiantil),
			fecha_nacimiento = COALESCE(EXCLUDED.fecha_nacimiento, usuarios.fecha_nacimiento),
			telefono = COALESCE(EXCLUDED.telefono, usuarios.telefono)
		RETURNING id`,
		user.FirebaseUID, user.Name, user.LastName, user.DocumentType, user.DocumentNumber, user.Email,
		user.StudentCode, user.BirthDate, user.Phone,
	).Scan(&id)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return "", domain.ErrDuplicateIdentity
	}
	return id, err
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (domain.User, error) {
	return r.find(ctx, `WHERE id = $1`, id)
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (domain.User, error) {
	return r.find(ctx, `WHERE lower(email) = lower($1)`, email)
}

func (r *UserRepository) FindByFirebaseUID(ctx context.Context, uid string) (domain.User, error) {
	return r.find(ctx, `WHERE firebase_uid = $1`, uid)
}

func (r *UserRepository) find(ctx context.Context, where string, arg string) (domain.User, error) {
	var user domain.User
	err := r.db.QueryRow(ctx, `SELECT id, firebase_uid, nombre, COALESCE(apellido, ''), email, COALESCE(tipo_dni, ''), COALESCE(dni, ''), COALESCE(codigo_estudiantil, ''), COALESCE(fecha_nacimiento::text, ''), COALESCE(telefono, ''), rol, estado FROM usuarios `+where, arg).
		Scan(&user.ID, &user.FirebaseUID, &user.Name, &user.LastName, &user.Email, &user.DocumentType, &user.DocumentNumber, &user.StudentCode, &user.BirthDate, &user.Phone, &user.Role, &user.Status)
	return user, err
}

func (r *UserRepository) SetPassword(ctx context.Context, id, passwordHash string) error {
	_, err := r.db.Exec(ctx, `UPDATE usuarios SET password_hash = $1 WHERE id = $2`, passwordHash, id)
	return err
}

func (r *UserRepository) Search(ctx context.Context, query string, limit int, includeInactive bool) ([]domain.User, error) {
	pattern := "%" + strings.ToLower(strings.TrimSpace(query)) + "%"
	rows, err := r.db.Query(ctx, `SELECT id, firebase_uid, nombre, COALESCE(apellido, ''), email, COALESCE(tipo_dni, ''), COALESCE(dni, ''), rol, estado
		FROM usuarios WHERE ($3 OR estado = 'activo') AND (lower(nombre || ' ' || COALESCE(apellido, '')) LIKE $1 OR lower(email) LIKE $1)
		ORDER BY nombre, apellido, email LIMIT $2`, pattern, limit, includeInactive)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	users := make([]domain.User, 0, limit)
	for rows.Next() {
		var user domain.User
		if err := rows.Scan(&user.ID, &user.FirebaseUID, &user.Name, &user.LastName, &user.Email, &user.DocumentType, &user.DocumentNumber, &user.Role, &user.Status); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func (r *UserRepository) UpdateRole(ctx context.Context, firebaseUID, role string) error {
	result, err := r.db.Exec(ctx, `UPDATE usuarios SET rol = $1 WHERE firebase_uid = $2 AND estado = 'activo'`, role, firebaseUID)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func memberSelect() string {
	return `SELECT u.id, u.firebase_uid, u.nombre, COALESCE(u.apellido,''), u.email, COALESCE(u.tipo_dni,''), COALESCE(u.dni,''), u.rol, u.estado, COALESCE(p.area_investigacion,''), COALESCE(p.formacion,''), COALESCE(p.orcid,''), COALESCE(p.cvlac,''), COALESCE(p.visible_publicamente, true), COALESCE((SELECT array_agg(sm.semillero_id::text ORDER BY sm.semillero_id) FROM semillero_miembros sm WHERE sm.usuario_id = u.id AND sm.retirado_en IS NULL), ARRAY[]::text[]) FROM usuarios u LEFT JOIN perfil_investigador p ON p.usuario_id = u.id `
}

func scanMember(row interface{ Scan(...any) error }) (domain.MemberProfile, error) {
	var p domain.MemberProfile
	err := row.Scan(&p.ID, &p.FirebaseUID, &p.Name, &p.LastName, &p.Email, &p.DocumentType, &p.DocumentNumber, &p.Role, &p.Status, &p.Area, &p.Formation, &p.ORCID, &p.CvLAC, &p.VisiblePublicly, &p.SeedbedIDs)
	return p, err
}

func (r *UserRepository) List(ctx context.Context, f ports.MemberFilter) ([]domain.MemberProfile, error) {
	where := []string{"u.estado = 'activo'", "COALESCE(p.visible_publicamente, true) = true"}
	args := []any{}
	if q := strings.TrimSpace(f.Query); q != "" {
		args = append(args, "%"+strings.ToLower(q)+"%")
		where = append(where, "(lower(u.nombre || ' ' || COALESCE(u.apellido,'')) LIKE $"+fmt.Sprint(len(args))+" OR lower(u.email) LIKE $"+fmt.Sprint(len(args))+")")
	}
	if f.Role != "" {
		args = append(args, f.Role)
		where = append(where, "lower(u.rol) = lower($"+fmt.Sprint(len(args))+")")
	}
	if f.Area != "" {
		args = append(args, "%"+strings.ToLower(f.Area)+"%")
		where = append(where, "lower(COALESCE(p.area_investigacion,'')) LIKE $"+fmt.Sprint(len(args)))
	}
	if f.SeedbedID != "" {
		args = append(args, f.SeedbedID)
		where = append(where, "EXISTS (SELECT 1 FROM semillero_miembros smf WHERE smf.usuario_id=u.id AND smf.semillero_id=$"+fmt.Sprint(len(args))+" AND smf.retirado_en IS NULL)")
	}
	args = append(args, f.Limit)
	rows, err := r.db.Query(ctx, memberSelect()+"WHERE "+strings.Join(where, " AND ")+" ORDER BY u.nombre, u.apellido LIMIT $"+fmt.Sprint(len(args)), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.MemberProfile, 0)
	for rows.Next() {
		p, err := scanMember(rows)
		if err != nil {
			return nil, err
		}
		p.FirebaseUID = ""
		p.Email = ""
		p.DocumentType = ""
		p.DocumentNumber = ""
		items = append(items, p)
	}
	return items, rows.Err()
}

func (r *UserRepository) Find(ctx context.Context, id string) (domain.MemberProfile, error) {
	p, err := scanMember(r.db.QueryRow(ctx, memberSelect()+"WHERE u.id=$1", id))
	if err == pgx.ErrNoRows {
		return domain.MemberProfile{}, domain.ErrMemberNotFound
	}
	return p, err
}

func (r *UserRepository) CreateMember(ctx context.Context, c memberapp.CreateCommand) (domain.MemberProfile, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return domain.MemberProfile{}, err
	}
	defer tx.Rollback(ctx)
	var id string
	err = tx.QueryRow(ctx, `INSERT INTO usuarios(firebase_uid,nombre,apellido,tipo_dni,dni,email,rol) VALUES($1,$2,$3,NULLIF($4,''),NULLIF($5,''),$6,$7) RETURNING id`, c.FirebaseUID, c.Name, c.LastName, c.DocumentType, c.DocumentNumber, c.Email, strings.ToLower(c.Role)).Scan(&id)
	if err != nil {
		return domain.MemberProfile{}, err
	}
	_, err = tx.Exec(ctx, `INSERT INTO perfil_investigador(usuario_id,area_investigacion,formacion,orcid,cvlac,visible_publicamente) VALUES($1,$2,$3,$4,$5,$6)`, id, c.Area, c.Formation, c.ORCID, c.CvLAC, c.VisiblePublicly)
	if err != nil {
		return domain.MemberProfile{}, err
	}
	for _, seedbedID := range c.SeedbedIDs {
		if _, err = tx.Exec(ctx, `INSERT INTO semillero_miembros(semillero_id,usuario_id) VALUES($1,$2)`, seedbedID, id); err != nil {
			return domain.MemberProfile{}, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.MemberProfile{}, err
	}
	return r.Find(ctx, id)
}

func (r *UserRepository) Update(ctx context.Context, id string, c memberapp.UpdateCommand) (domain.MemberProfile, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return domain.MemberProfile{}, err
	}
	defer tx.Rollback(ctx)
	result, err := tx.Exec(ctx, `UPDATE usuarios SET nombre=COALESCE(NULLIF($1,''),nombre), apellido=COALESCE(NULLIF($2,''),apellido), email=COALESCE(NULLIF($3,''),email), tipo_dni=COALESCE(NULLIF($4,''),tipo_dni), dni=COALESCE(NULLIF($5,''),dni), rol=COALESCE(NULLIF($6,''),rol) WHERE id=$7`, c.Name, c.LastName, c.Email, c.DocumentType, c.DocumentNumber, strings.ToLower(c.Role), id)
	if err != nil {
		return domain.MemberProfile{}, err
	}
	if result.RowsAffected() == 0 {
		return domain.MemberProfile{}, domain.ErrMemberNotFound
	}
	_, err = tx.Exec(ctx, `INSERT INTO perfil_investigador(usuario_id,area_investigacion,formacion,orcid,cvlac,visible_publicamente) VALUES($6,$1,$2,$3,$4,COALESCE($5,true)) ON CONFLICT(usuario_id) DO UPDATE SET area_investigacion=COALESCE(NULLIF($1,''),perfil_investigador.area_investigacion), formacion=COALESCE(NULLIF($2,''),perfil_investigador.formacion), orcid=COALESCE(NULLIF($3,''),perfil_investigador.orcid), cvlac=COALESCE(NULLIF($4,''),perfil_investigador.cvlac), visible_publicamente=COALESCE($5,perfil_investigador.visible_publicamente)`, c.Area, c.Formation, c.ORCID, c.CvLAC, c.VisiblePublicly, id)
	if err != nil {
		return domain.MemberProfile{}, err
	}
	if c.SeedbedIDs != nil {
		if _, err = tx.Exec(ctx, `UPDATE semillero_miembros SET retirado_en=now() WHERE usuario_id=$1 AND retirado_en IS NULL AND (cardinality($2::text[]) = 0 OR NOT (semillero_id::text = ANY($2::text[])))`, id, c.SeedbedIDs); err != nil {
			return domain.MemberProfile{}, err
		}
		for _, sid := range c.SeedbedIDs {
			if _, err = tx.Exec(ctx, `INSERT INTO semillero_miembros(semillero_id,usuario_id) VALUES($1,$2) ON CONFLICT (semillero_id,usuario_id) DO UPDATE SET retirado_en=NULL`, sid, id); err != nil {
				return domain.MemberProfile{}, err
			}
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return domain.MemberProfile{}, err
	}
	return r.Find(ctx, id)
}

func (r *UserRepository) SetStatus(ctx context.Context, id, status string) (domain.MemberProfile, error) {
	result, err := r.db.Exec(ctx, `UPDATE usuarios SET estado=$1 WHERE id=$2`, status, id)
	if err != nil {
		return domain.MemberProfile{}, err
	}
	if result.RowsAffected() == 0 {
		return domain.MemberProfile{}, domain.ErrMemberNotFound
	}
	return r.Find(ctx, id)
}
func (r *UserRepository) Publications(ctx context.Context, id string) ([]domain.Publication, error) {
	rows, err := r.db.Query(ctx, `SELECT id,titulo,COALESCE(descripcion,''),COALESCE(url,''),publicado_en FROM publicaciones WHERE usuario_id=$1 AND visible=true ORDER BY publicado_en DESC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Publication{}
	for rows.Next() {
		var p domain.Publication
		if err := rows.Scan(&p.ID, &p.Title, &p.Description, &p.URL, &p.PublishedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *UserRepository) Create(ctx context.Context, userID, tokenHash string) error {
	_, err := r.db.Exec(ctx, `INSERT INTO sesiones (usuario_id, token_hash, expira_en) VALUES ($1, $2, now() + interval '24 hours')`, userID, tokenHash)
	return err
}

func (r *UserRepository) FindUserID(ctx context.Context, tokenHash string) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `SELECT usuario_id FROM sesiones WHERE token_hash = $1 AND revocada_en IS NULL AND expira_en > now()`, tokenHash).Scan(&id)
	return id, err
}

func (r *UserRepository) Revoke(ctx context.Context, tokenHash string) error {
	_, err := r.db.Exec(ctx, `UPDATE sesiones SET revocada_en = now() WHERE token_hash = $1 AND revocada_en IS NULL`, tokenHash)
	return err
}

func (r *UserRepository) RevokeAllForUser(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `UPDATE sesiones SET revocada_en = now() WHERE usuario_id = $1 AND revocada_en IS NULL`, userID)
	return err
}

// DeleteExpiredSessions removes rows that FindUserID would already reject.
func (r *UserRepository) DeleteExpiredSessions(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `DELETE FROM sesiones WHERE expira_en <= now() OR revocada_en IS NOT NULL`)
	return err
}

func (r *UserRepository) CreateRecovery(ctx context.Context, userID, tokenHash string) error {
	_, err := r.db.Exec(ctx, `INSERT INTO recuperaciones_password (usuario_id, token_hash, expira_en) VALUES ($1, $2, now() + interval '30 minutes')`, userID, tokenHash)
	return err
}

func (r *UserRepository) Consume(ctx context.Context, tokenHash string) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `UPDATE recuperaciones_password SET usado_en = now() WHERE token_hash = $1 AND usado_en IS NULL AND expira_en > now() RETURNING usuario_id`, tokenHash).Scan(&id)
	return id, err
}

// DeleteExpiredRecoveries removes recovery tokens that are expired or already used.
func (r *UserRepository) DeleteExpiredRecoveries(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `DELETE FROM recuperaciones_password WHERE expira_en <= now() OR usado_en IS NOT NULL`)
	return err
}

func (r *UserRepository) Ping(ctx context.Context) error {
	return r.db.Ping(ctx)
}
