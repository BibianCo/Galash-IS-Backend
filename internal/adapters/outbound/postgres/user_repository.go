package postgres

import (
	"context"

	"github.com/galash-uptc/backend/internal/domain"
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
		INSERT INTO usuarios (firebase_uid, nombre, apellido, email)
		VALUES ($1, $2, NULLIF($3, ''), $4)
		ON CONFLICT (firebase_uid) DO UPDATE SET
			nombre = EXCLUDED.nombre,
			apellido = COALESCE(EXCLUDED.apellido, usuarios.apellido),
			email = EXCLUDED.email
		RETURNING id`, user.FirebaseUID, user.Name, user.LastName, user.Email).Scan(&id)
	return id, err
}

func (r *UserRepository) Ping(ctx context.Context) error {
	return r.db.Ping(ctx)
}
