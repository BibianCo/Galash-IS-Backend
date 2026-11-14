package member_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	postgresadapter "github.com/galash-uptc/backend/internal/adapters/outbound/postgres"
	"github.com/galash-uptc/backend/internal/ports"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestMemberProfilesIncludePhotoURL(t *testing.T) {
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
	uid := fmt.Sprintf("photo-member-%d", time.Now().UnixNano())
	defer func() { _, _ = db.Exec(ctx, `DELETE FROM usuarios WHERE firebase_uid = $1`, uid) }()
	var id string
	if err := db.QueryRow(ctx, `INSERT INTO usuarios (firebase_uid,nombre,email,rol) VALUES ($1,$1,$1||'@example.test','estudiante') RETURNING id`, uid).Scan(&id); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	repository := postgresadapter.NewUserRepository(db)
	const photo = "https://example.test/foto.jpg"
	if err := repository.SetPhotoURL(ctx, id, photo); err != nil {
		t.Fatalf("SetPhotoURL(): %v", err)
	}
	found, err := repository.Find(ctx, id)
	if err != nil || found.PhotoURL != photo {
		t.Fatalf("Find() photo=%q err=%v", found.PhotoURL, err)
	}
	listed, err := repository.List(ctx, ports.MemberFilter{Query: uid, Limit: 10})
	if err != nil || len(listed) != 1 || listed[0].PhotoURL != photo {
		t.Fatalf("List()=%+v err=%v", listed, err)
	}
}
