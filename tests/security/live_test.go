package security_test

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Runs against the running API (GALASH_API_URL) and its database (TEST_DATABASE_URL).
type live struct {
	api, run string
	db       *pgxpool.Pool
	ids      map[string]string
}

func newLive(t *testing.T) *live {
	t.Helper()
	api, dsn := os.Getenv("GALASH_API_URL"), os.Getenv("TEST_DATABASE_URL")
	if api == "" || dsn == "" {
		t.Skip("GALASH_API_URL or TEST_DATABASE_URL is not set")
	}
	db, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	l := &live{api: strings.TrimRight(api, "/"), run: fmt.Sprintf("sec-%d", time.Now().UnixNano()), db: db, ids: map[string]string{}}
	t.Cleanup(func() {
		ctx, like := context.Background(), l.run+"%"
		_, _ = db.Exec(ctx, `DELETE FROM semilleros WHERE lider_id IN (SELECT id FROM usuarios WHERE firebase_uid LIKE $1)`, like)
		_, _ = db.Exec(ctx, `DELETE FROM sesiones WHERE usuario_id IN (SELECT id FROM usuarios WHERE firebase_uid LIKE $1)`, like)
		_, _ = db.Exec(ctx, `DELETE FROM usuarios WHERE firebase_uid LIKE $1`, like)
		db.Close()
	})
	for _, role := range []string{"admin", "docente"} {
		var id string
		if err := db.QueryRow(context.Background(), `INSERT INTO usuarios (firebase_uid,nombre,apellido,email,rol) VALUES ($1,'Prueba',$2,$1||'@example.test',$2) RETURNING id`, l.run+"-"+role, role).Scan(&id); err != nil {
			t.Fatalf("seed %s: %v", role, err)
		}
		l.ids[role] = id
	}
	return l
}

// session stores a session for role expiring at the given SQL interval; revoked marks it revoked.
func (l *live) session(t *testing.T, role, expires string, revoked bool) string {
	t.Helper()
	raw := make([]byte, 32)
	_, _ = rand.Read(raw)
	token := base64.RawURLEncoding.EncodeToString(raw)
	sum := sha256.Sum256([]byte(token))
	revokedAt := "NULL"
	if revoked {
		revokedAt = "now()"
	}
	if _, err := l.db.Exec(context.Background(), `INSERT INTO sesiones (usuario_id,token_hash,expira_en,revocada_en) VALUES ($1,$2,now()+$3::interval,`+revokedAt+`)`, l.ids[role], base64.RawURLEncoding.EncodeToString(sum[:]), expires); err != nil {
		t.Fatalf("seed session: %v", err)
	}
	return token
}

