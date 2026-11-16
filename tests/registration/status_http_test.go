package registration_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	httpadapter "github.com/galash-uptc/backend/internal/adapters/inbound/http"
	postgresadapter "github.com/galash-uptc/backend/internal/adapters/outbound/postgres"
	authapp "github.com/galash-uptc/backend/internal/application/auth"
	application "github.com/galash-uptc/backend/internal/application/registration"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PT-AUTH-05: the applicant reads their own request in every state; another account gets 404.
func TestApplicantReadsOwnRequestStatusOnly(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	run := fmt.Sprintf("status-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		_, _ = db.Exec(ctx, `DELETE FROM solicitudes_registro WHERE usuario_id IN (SELECT id FROM usuarios WHERE firebase_uid LIKE $1)`, run+"%")
		_, _ = db.Exec(ctx, `DELETE FROM usuarios WHERE firebase_uid LIKE $1`, run+"%")
		db.Close()
	})
	store := &accounts{sessions: map[string]string{}, byID: map[string]domain.User{}}
	ids := map[string]string{}
	for _, name := range []string{"owner", "other"} {
		var id string
		if err := db.QueryRow(ctx, `INSERT INTO usuarios (firebase_uid,nombre,email,rol) VALUES ($1,'Prueba',$1||'@example.test','visitante') RETURNING id`, run+"-"+name).Scan(&id); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
		ids[name] = id
		store.byID[id] = domain.User{ID: id, FirebaseUID: id, Role: "visitante", Status: "activo"}
	}
	auth := authapp.NewService(tokenVerifier{}, store)
	handler := httpadapter.NewHandlerWithRegistrations(auth, healthChecker{}, nil, application.NewService(postgresadapter.NewRegistrationRepository(db))).Routes()
	get := func(path, user string) (int, map[string]any) {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		if user != "" {
			login, err := auth.Login(ctx, ids[user])
			if err != nil {
				t.Fatalf("login %s: %v", user, err)
			}
			request.Header.Set("Authorization", "Bearer "+login.Token)
		}
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		body := map[string]any{}
		_ = json.Unmarshal(response.Body.Bytes(), &body)
		return response.Code, body
	}

	for _, state := range []string{domain.RegistrationFiled, domain.RegistrationReview, domain.RegistrationApproved, domain.RegistrationRejected, domain.RegistrationPending} {
		token := run + state
		digest := sha256.Sum256([]byte(token))
		var id string
		if err := db.QueryRow(ctx, `INSERT INTO solicitudes_registro (usuario_id,lugar_nacimiento,direccion_residencia,link_cvlac,link_orcid,maximo_nivel_estudio,titulo_maximo_nivel_estudio,titulo_pregrado,vinculacion,categoria_actual_investigador,convocatoria_minciencias,estado,observacion,token_hash)
			VALUES ($1,'Tunja','Calle 1','https://cv.example','https://orcid.example','Pregrado','','Ing','Estudiante','Junior','2025',$2,'obs',$3) RETURNING id`,
			ids["owner"], state, base64.RawURLEncoding.EncodeToString(digest[:])).Scan(&id); err != nil {
			t.Fatalf("seed %s: %v", state, err)
		}
		path := "/registration-requests/" + id + "/status"
		if code, body := get(path, "owner"); code != http.StatusOK || body["estado"] != state || body["observacion"] != "obs" {
			t.Errorf("%s owner: %d %v", state, code, body)
		}
		if code, body := get(path+"?trackingToken="+url.QueryEscape(token), ""); code != http.StatusOK || body["estado"] != state {
			t.Errorf("%s tracking token: %d %v", state, code, body)
		}
		if code, body := get(path, "other"); code != http.StatusNotFound || body["estado"] != nil {
			t.Errorf("%s other account: %d %v, want 404 without data", state, code, body)
		}
		if code, _ := get(path+"?trackingToken=wrong", ""); code != http.StatusNotFound {
			t.Errorf("%s wrong token: %d, want 404", state, code)
		}
		if code, _ := get(path, ""); code != http.StatusUnauthorized {
			t.Errorf("%s anonymous: %d, want 401", state, code)
		}
	}
}
