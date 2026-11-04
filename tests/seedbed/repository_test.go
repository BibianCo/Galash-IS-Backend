package seedbed_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"testing"
	"time"

	postgresadapter "github.com/galash-uptc/backend/internal/adapters/outbound/postgres"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepositoryDelegatesAreActiveMembersOnly(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	defer db.Close()
	run := fmt.Sprintf("dlg-%d", time.Now().UnixNano())
	var director, member, outsider, seedbed string
	for _, u := range []*string{&director, &member, &outsider} {
		if err := db.QueryRow(ctx, `INSERT INTO usuarios (firebase_uid,nombre,email,rol,estado) VALUES ($1,'U',$1||'@example.test','docente','activo') RETURNING id`, fmt.Sprintf("%s-%p", run, u)).Scan(u); err != nil {
			t.Fatalf("seed user: %v", err)
		}
	}
	if err := db.QueryRow(ctx, `INSERT INTO semilleros (nombre,lider_id) VALUES ($1,$2) RETURNING id`, run, director).Scan(&seedbed); err != nil {
		t.Fatalf("seed seedbed: %v", err)
	}
	defer func() {
		_, _ = db.Exec(ctx, `DELETE FROM semillero_miembros WHERE semillero_id=$1`, seedbed)
		_, _ = db.Exec(ctx, `DELETE FROM semilleros WHERE id=$1`, seedbed)
		_, _ = db.Exec(ctx, `DELETE FROM usuarios WHERE id IN ($1,$2,$3)`, director, member, outsider)
	}()
	repo := postgresadapter.NewSeedbedRepository(db)
	if err := repo.AddMember(ctx, seedbed, member); err != nil {
		t.Fatalf("add member: %v", err)
	}
	if err := repo.SetDelegates(ctx, seedbed, []string{member, outsider}); !errors.Is(err, domain.ErrInvalidDelegates) {
		t.Fatalf("non-member delegate error=%v, want ErrInvalidDelegates", err)
	}
	if managed, _ := repo.ManagedSeedbeds(ctx, member); len(managed) != 0 {
		t.Fatalf("failed SetDelegates left partial state: %v", managed)
	}
	if err := repo.SetDelegates(ctx, seedbed, []string{member}); err != nil {
		t.Fatalf("set delegates: %v", err)
	}
	for _, uid := range []string{director, member} {
		if managed, err := repo.ManagedSeedbeds(ctx, uid); err != nil || !slices.Contains(managed, seedbed) {
			t.Fatalf("managed(%s)=%v err=%v", uid, managed, err)
		}
	}
	if members, err := repo.ListMembers(ctx, seedbed, false); err != nil || len(members) != 1 || !members[0].Delegate {
		t.Fatalf("members=%+v err=%v", members, err)
	}
	if err := repo.RemoveMember(ctx, seedbed, member); err != nil {
		t.Fatalf("remove member: %v", err)
	}
	if managed, err := repo.ManagedSeedbeds(ctx, member); err != nil || len(managed) != 0 {
		t.Fatalf("withdrawn member still manages: %v err=%v", managed, err)
	}
}
