package seedbed_test

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// live runs against the running API (GALASH_API_URL) and its database (TEST_DATABASE_URL).
type live struct {
	api, run string
	db       *pgxpool.Pool
	tokens   map[string]string // role -> session token
	ids      map[string]string // role -> user id
}

func newLive(t *testing.T) *live {
	t.Helper()
	api, dsn := os.Getenv("GALASH_API_URL"), os.Getenv("TEST_DATABASE_URL")
	if api == "" || dsn == "" {
		t.Skip("GALASH_API_URL or TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	l := &live{api: strings.TrimRight(api, "/"), run: fmt.Sprintf("live-%d", time.Now().UnixNano()), db: db, tokens: map[string]string{}, ids: map[string]string{}}
	t.Cleanup(l.cleanup)
	for _, role := range []string{"admin", "profesor", "estudiante"} {
		uid := l.run + "-" + role
		var id string
		if err := db.QueryRow(ctx, `INSERT INTO usuarios (firebase_uid,nombre,apellido,email,rol) VALUES ($1,'Prueba',$2,$3,$2) RETURNING id`, uid, role, uid+"@example.test").Scan(&id); err != nil {
			t.Fatalf("seed %s: %v", role, err)
		}
		raw := make([]byte, 32)
		_, _ = rand.Read(raw)
		token := base64.RawURLEncoding.EncodeToString(raw)
		sum := sha256.Sum256([]byte(token))
		if _, err := db.Exec(ctx, `INSERT INTO sesiones (usuario_id,token_hash,expira_en) VALUES ($1,$2,now()+interval '1 hour')`, id, base64.RawURLEncoding.EncodeToString(sum[:])); err != nil {
			t.Fatalf("seed session %s: %v", role, err)
		}
		l.ids[role], l.tokens[role] = id, token
	}
	return l
}

func (l *live) cleanup() {
	ctx := context.Background()
	like := l.run + "%"
	_, _ = l.db.Exec(ctx, `DELETE FROM semillero_miembros WHERE semillero_id IN (SELECT id FROM semilleros WHERE nombre LIKE $1)`, like)
	_, _ = l.db.Exec(ctx, `DELETE FROM semilleros WHERE nombre LIKE $1`, like)
	_, _ = l.db.Exec(ctx, `DELETE FROM sesiones WHERE usuario_id IN (SELECT id FROM usuarios WHERE firebase_uid LIKE $1)`, like)
	_, _ = l.db.Exec(ctx, `DELETE FROM usuarios WHERE firebase_uid LIKE $1`, like)
	l.db.Close()
}

// call sends a request as role; an empty role sends it without a session.
func (l *live) call(t *testing.T, method, path, role, body string) (int, string) {
	t.Helper()
	request, _ := http.NewRequest(method, l.api+path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if role != "" {
		request.Header.Set("Authorization", "Bearer "+l.tokens[role])
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer response.Body.Close()
	data, _ := io.ReadAll(response.Body)
	return response.StatusCode, string(data)
}

func (l *live) check(t *testing.T, name string, want int, method, path, role, body string) string {
	t.Helper()
	status, text := l.call(t, method, path, role, body)
	if status != want {
		t.Errorf("%s: %s %s status=%d, want %d: %s", name, method, path, status, want, text)
	}
	return text
}

func (l *live) createSeedbed(t *testing.T, suffix string) string {
	t.Helper()
	body := fmt.Sprintf(`{"name":"%s-%s","description":"d","researchLine":"IA","objective":"o","responsibleTeacherId":"%s","contactEmail":"semillero@example.test"}`, l.run, suffix, l.ids["profesor"])
	text := l.check(t, "create "+suffix, http.StatusCreated, http.MethodPost, "/seedbeds", "admin", body)
	var s struct{ ID, Status string }
	if json.Unmarshal([]byte(text), &s) != nil || s.ID == "" {
		t.Fatalf("create %s returned no id: %s", suffix, text)
	}
	return s.ID
}

func TestLiveCreateRejectsDuplicateName(t *testing.T) {
	l := newLive(t)
	l.createSeedbed(t, "dup")
	body := fmt.Sprintf(`{"name":"%s-dup","description":"d","researchLine":"IA","objective":"o","responsibleTeacherId":"%s","contactEmail":"semillero@example.test"}`, l.run, l.ids["profesor"])
	l.check(t, "duplicate name", http.StatusConflict, http.MethodPost, "/seedbeds", "admin", body)
}

func TestLiveUnknownSeedbedReturnsNotFound(t *testing.T) {
	l := newLive(t)
	missing := "00000000-0000-0000-0000-000000000000"
	for _, path := range []string{"/seedbeds/" + missing, "/seedbeds/" + missing + "/space", "/seedbeds/" + missing + "/members"} {
		l.check(t, "unknown seedbed", http.StatusNotFound, http.MethodGet, path, "", "")
	}
}

func TestLiveVisibilityControlsPublicAccess(t *testing.T) {
	l := newLive(t)
	id := l.createSeedbed(t, "visibility")
	l.check(t, "draft anonymous", http.StatusUnauthorized, http.MethodGet, "/seedbeds/"+id, "", "")
	l.check(t, "publish by responsible", http.StatusOK, http.MethodPatch, "/seedbeds/"+id+"/visibility", "profesor", `{"status":"Publicado"}`)
	l.check(t, "published anonymous", http.StatusOK, http.MethodGet, "/seedbeds/"+id, "", "")
	if space := l.check(t, "published space", http.StatusOK, http.MethodGet, "/seedbeds/"+id+"/space", "", ""); strings.Contains(space, l.run+"-profesor@example.test") {
		t.Errorf("public space exposes responsible email: %s", space)
	}
	l.check(t, "hide by student", http.StatusForbidden, http.MethodPatch, "/seedbeds/"+id+"/visibility", "estudiante", `{"status":"Oculto"}`)
	l.check(t, "hide by admin", http.StatusOK, http.MethodPatch, "/seedbeds/"+id+"/visibility", "admin", `{"status":"Oculto"}`)
	l.check(t, "hidden student", http.StatusNotFound, http.MethodGet, "/seedbeds/"+id, "estudiante", "")
	l.check(t, "inactivate by responsible", http.StatusOK, http.MethodPatch, "/seedbeds/"+id+"/visibility", "profesor", `{"status":"Inactivo"}`)
	if list := l.check(t, "public list", http.StatusOK, http.MethodGet, "/seedbeds?q="+l.run, "", ""); strings.Contains(list, id) {
		t.Errorf("public list includes inactive seedbed: %s", list)
	}
	if list := l.check(t, "admin list", http.StatusOK, http.MethodGet, "/seedbeds?includeInactive=true&q="+l.run, "admin", ""); !strings.Contains(list, id) {
		t.Errorf("admin list omits inactive seedbed: %s", list)
	}
}

func TestLiveHiddenMembersAnonymousSingleError(t *testing.T) {
	l := newLive(t)
	id := l.createSeedbed(t, "hidden-members")
	text := l.check(t, "hidden members anonymous", http.StatusUnauthorized, http.MethodGet, "/seedbeds/"+id+"/members", "", "")
	if strings.Count(text, `"error"`) != 1 {
		t.Errorf("hidden members anonymous wrote several error bodies: %s", text)
	}
}

func TestLiveWithdrawKeepsHistory(t *testing.T) {
	l := newLive(t)
	id := l.createSeedbed(t, "members")
	l.check(t, "publish", http.StatusOK, http.MethodPatch, "/seedbeds/"+id+"/visibility", "admin", `{"status":"Publicado"}`)
	student := l.ids["estudiante"]
	l.check(t, "link student", http.StatusNoContent, http.MethodPost, "/seedbeds/"+id+"/members", "profesor", `{"userId":"`+student+`"}`)
	if members := l.check(t, "current members", http.StatusOK, http.MethodGet, "/seedbeds/"+id+"/members", "", ""); !strings.Contains(members, student) {
		t.Errorf("current members omit linked student: %s", members)
	}
	l.check(t, "withdraw student", http.StatusNoContent, http.MethodDelete, "/seedbeds/"+id+"/members/"+student, "profesor", "")
	if members := l.check(t, "current after withdraw", http.StatusOK, http.MethodGet, "/seedbeds/"+id+"/members", "", ""); strings.Contains(members, student) {
		t.Errorf("withdrawn student still listed: %s", members)
	}
	if history := l.check(t, "history", http.StatusOK, http.MethodGet, "/seedbeds/"+id+"/members?history=true", "profesor", ""); !strings.Contains(history, student) || !strings.Contains(history, `"leftAt"`) {
		t.Errorf("history lost withdrawn student or leftAt: %s", history)
	}
	l.check(t, "withdraw non-member", http.StatusNotFound, http.MethodDelete, "/seedbeds/"+id+"/members/"+student, "profesor", "")
}
