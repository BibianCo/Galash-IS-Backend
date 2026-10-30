package registration_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	postgresadapter "github.com/galash-uptc/backend/internal/adapters/outbound/postgres"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestRepositoryReadsPersonalDataFromAccount(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	defer db.Close()
	run := fmt.Sprintf("reg-%d", time.Now().UnixNano())
	var complete, incomplete, other string
	insert := `INSERT INTO usuarios (firebase_uid,nombre,apellido,dni,codigo_estudiantil,fecha_nacimiento,telefono,email) VALUES ($1,'Ana','Pérez',NULLIF($2,''),NULLIF($3,''),NULLIF($4,'')::date,NULLIF($5,''),$1||'@example.test') RETURNING id`
	for _, u := range []struct {
		id                             *string
		suffix, dni, cod, birth, phone string
	}{{&complete, "a", run + "-dni", "", "2001-05-20", "300"}, {&incomplete, "b", run + "-dni2", "", "", ""}, {&other, "c", run + "-dni3", run + "-taken", "2000-01-01", "301"}} {
		if err := db.QueryRow(ctx, insert, run+u.suffix, u.dni, u.cod, u.birth, u.phone).Scan(u.id); err != nil {
			t.Fatalf("seed user: %v", err)
		}
	}
	defer func() {
		_, _ = db.Exec(ctx, `DELETE FROM seguimiento_solicitud WHERE solicitud_id IN (SELECT id FROM solicitudes_registro WHERE usuario_id IN ($1,$2,$3))`, complete, incomplete, other)
		_, _ = db.Exec(ctx, `DELETE FROM solicitudes_registro WHERE usuario_id IN ($1,$2,$3)`, complete, incomplete, other)
		_, _ = db.Exec(ctx, `DELETE FROM usuarios WHERE id IN ($1,$2,$3)`, complete, incomplete, other)
	}()

	repo := postgresadapter.NewRegistrationRepository(db)
	request := validRequest()
	request.UserID = incomplete
	if _, err := repo.CreateRegistration(ctx, request, run+"-h1"); !errors.Is(err, domain.ErrIncompleteProfile) {
		t.Fatalf("incomplete profile error=%v", err)
	}

	request.UserID, request.StudentCode = complete, run+"-taken"
	if _, err := repo.CreateRegistration(ctx, request, run+"-h2"); !errors.Is(err, domain.ErrDuplicateIdentity) {
		t.Fatalf("duplicate student code error=%v", err)
	}

	request.StudentCode = run + "-cod"
	created, err := repo.CreateRegistration(ctx, request, run+"-h3")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.Name != "Ana Pérez" || created.DocumentNumber != run+"-dni" || created.StudentCode != run+"-cod" || created.Phone != "300" || created.BirthDate != "2001-05-20" {
		t.Fatalf("created=%+v", created)
	}
	var copied bool
	if err := db.QueryRow(ctx, `SELECT nombre IS NOT NULL OR cc IS NOT NULL OR cod IS NOT NULL OR telefono IS NOT NULL OR fecha_nacimiento IS NOT NULL FROM solicitudes_registro WHERE id=$1`, created.ID).Scan(&copied); err != nil || copied {
		t.Fatalf("personal data copied into request: copied=%v err=%v", copied, err)
	}

	status, err := repo.GetRegistrationStatus(ctx, created.ID, "", complete)
	if err != nil || status.DocumentNumber != run+"-dni" {
		t.Fatalf("status=%+v err=%v", status, err)
	}
	if _, err := repo.UpdateRegistrationStatus(ctx, created.ID, domain.RegistrationReview, "ok", ""); err != nil {
		t.Fatalf("review: %v", err)
	}
	approved, err := repo.UpdateRegistrationStatus(ctx, created.ID, domain.RegistrationApproved, "ok", "")
	if err != nil || approved.Phone != "300" {
		t.Fatalf("approved=%+v err=%v", approved, err)
	}
	var role string
	if err := db.QueryRow(ctx, `SELECT rol FROM usuarios WHERE id=$1`, complete).Scan(&role); err != nil || role != "estudiante" {
		t.Fatalf("role=%q err=%v", role, err)
	}
	list, err := repo.ListRegistrations(ctx, domain.RegistrationApproved, "", "", "", "")
	if err != nil || len(list) == 0 {
		t.Fatalf("list=%d err=%v", len(list), err)
	}
	if detail, err := repo.GetRegistrationDetail(ctx, created.ID); err != nil || detail.Name != "Ana Pérez" || len(detail.History) != 3 {
		t.Fatalf("detail=%+v err=%v", detail, err)
	}
}
