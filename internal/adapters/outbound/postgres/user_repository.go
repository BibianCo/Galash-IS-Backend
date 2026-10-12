package postgres

import (
	"context"
	"strings"

	"github.com/galash-uptc/backend/internal/domain"
	"github.com/jackc/pgx/v5"
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
		INSERT INTO usuarios (firebase_uid, nombre, apellido, tipo_dni, dni, email, rol)
		VALUES (
			$1,
			$2,
			NULLIF($3, ''),
			NULLIF($4, ''),
			NULLIF($5, ''),
			$6,
			COALESCE(NULLIF($7, ''), 'estudiante')
		)
		ON CONFLICT (firebase_uid) DO UPDATE SET
			nombre = COALESCE(NULLIF(EXCLUDED.nombre, ''), usuarios.nombre),
			apellido = COALESCE(NULLIF(EXCLUDED.apellido, ''), usuarios.apellido),
			tipo_dni = COALESCE(NULLIF(EXCLUDED.tipo_dni, ''), usuarios.tipo_dni),
			dni = COALESCE(NULLIF(EXCLUDED.dni, ''), usuarios.dni),
			email = EXCLUDED.email,
			rol = COALESCE(NULLIF(EXCLUDED.rol, ''), usuarios.rol)
		RETURNING id`,
		user.FirebaseUID, user.Name, user.LastName, user.DocumentType, user.DocumentNumber, user.Email, user.Role,
	).Scan(&id)
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
	err := r.db.QueryRow(ctx, `SELECT id, firebase_uid, nombre, COALESCE(apellido, ''), email, rol, estado FROM usuarios `+where, arg).
		Scan(&user.ID, &user.FirebaseUID, &user.Name, &user.LastName, &user.Email, &user.Role, &user.Status)
	return user, err
}

func (r *UserRepository) SetPassword(ctx context.Context, id, passwordHash string) error {
	_, err := r.db.Exec(ctx, `UPDATE usuarios SET password_hash = $1 WHERE id = $2`, passwordHash, id)
	return err
}

func (r *UserRepository) Search(ctx context.Context, query string, limit int) ([]domain.User, error) {
	pattern := "%" + strings.ToLower(strings.TrimSpace(query)) + "%"
	rows, err := r.db.Query(ctx, `SELECT id, firebase_uid, nombre, COALESCE(apellido, ''), email, COALESCE(tipo_dni, ''), COALESCE(dni, ''), rol, estado
		FROM usuarios WHERE estado = 'activo' AND (lower(nombre || ' ' || COALESCE(apellido, '')) LIKE $1 OR lower(email) LIKE $1)
		ORDER BY nombre, apellido, email LIMIT $2`, pattern, limit)
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

func (r *UserRepository) CreateRecovery(ctx context.Context, userID, tokenHash string) error {
	_, err := r.db.Exec(ctx, `INSERT INTO recuperaciones_password (usuario_id, token_hash, expira_en) VALUES ($1, $2, now() + interval '30 minutes')`, userID, tokenHash)
	return err
}

func (r *UserRepository) Consume(ctx context.Context, tokenHash string) (string, error) {
	var id string
	err := r.db.QueryRow(ctx, `UPDATE recuperaciones_password SET usado_en = now() WHERE token_hash = $1 AND usado_en IS NULL AND expira_en > now() RETURNING usuario_id`, tokenHash).Scan(&id)
	return id, err
}

func (r *UserRepository) Ping(ctx context.Context) error {
	return r.db.Ping(ctx)
}
