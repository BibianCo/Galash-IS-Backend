package content_test

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	httpadapter "github.com/galash-uptc/backend/internal/adapters/inbound/http"
	postgresadapter "github.com/galash-uptc/backend/internal/adapters/outbound/postgres"
	authapp "github.com/galash-uptc/backend/internal/application/auth"
	contentapp "github.com/galash-uptc/backend/internal/application/content"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

func seed(t *testing.T) (*postgresadapter.ContentPageRepository, string, string) {
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
	run := fmt.Sprintf("content-%d", time.Now().UnixNano())
	var user string
	if err := db.QueryRow(ctx, `INSERT INTO usuarios (firebase_uid,nombre,apellido,email,rol) VALUES ($1,'Prueba','Admin',$1||'@example.test','admin') RETURNING id`, run).Scan(&user); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() {
		for _, sql := range []string{`DELETE FROM contenido_paginas WHERE autor_id=$1`, `DELETE FROM usuarios WHERE id=$1`} {
			if _, err := db.Exec(ctx, sql, user); err != nil {
				t.Errorf("cleanup %q: %v", sql, err)
			}
		}
		db.Close()
	})
	return postgresadapter.NewContentPageRepository(db), user, run
}

// PT-INFO-02: PostgreSQL persists draft, author name, update and publication dates.
func TestRepositoryDraftAndPublish(t *testing.T) {
	repo, user, run := seed(t)
	ctx := context.Background()
	p, err := repo.Create(ctx, domain.ContentPage{Slug: run + "-mision", Title: "Misión", Content: "<p>x</p>", Status: "borrador", AuthorID: user})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if p.ID == "" || p.AuthorName != "Prueba Admin" || p.Status != "borrador" || p.PublishedAt != nil || p.UpdatedAt.IsZero() {
		t.Errorf("created = %+v", p)
	}
	if _, err := repo.FindBySlug(ctx, p.Slug); !errors.Is(err, domain.ErrContentNotFound) {
		t.Errorf("draft by slug err = %v, want not found", err)
	}
	time.Sleep(10 * time.Millisecond)
	u, err := repo.Update(ctx, p.ID, "Misión GALASH", "<p>y</p>", p.Slug)
	if err != nil || u.Title != "Misión GALASH" || u.AuthorName != "Prueba Admin" || !u.UpdatedAt.After(p.UpdatedAt) {
		t.Errorf("update = %+v, %v", u, err)
	}
	pub, err := repo.Publish(ctx, p.ID)
	if err != nil || pub.Status != "publicado" || pub.PublishedAt == nil {
		t.Fatalf("publish = %+v, %v", pub, err)
	}
	if got, err := repo.FindBySlug(ctx, p.Slug); err != nil || got.ID != p.ID || got.Content != "<p>y</p>" {
		t.Errorf("published by slug = %+v, %v", got, err)
	}
	public, err := repo.List(ctx, false)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	found := false
	for _, x := range public {
		found = found || x.ID == p.ID
		if x.Status != "publicado" {
			t.Errorf("public list contains %s with status %q", x.Slug, x.Status)
		}
	}
	if !found {
		t.Errorf("published page missing from public list")
	}
	if _, err := repo.Find(ctx, "00000000-0000-0000-0000-000000000000"); !errors.Is(err, domain.ErrContentNotFound) {
		t.Errorf("unknown uuid err = %v, want not found", err)
	}
}

// PT-INFO-02: real adapter errors must map to client errors, not HTTP 500.
func TestRepositoryErrorsThroughHTTP(t *testing.T) {
	repo, user, run := seed(t)
	u := users{user: {ID: user, Role: "admin", Status: "activo"}}
	auth := authapp.NewService(verifier{}, &authStore{users: u, sessions: map[string]string{}})
	s := server{handler: httpadapter.NewHandlerWithContent(auth, healthChecker{}, nil, nil, nil, nil, nil, contentapp.NewService(repo, u)).Routes(), auth: auth}

	body := page(run+"-vision", "<p>Visión</p>")
	expect(t, "create", s.do(t, http.MethodPost, "/admin/content/pages", user, body), http.StatusCreated)
	if r := s.do(t, http.MethodPost, "/admin/content/pages", user, body); r.Code != http.StatusBadRequest && r.Code != http.StatusConflict {
		t.Errorf("duplicate slug: status=%d, want 400 or 409: %s", r.Code, strings.TrimSpace(r.Body.String()))
	}
	expect(t, "non-uuid id edit", s.do(t, http.MethodPatch, "/admin/content/pages/999", user, `{"title":"x"}`), http.StatusNotFound)
	expect(t, "non-uuid id publish", s.do(t, http.MethodPost, "/admin/content/pages/999/publish", user, ""), http.StatusNotFound)
}
