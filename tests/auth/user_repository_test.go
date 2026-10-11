package auth_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	postgresadapter "github.com/galash-uptc/backend/internal/adapters/outbound/postgres"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestUpsertPreservesExistingLastName(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	defer db.Close()

	uid := fmt.Sprintf("lastname-test-%d", time.Now().UnixNano())
	defer func() {
		_, _ = db.Exec(ctx, `DELETE FROM usuarios WHERE firebase_uid = $1`, uid)
	}()

	repository := postgresadapter.NewUserRepository(db)
	user := domain.User{
		FirebaseUID: uid,
		Name:        "Ana",
		LastName:    "Pérez",
		Email:       uid + "@example.test",
		Role:        "estudiante",
	}
	if _, err := repository.Upsert(ctx, user); err != nil {
		t.Fatalf("initial Upsert(): %v", err)
	}

	user.LastName = ""
	if _, err := repository.Upsert(ctx, user); err != nil {
		t.Fatalf("Upsert() with empty last name: %v", err)
	}

	var lastName string
	if err := db.QueryRow(ctx, `SELECT apellido FROM usuarios WHERE firebase_uid = $1`, uid).Scan(&lastName); err != nil {
		t.Fatalf("read stored last name: %v", err)
	}
	if lastName != "Pérez" {
		t.Fatalf("stored last name=%q, want %q", lastName, "Pérez")
	}
}
