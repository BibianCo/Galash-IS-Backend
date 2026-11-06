package files_test

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

// TDB-T01 / TDB-T09: persistence and download rules of archivos_adjuntos.
func TestRepositoryAccessRules(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	run := fmt.Sprintf("m13-%d", time.Now().UnixNano())
	var admin, teacher, student, other, inactive, published, draft string
	users := []string{}
	for i, u := range []struct {
		id           *string
		role, status string
	}{{&admin, "admin", "activo"}, {&teacher, "docente", "activo"}, {&student, "estudiante", "activo"}, {&other, "estudiante", "activo"}, {&inactive, "estudiante", "inactivo"}} {
		uid := fmt.Sprintf("%s-%d", run, i)
		if err := db.QueryRow(ctx, `INSERT INTO usuarios (firebase_uid,nombre,apellido,email,rol,estado) VALUES ($1,'Prueba',$2,$1||'@example.test',$2,$3) RETURNING id`, uid, u.role, u.status).Scan(u.id); err != nil {
			t.Fatalf("seed user: %v", err)
		}
		users = append(users, *u.id)
	}
	for _, c := range []struct {
		id     *string
		estado string
	}{{&published, "publicado"}, {&draft, "borrador"}} {
		if err := db.QueryRow(ctx, `INSERT INTO comunicados (titulo,descripcion,estado,autor_id) VALUES ('t','d',$1,$2) RETURNING id`, c.estado, teacher).Scan(c.id); err != nil {
			t.Fatalf("seed announcement: %v", err)
		}
	}
	t.Cleanup(func() {
		for _, sql := range []string{
			`DELETE FROM archivos_adjuntos WHERE subido_por = ANY($1::uuid[])`,
			`DELETE FROM comunicados WHERE autor_id = ANY($1::uuid[])`,
			`DELETE FROM usuarios WHERE id = ANY($1::uuid[])`,
		} {
			if _, err := db.Exec(ctx, sql, users); err != nil {
				t.Errorf("cleanup %q: %v", sql, err)
			}
		}
		db.Close()
	})

	repo := postgresadapter.NewFileRepository(db)
	create := func(entityType, entityID, uploader string) domain.FileAttachment {
		f, err := repo.Create(ctx, domain.FileAttachment{EntityType: entityType, EntityID: entityID, Name: "a.pdf", ContentType: "application/pdf", Size: 4, UploadedBy: uploader}, []byte("%PDF"))
		if err != nil || f.ID == "" || f.CreatedAt.IsZero() {
			t.Fatalf("create = %+v, %v", f, err)
		}
		return f
	}
	onPublished := create("announcement", published, teacher)
	onDraft := create("announcement", draft, teacher)

	cases := []struct {
		name, file, actor string
		allowed           bool
	}{
		{"uploader reads draft attachment", onDraft.ID, teacher, true},
		{"admin reads draft attachment", onDraft.ID, admin, true},
		{"other user reads draft attachment", onDraft.ID, other, false},
		{"other user reads published attachment", onPublished.ID, other, true},
		{"inactive user reads published attachment", onPublished.ID, inactive, false},
	}
	for _, c := range cases {
		f, content, err := repo.Find(ctx, c.file, c.actor, "")
		if c.allowed && (err != nil || string(content) != "%PDF" || f.UploadedBy != teacher) {
			t.Fatalf("%s: %+v %q %v", c.name, f, content, err)
		}
		if !c.allowed && !errors.Is(err, domain.ErrFileNotFound) {
			t.Fatalf("%s: err=%v", c.name, err)
		}
	}
	for _, id := range []string{"00000000-0000-0000-0000-000000000000", "no-es-uuid"} {
		if _, _, err := repo.Find(ctx, id, admin, "admin"); !errors.Is(err, domain.ErrFileNotFound) {
			t.Fatalf("find %s: err=%v, want ErrFileNotFound", id, err)
		}
	}
	if _, err := repo.Create(ctx, domain.FileAttachment{EntityType: "announcement", EntityID: published, Name: "a.pdf", ContentType: "application/pdf", Size: 0, UploadedBy: teacher}, []byte{}); err == nil {
		t.Fatal("size CHECK accepted 0 bytes")
	}
}

