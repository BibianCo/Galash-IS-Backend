package auth_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	postgresadapter "github.com/galash-uptc/backend/internal/adapters/outbound/postgres"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/jackc/pgx/v5"
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

func TestUpsertAssignsVisitanteAndNeverChangesExistingRole(t *testing.T) {
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

	uid := fmt.Sprintf("role-register-test-%d", time.Now().UnixNano())
	defer func() {
		_, _ = db.Exec(ctx, `DELETE FROM usuarios WHERE firebase_uid = $1`, uid)
	}()

	repository := postgresadapter.NewUserRepository(db)
	// Even if a caller passes a privileged role, self-registration stores visitante.
	if _, err := repository.Upsert(ctx, domain.User{FirebaseUID: uid, Name: "Ana", Email: uid + "@example.test", Role: "admin"}); err != nil {
		t.Fatalf("initial Upsert(): %v", err)
	}
	var role string
	if err := db.QueryRow(ctx, `SELECT rol FROM usuarios WHERE firebase_uid = $1`, uid).Scan(&role); err != nil {
		t.Fatalf("read role: %v", err)
	}
	if role != "visitante" {
		t.Fatalf("new role=%q, want visitante", role)
	}

	// An admin assigns a role later; re-registration must not overwrite it.
	if _, err := db.Exec(ctx, `UPDATE usuarios SET rol = 'profesor' WHERE firebase_uid = $1`, uid); err != nil {
		t.Fatalf("assign role: %v", err)
	}
	if _, err := repository.Upsert(ctx, domain.User{FirebaseUID: uid, Name: "Ana", Email: uid + "@example.test", Role: "estudiante"}); err != nil {
		t.Fatalf("re-register Upsert(): %v", err)
	}
	if err := db.QueryRow(ctx, `SELECT rol FROM usuarios WHERE firebase_uid = $1`, uid).Scan(&role); err != nil {
		t.Fatalf("read role after re-register: %v", err)
	}
	if role != "profesor" {
		t.Fatalf("re-registration changed role to %q, want profesor", role)
	}
}

func TestSearchReturnsOnlyActiveMatchingUsers(t *testing.T) {
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

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	activeUID := "member-search-active-" + suffix
	inactiveUID := "member-search-inactive-" + suffix
	defer func() {
		_, _ = db.Exec(ctx, `DELETE FROM usuarios WHERE firebase_uid IN ($1, $2)`, activeUID, inactiveUID)
	}()

	repository := postgresadapter.NewUserRepository(db)
	for _, user := range []domain.User{
		{FirebaseUID: activeUID, Name: "UniqueMember" + suffix, LastName: "Active", Email: "active-" + suffix + "@example.test"},
		{FirebaseUID: inactiveUID, Name: "UniqueMember" + suffix, LastName: "Inactive", Email: "inactive-" + suffix + "@example.test"},
	} {
		if _, err := repository.Upsert(ctx, user); err != nil {
			t.Fatalf("insert search fixture: %v", err)
		}
	}
	// Registration always creates visitante; the profesor role is assigned by an admin.
	if _, err := db.Exec(ctx, `UPDATE usuarios SET rol = 'profesor' WHERE firebase_uid = $1`, activeUID); err != nil {
		t.Fatalf("assign search fixture role: %v", err)
	}
	if _, err := db.Exec(ctx, `UPDATE usuarios SET estado = 'inactivo' WHERE firebase_uid = $1`, inactiveUID); err != nil {
		t.Fatalf("deactivate search fixture: %v", err)
	}

	users, err := repository.Search(ctx, strings.ToLower("uniquemember"+suffix), 10)
	if err != nil {
		t.Fatalf("Search() error=%v", err)
	}
	if len(users) != 1 || users[0].FirebaseUID != activeUID || users[0].Role != "profesor" {
		t.Fatalf("Search() users=%+v, want only active fixture %q", users, activeUID)
	}

	byEmail, err := repository.Search(ctx, "active-"+suffix+"@example.test", 10)
	if err != nil {
		t.Fatalf("Search() by email error=%v", err)
	}
	if len(byEmail) != 1 || byEmail[0].FirebaseUID != activeUID {
		t.Fatalf("Search() by email users=%+v, want active fixture %q", byEmail, activeUID)
	}
}

func TestUpdateRoleUpdatesActiveUserAndRejectsInactiveOrMissingUID(t *testing.T) {
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

	suffix := fmt.Sprintf("%d", time.Now().UnixNano())
	activeUID := "role-update-active-" + suffix
	inactiveUID := "role-update-inactive-" + suffix
	defer func() {
		_, _ = db.Exec(ctx, `DELETE FROM usuarios WHERE firebase_uid IN ($1, $2)`, activeUID, inactiveUID)
	}()

	repository := postgresadapter.NewUserRepository(db)
	for _, uid := range []string{activeUID, inactiveUID} {
		if _, err := repository.Upsert(ctx, domain.User{FirebaseUID: uid, Name: "Role target", Email: uid + "@example.test", Role: "estudiante"}); err != nil {
			t.Fatalf("insert role fixture: %v", err)
		}
	}
	if _, err := db.Exec(ctx, `UPDATE usuarios SET estado = 'inactivo' WHERE firebase_uid = $1`, inactiveUID); err != nil {
		t.Fatalf("deactivate role fixture: %v", err)
	}

	if err := repository.UpdateRole(ctx, activeUID, "profesor"); err != nil {
		t.Fatalf("UpdateRole() active user error=%v", err)
	}
	var role string
	if err := db.QueryRow(ctx, `SELECT rol FROM usuarios WHERE firebase_uid = $1`, activeUID).Scan(&role); err != nil {
		t.Fatalf("read updated role: %v", err)
	}
	if role != "profesor" {
		t.Fatalf("stored role=%q, want profesor", role)
	}

	for _, uid := range []string{inactiveUID, "missing-" + suffix} {
		if err := repository.UpdateRole(ctx, uid, "admin"); !errors.Is(err, pgx.ErrNoRows) {
			t.Errorf("UpdateRole(%q) error=%v, want pgx.ErrNoRows", uid, err)
		}
	}
}
