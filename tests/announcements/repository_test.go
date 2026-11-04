package announcements_test

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

type dbFixture struct {
	repo                     *postgresadapter.AnnouncementRepository
	db                       *pgxpool.Pool
	admin, teacher, teacher2 string
}

func seedDB(t *testing.T) dbFixture {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	run := fmt.Sprintf("m8-%d", time.Now().UnixNano())
	ids := []string{}
	for _, role := range []string{"admin", "docente", "docente"} {
		var id string
		uid := fmt.Sprintf("%s-%d", run, len(ids))
		if err := db.QueryRow(ctx, `INSERT INTO usuarios (firebase_uid,nombre,apellido,email,rol,estado) VALUES ($1,'Prueba',$2,$1||'@example.test',$2,'activo') RETURNING id`, uid, role).Scan(&id); err != nil {
			t.Fatalf("seed user: %v", err)
		}
		ids = append(ids, id)
	}
	t.Cleanup(func() {
		for _, id := range ids {
			for _, sql := range []string{`DELETE FROM comunicados WHERE autor_id=$1`, `DELETE FROM usuarios WHERE id=$1`} {
				if _, err := db.Exec(ctx, sql, id); err != nil {
					t.Errorf("cleanup %q: %v", sql, err)
				}
			}
		}
		db.Close()
	})
	return dbFixture{postgresadapter.NewAnnouncementRepository(db), db, ids[0], ids[1], ids[2]}
}

var attachments = []domain.AnnouncementAttachment{{Name: "guia.pdf", URL: "https://files.example/guia.pdf"}}

// TDB-T01 / TDB-T08: persistence, author name, draft visibility and publication dates.
func TestRepositoryDraftVisibilityAndPublish(t *testing.T) {
	f := seedDB(t)
	ctx := context.Background()
	a, err := f.repo.Create(ctx, domain.Announcement{Title: "Directriz", Description: "Texto", Attachments: attachments, AuthorID: f.teacher})
	if err != nil || a.ID == "" || a.Status != "borrador" || a.PublishedAt != nil || len(a.Attachments) != 1 {
		t.Fatalf("create = %+v, %v", a, err)
	}
	if got, err := f.repo.Find(ctx, a.ID, f.teacher, false); err != nil || got.AuthorName != "Prueba docente" {
		t.Errorf("author find = %+v, %v", got, err)
	}
	if _, err := f.repo.Find(ctx, a.ID, f.teacher2, false); !errors.Is(err, domain.ErrAnnouncementNotFound) {
		t.Errorf("other teacher find draft err = %v, want not found", err)
	}
	if _, err := f.repo.Find(ctx, "not-a-uuid", f.teacher, false); !errors.Is(err, domain.ErrAnnouncementNotFound) {
		t.Errorf("invalid uuid err = %v, want not found", err)
	}
	if list, _ := f.repo.List(ctx, f.teacher2, false); contains(list, a.ID) {
		t.Errorf("other teacher lists foreign draft")
	}
	if list, _ := f.repo.List(ctx, f.admin, true); !contains(list, a.ID) {
		t.Errorf("admin list misses draft")
	}
	time.Sleep(10 * time.Millisecond)
	u, err := f.repo.Update(ctx, a.ID, "Directriz v2", "Texto 2", nil)
	if err != nil || u.Title != "Directriz v2" || !u.UpdatedAt.After(a.UpdatedAt) {
		t.Errorf("update = %+v, %v", u, err)
	}
	p, err := f.repo.Publish(ctx, a.ID)
	if err != nil || p.Status != "publicado" || p.PublishedAt == nil {
		t.Fatalf("publish = %+v, %v", p, err)
	}
	if _, err := f.repo.Publish(ctx, a.ID); !errors.Is(err, domain.ErrAnnouncementNotFound) {
		t.Errorf("republish err = %v, want not found (no borrador row)", err)
	}
	if _, err := f.repo.Update(ctx, a.ID, "x", "x", nil); !errors.Is(err, domain.ErrAnnouncementNotFound) {
		t.Errorf("update published err = %v, want not found", err)
	}
	if list, _ := f.repo.List(ctx, f.teacher2, false); !contains(list, a.ID) {
		t.Errorf("published not listed for other teacher")
	}
}