// D032 / PT-SEM-06 / TDB-T09: upload and download follow the associated
// entity's read rule; missing or inaccessible entities are rejected.
func TestRepositoryEntityAccessRules(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	run := fmt.Sprintf("d032-%d", time.Now().UnixNano())
	var admin, teacher, author, student, seedbed, activity, project, proposal, draftEvent, draftNews string
	users := []string{}
	for i, u := range []struct {
		id   *string
		role string
	}{{&admin, "admin"}, {&teacher, "docente"}, {&author, "estudiante"}, {&student, "estudiante"}} {
		uid := fmt.Sprintf("%s-%d", run, i)
		if err := db.QueryRow(ctx, `INSERT INTO usuarios (firebase_uid,nombre,apellido,email,rol,estado) VALUES ($1,'Prueba',$2,$1||'@example.test',$2,'activo') RETURNING id`, uid, u.role).Scan(u.id); err != nil {
			t.Fatalf("seed user: %v", err)
		}
		users = append(users, *u.id)
	}
	seeds := []struct {
		id   *string
		sql  string
		args []any
	}{
		{&seedbed, `INSERT INTO semilleros (nombre,lider_id) VALUES ($1,$2) RETURNING id`, []any{run, teacher}},
		{&activity, `INSERT INTO actividades (titulo,semillero_id) VALUES ('a',$1) RETURNING id`, nil},
		{&project, `INSERT INTO proyectos (titulo,semillero_id) VALUES ('p',$1) RETURNING id`, nil},
		{&proposal, `INSERT INTO propuestas_investigacion (titulo,problema,objetivos,metodologia,autor_id) VALUES ('t','p','o','m',$1) RETURNING id`, []any{author}},
		{&draftEvent, `INSERT INTO eventos (titulo,descripcion,estado,organizador_id) VALUES ('e','d','borrador',$1) RETURNING id`, []any{teacher}},
		{&draftNews, `INSERT INTO comunicados (titulo,descripcion,estado,autor_id) VALUES ('c','d','borrador',$1) RETURNING id`, []any{teacher}},
	}
	for _, s := range seeds {
		args := s.args
		if args == nil {
			args = []any{seedbed}
		}
		if err := db.QueryRow(ctx, s.sql, args...).Scan(s.id); err != nil {
			t.Fatalf("seed %q: %v", s.sql, err)
		}
	}
	t.Cleanup(func() {
		for _, c := range []struct {
			sql string
			arg any
		}{
			{`DELETE FROM archivos_adjuntos WHERE subido_por = ANY($1::uuid[])`, users},
			{`DELETE FROM comunicados WHERE id=$1`, draftNews},
			{`DELETE FROM eventos WHERE id=$1`, draftEvent},
			{`DELETE FROM propuestas_investigacion WHERE id=$1`, proposal},
			{`DELETE FROM proyectos WHERE id=$1`, project},
			{`DELETE FROM actividades WHERE id=$1`, activity},
			{`DELETE FROM semilleros WHERE id=$1`, seedbed},
			{`DELETE FROM usuarios WHERE id = ANY($1::uuid[])`, users},
		} {
			if _, err := db.Exec(ctx, c.sql, c.arg); err != nil {
				t.Errorf("cleanup %q: %v", c.sql, err)
			}
		}
		db.Close()
	})

	repo := postgresadapter.NewFileRepository(db)
	upload := func(entityType, entityID, actor string) (domain.FileAttachment, error) {
		return repo.Create(ctx, domain.FileAttachment{EntityType: entityType, EntityID: entityID, Name: "a.pdf", ContentType: "application/pdf", Size: 4, UploadedBy: actor}, []byte("%PDF"))
	}
	missing := "00000000-0000-0000-0000-000000000000"
	for _, c := range []struct {
		name, entityType, entityID, actor string
	}{
		{"missing activity", "activity", missing, admin},
		{"missing project as admin", "project", missing, admin},
		{"proposal of another student", "proposal", proposal, student},
		{"draft event of another user", "event", draftEvent, student},
		{"draft announcement of another user", "announcement", draftNews, student},
		{"entity type mismatch", "project", activity, teacher},
	} {
		if _, err := upload(c.entityType, c.entityID, c.actor); !errors.Is(err, domain.ErrForbidden) {
			t.Fatalf("%s: err=%v, want ErrForbidden", c.name, err)
		}
	}
	files := map[string]domain.FileAttachment{}
	for _, c := range []struct {
		name, entityType, entityID, actor string
	}{
		{"activity", "activity", activity, student},
		{"evidence", "evidence", activity, teacher},
		{"project", "project", project, teacher},
		{"proposal", "proposal", proposal, author},
		{"event", "event", draftEvent, teacher},
		{"announcement", "announcement", draftNews, teacher},
	} {
		f, err := upload(c.entityType, c.entityID, c.actor)
		if err != nil {
			t.Fatalf("upload %s: %v", c.name, err)
		}
		files[c.name] = f
	}
	for _, c := range []struct {
		file, actor string
		allowed     bool
	}{
		{"proposal", author, true},
		{"proposal", teacher, true},
		{"proposal", admin, true},
		{"proposal", student, false},
		{"activity", teacher, true},
		{"evidence", student, true},
		{"project", student, true},
		{"event", student, false},
		{"announcement", student, false},
		{"event", admin, true},
	} {
		_, _, err := repo.Find(ctx, files[c.file].ID, c.actor, "")
		if c.allowed && err != nil || !c.allowed && !errors.Is(err, domain.ErrFileNotFound) {
			t.Fatalf("find %s as %s: err=%v allowed=%v", c.file, c.actor, err, c.allowed)
		}
	}
}