func (l *live) call(t *testing.T, method, path, token, body string) (int, http.Header, string) {
	t.Helper()
	request, _ := http.NewRequest(method, l.api+path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer response.Body.Close()
	data, _ := io.ReadAll(response.Body)
	return response.StatusCode, response.Header, string(data)
}

// PT-NF-04: expired or revoked sessions are rejected; a valid one is accepted.
func TestLiveSessionExpiryAndRevocation(t *testing.T) {
	l := newLive(t)
	for name, c := range map[string]struct {
		expires string
		revoked bool
		want    int
	}{
		"valid":   {"1 hour", false, http.StatusOK},
		"expired": {"-1 second", false, http.StatusUnauthorized},
		"revoked": {"1 hour", true, http.StatusUnauthorized},
	} {
		if status, _, body := l.call(t, http.MethodGet, "/users/me", l.session(t, "docente", c.expires, c.revoked), ""); status != c.want {
			t.Errorf("%s session: status=%d want %d: %s", name, status, c.want, body)
		}
	}
}

// PT-NF-04: SQL and script payloads in text fields are stored literally and never executed or interpreted.
func TestLiveInjectionPayloadsAreInert(t *testing.T) {
	l := newLive(t)
	admin := l.session(t, "admin", "1 hour", false)
	payloads := []string{
		`x'); DROP TABLE semilleros;--`,
		`<script>alert(1)</script>`,
		`<img src=x onerror=alert(1)>`,
	}
	for i, p := range payloads {
		name := fmt.Sprintf("%s-%d %s", l.run, i, p)
		body, _ := json.Marshal(map[string]string{"name": name, "description": p, "researchLine": p, "objective": p, "responsibleTeacherId": l.ids["docente"], "contactEmail": "semillero@example.test"})
		status, header, text := l.call(t, http.MethodPost, "/seedbeds", admin, string(body))
		if status != http.StatusCreated {
			t.Fatalf("create %q: %d %s", p, status, text)
		}
		if !strings.HasPrefix(header.Get("Content-Type"), "application/json") || strings.Contains(text, "<script") || strings.Contains(text, "<img") {
			t.Errorf("%q: response must be JSON with escaped markup: %s %s", p, header.Get("Content-Type"), text)
		}
		var got struct{ Name, Description string }
		if err := json.Unmarshal([]byte(text), &got); err != nil || got.Name != name || got.Description != p {
			t.Errorf("%q: stored %+v (%v), want literal", p, got, err)
		}
	}
	var tables int
	if err := l.db.QueryRow(context.Background(), `SELECT count(*) FROM information_schema.tables WHERE table_name='semilleros'`).Scan(&tables); err != nil || tables != 1 {
		t.Fatalf("semilleros table after payloads: %d %v", tables, err)
	}
	for _, q := range []string{`' OR '1'='1`, `%' OR 1=1--`, `'; SELECT * FROM usuarios;--`} {
		status, _, text := l.call(t, http.MethodGet, "/members?q="+url.QueryEscape(q), "", "")
		if status != http.StatusOK && status != http.StatusBadRequest {
			t.Errorf("members q=%q: status=%d %s", q, status, text)
		}
		if strings.Contains(text, l.run) {
			t.Errorf("members q=%q leaked synthetic accounts: %s", q, text)
		}
	}
}

// TDB-T15: anonymous public responses never expose document, email or Firebase UID of members.
func TestLivePublicResponsesHidePersonalData(t *testing.T) {
	l := newLive(t)
	ctx := context.Background()
	if _, err := l.db.Exec(ctx, `UPDATE usuarios SET dni=$1, tipo_dni='CC', telefono='3000000000' WHERE id=$2`, l.run+"-dni", l.ids["docente"]); err != nil {
		t.Fatal(err)
	}
	if _, err := l.db.Exec(ctx, `INSERT INTO perfil_investigador (usuario_id,visible_publicamente) VALUES ($1,true) ON CONFLICT (usuario_id) DO UPDATE SET visible_publicamente=true`, l.ids["docente"]); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = l.db.Exec(context.Background(), `DELETE FROM perfil_investigador WHERE usuario_id=$1`, l.ids["docente"])
	})
	admin := l.session(t, "admin", "1 hour", false)
	body, _ := json.Marshal(map[string]string{"name": l.run + "-pub", "description": "d", "researchLine": "IA", "objective": "o", "responsibleTeacherId": l.ids["docente"], "contactEmail": "semillero@example.test"})
	status, _, text := l.call(t, http.MethodPost, "/seedbeds", admin, string(body))
	var seedbed struct{ ID string }
	if status != http.StatusCreated || json.Unmarshal([]byte(text), &seedbed) != nil {
		t.Fatalf("create seedbed: %d %s", status, text)
	}
	if status, _, text := l.call(t, http.MethodPatch, "/seedbeds/"+seedbed.ID+"/visibility", admin, `{"status":"Publicado"}`); status != http.StatusOK {
		t.Fatalf("publish seedbed: %d %s", status, text)
	}
	secrets := []string{l.run + "-dni", l.run + "-docente@example.test", l.run + "-docente", "3000000000"}
	for _, path := range []string{"/members", "/members?q=Prueba", "/members/" + l.ids["docente"], "/seedbeds/" + seedbed.ID, "/seedbeds/" + seedbed.ID + "/space", "/seedbeds/" + seedbed.ID + "/members"} {
		status, _, text := l.call(t, http.MethodGet, path, "", "")
		if status != http.StatusOK {
			t.Errorf("%s: status %d, want 200 so the check is not vacuous: %s", path, status, text)
		}
		for _, s := range secrets {
			if strings.Contains(text, s) {
				t.Errorf("%s (anonymous) exposes %q: %s", path, s, text)
			}
		}
	}
}

// PT-NF-04 / D037: behind the TLS proxy (X-Forwarded-Proto: https) the API sends HSTS; nosniff always.
func TestLiveSecurityHeaders(t *testing.T) {
	l := newLive(t)
	for proto, wantHSTS := range map[string]bool{"https": true, "": false} {
		request, _ := http.NewRequest(http.MethodGet, l.api+"/seedbeds", nil)
		if proto != "" {
			request.Header.Set("X-Forwarded-Proto", proto)
		}
		response, err := http.DefaultClient.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if got := response.Header.Get("Strict-Transport-Security") != ""; got != wantHSTS {
			t.Errorf("proto %q: HSTS present=%v, want %v", proto, got, wantHSTS)
		}
		if response.Header.Get("X-Content-Type-Options") != "nosniff" {
			t.Errorf("proto %q: missing nosniff", proto)
		}
	}
}