// TDB-T08: an Admin can open the drafts that List shows to them.
func TestRepositoryAdminFindsListedDraft(t *testing.T) {
	f := seedDB(t)
	ctx := context.Background()
	a, err := f.repo.Create(ctx, domain.Announcement{Title: "Borrador", Description: "Texto", Attachments: attachments, AuthorID: f.teacher})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := f.repo.Find(ctx, a.ID, f.admin, true); err != nil {
		t.Errorf("admin find listed draft err = %v, want visible", err)
	}
}

// TDB-T02 / TDB-T03 / TDB-T12: idempotent read confirmation keeps first timestamp; status per active docente.
func TestRepositoryReadConfirmation(t *testing.T) {
	f := seedDB(t)
	ctx := context.Background()
	a, _ := f.repo.Create(ctx, domain.Announcement{Title: "Directriz", Description: "Texto", Attachments: attachments, AuthorID: f.teacher})

	if err := f.repo.ConfirmRead(ctx, a.ID, f.teacher2); err != nil {
		t.Fatalf("confirm draft: %v", err)
	}
	var n int
	_ = f.db.QueryRow(ctx, `SELECT count(*) FROM comunicados_lecturas WHERE comunicado_id=$1`, a.ID).Scan(&n)
	if n != 0 {
		t.Errorf("draft confirmation stored %d rows, want 0", n)
	}

	if _, err := f.repo.Publish(ctx, a.ID); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if err := f.repo.ConfirmRead(ctx, a.ID, f.teacher2); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	var first time.Time
	_ = f.db.QueryRow(ctx, `SELECT confirmado_en FROM comunicados_lecturas WHERE comunicado_id=$1 AND usuario_id=$2`, a.ID, f.teacher2).Scan(&first)
	time.Sleep(10 * time.Millisecond)
	if err := f.repo.ConfirmRead(ctx, a.ID, f.teacher2); err != nil {
		t.Fatalf("confirm again: %v", err)
	}
	var again time.Time
	_ = f.db.QueryRow(ctx, `SELECT confirmado_en FROM comunicados_lecturas WHERE comunicado_id=$1 AND usuario_id=$2`, a.ID, f.teacher2).Scan(&again)
	if !again.Equal(first) {
		t.Errorf("second confirmation changed timestamp %v -> %v", first, again)
	}
	if err := f.repo.ConfirmRead(ctx, "not-a-uuid", f.teacher2); !errors.Is(err, domain.ErrAnnouncementNotFound) {
		t.Errorf("confirm invalid uuid err = %v, want not found", err)
	}
	if got, _ := f.repo.Find(ctx, a.ID, f.teacher2, false); !got.ReadConfirmed {
		t.Errorf("readConfirmed=false for confirming teacher")
	}

	statuses, err := f.repo.ReadStatus(ctx, a.ID)
	if err != nil {
		t.Fatalf("read status: %v", err)
	}
	byID := map[string]domain.AnnouncementReadStatus{}
	for _, s := range statuses {
		byID[s.UserID] = s
	}
	if byID[f.teacher2].Status != "Leído" || byID[f.teacher2].ConfirmedAt == nil || byID[f.teacher].Status != "Pendiente" {
		t.Errorf("status = %+v", statuses)
	}
	if _, ok := byID[f.admin]; ok {
		t.Errorf("admin listed as receptor")
	}
}

// TDB-T02: FK rejects unknown author; cascade removes confirmations with the announcement.
func TestRepositoryReferentialIntegrity(t *testing.T) {
	f := seedDB(t)
	ctx := context.Background()
	if _, err := f.repo.Create(ctx, domain.Announcement{Title: "x", Description: "x", AuthorID: "00000000-0000-0000-0000-000000000000"}); err == nil {
		t.Errorf("create with unknown author succeeded")
	}
	a, _ := f.repo.Create(ctx, domain.Announcement{Title: "x", Description: "x", AuthorID: f.teacher})
	_, _ = f.repo.Publish(ctx, a.ID)
	_ = f.repo.ConfirmRead(ctx, a.ID, f.teacher2)
	if _, err := f.db.Exec(ctx, `DELETE FROM comunicados WHERE id=$1`, a.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	var n int
	_ = f.db.QueryRow(ctx, `SELECT count(*) FROM comunicados_lecturas WHERE comunicado_id=$1`, a.ID).Scan(&n)
	if n != 0 {
		t.Errorf("orphan confirmations = %d", n)
	}
}

func contains(list []domain.Announcement, id string) bool {
	for _, a := range list {
		if a.ID == id {
			return true
		}
	}
	return false
}
