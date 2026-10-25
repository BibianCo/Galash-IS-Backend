package event_test

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

func seed(t *testing.T) (*postgresadapter.EventRepository, string, string) {
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
	run := fmt.Sprintf("event-%d", time.Now().UnixNano())
	var user string
	if err := db.QueryRow(ctx, `INSERT INTO usuarios (firebase_uid,nombre,email,rol) VALUES ($1,'Prueba',$1||'@example.test','docente') RETURNING id`, run).Scan(&user); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() {
		for _, sql := range []string{`DELETE FROM eventos WHERE organizador_id=$1`, `DELETE FROM usuarios WHERE id=$1`} {
			if _, err := db.Exec(ctx, sql, user); err != nil {
				t.Errorf("cleanup %q: %v", sql, err)
			}
		}
		db.Close()
	})
	return postgresadapter.NewEventRepository(db), user, run
}

func TestEventRepositoryPersistsLifecycleAndFilters(t *testing.T) {
	r, user, venue := seed(t)
	ctx := context.Background()
	start := time.Date(2030, 5, 10, 14, 0, 0, 0, time.UTC)
	end := start.Add(2 * time.Hour)
	e, err := r.Create(ctx, domain.Event{Title: "Taller", Type: venue, StartAt: start, EndAt: &end, OrganizerID: user, Venue: venue, ImageURL: "https://example.com/i.png", Status: "borrador"})
	if err != nil || e.ID == "" || e.Status != "borrador" || e.EndAt == nil || !e.EndAt.Equal(end) {
		t.Fatalf("create = %+v, %v", e, err)
	}
	found, err := r.Find(ctx, e.ID)
	if err != nil || found.OrganizerName == "" || found.ImageURL != e.ImageURL {
		t.Fatalf("find = %+v, %v", found, err)
	}
	if _, err := r.Find(ctx, "not-a-uuid"); !errors.Is(err, domain.ErrEventNotFound) {
		t.Errorf("find invalid id error = %v", err)
	}

	time.Sleep(10 * time.Millisecond)
	found.Title, found.EndAt = "Taller editado", nil
	updated, err := r.Update(ctx, e.ID, found)
	if err != nil || updated.Title != "Taller editado" || updated.EndAt != nil || !updated.UpdatedAt.After(e.UpdatedAt) {
		t.Fatalf("update = %+v, %v", updated, err)
	}
	if published, err := r.SetStatus(ctx, e.ID, "publicado"); err != nil || published.Status != "publicado" {
		t.Fatalf("publish = %+v, %v", published, err)
	}

	from := start.Add(-time.Hour)
	for status, want := range map[string]int{"publicado": 1, "borrador": 0, "": 1} {
		list, err := r.List(ctx, venue, status, &from, nil)
		if err != nil || len(list) != want {
			t.Errorf("list status %q = %d events, %v; want %d", status, len(list), err, want)
		}
	}
	if list, err := r.List(ctx, venue, "", nil, &from); err != nil || len(list) != 0 {
		t.Errorf("list to before start = %d, %v", len(list), err)
	}

	later := start.Add(30 * time.Minute)
	for name, c := range map[string]struct {
		event   domain.Event
		exclude string
		want    bool
	}{
		"same instant without end": {domain.Event{Venue: venue, StartAt: start}, "", true},
		"inside open-ended event":  {domain.Event{Venue: " " + venue + " ", StartAt: start, EndAt: &later}, "", true},
		"excluding itself":         {domain.Event{Venue: venue, StartAt: start}, e.ID, false},
		"other venue":              {domain.Event{Venue: venue + "-b", StartAt: start}, "", false},
		"empty venue":              {domain.Event{StartAt: start}, "", false},
		"after open-ended instant": {domain.Event{Venue: venue, StartAt: later}, "", false},
	} {
		got, err := r.HasConflict(ctx, c.event, c.exclude)
		if err != nil || got != c.want {
			t.Errorf("%s: conflict = %v, %v; want %v", name, got, err, c.want)
		}
	}
	if _, err := r.SetStatus(ctx, e.ID, "cancelado"); err != nil {
		t.Fatal(err)
	}
	if got, err := r.HasConflict(ctx, domain.Event{Venue: venue, StartAt: start}, ""); err != nil || got {
		t.Errorf("cancelled event conflict = %v, %v", got, err)
	}
}
