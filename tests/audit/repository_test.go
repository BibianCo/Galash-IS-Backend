package audit_test

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	postgresadapter "github.com/galash-uptc/backend/internal/adapters/outbound/postgres"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
	"github.com/jackc/pgx/v5/pgxpool"
)

// TDB-T10: fields, filters and immutability of audit_logs.
// Audit rows are immutable by design, so the fixture actor is not deleted.
func TestRepositoryFiltersAndImmutability(t *testing.T) {
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
	run := fmt.Sprintf("m12-%d", time.Now().UnixNano())
	var actor string
	if err := db.QueryRow(ctx, `INSERT INTO usuarios (firebase_uid,nombre,apellido,email,rol,estado) VALUES ($1,'Prueba','Admin',$1||'@example.test','admin','activo') RETURNING id`, run).Scan(&actor); err != nil {
		t.Fatalf("seed actor: %v", err)
	}
	repo := postgresadapter.NewAuditRepository(db)
	entity := "/links-" + run
	before := time.Now().Add(-time.Minute)
	for _, item := range []domain.AuditLog{
		{ActorID: actor, Action: "POST " + entity, Entity: entity, Result: "success", Origin: "192.0.2.1:1"},
		{ActorID: actor, Action: "PATCH " + entity, Entity: entity, Result: "failure"},
	} {
		if err := repo.Append(ctx, item); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	after := time.Now().Add(time.Minute)

	all, err := repo.List(ctx, ports.AuditFilter{ActorID: actor, Entity: entity, From: &before, To: &after})
	if err != nil || len(all) != 2 {
		t.Fatalf("list = %+v, %v", all, err)
	}
	if all[0].ID == "" || all[0].Date.IsZero() || all[0].ActorID != actor {
		t.Fatalf("entry = %+v", all[0])
	}
	byAction, _ := repo.List(ctx, ports.AuditFilter{Action: "POST " + entity})
	if len(byAction) != 1 || byAction[0].Result != "success" || byAction[0].Origin != "192.0.2.1:1" {
		t.Fatalf("action filter = %+v", byAction)
	}
	past := before.Add(-time.Hour)
	if none, _ := repo.List(ctx, ports.AuditFilter{Entity: entity, To: &past}); len(none) != 0 {
		t.Fatalf("period filter = %+v", none)
	}
	if _, err := db.Exec(ctx, `UPDATE audit_logs SET result='success' WHERE entity=$1`, entity); err == nil {
		t.Fatal("audit_logs accepted UPDATE")
	}
	if _, err := db.Exec(ctx, `DELETE FROM audit_logs WHERE entity=$1`, entity); err == nil {
		t.Fatal("audit_logs accepted DELETE")
	}
	if err := repo.Append(ctx, domain.AuditLog{ActorID: "00000000-0000-0000-0000-000000000000", Action: "x", Entity: "x", Result: "success"}); err == nil {
		t.Fatal("unknown actor accepted by FK")
	}
}
