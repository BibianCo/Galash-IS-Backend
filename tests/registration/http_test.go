package registration_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	httpadapter "github.com/galash-uptc/backend/internal/adapters/inbound/http"
	authapp "github.com/galash-uptc/backend/internal/application/auth"
	application "github.com/galash-uptc/backend/internal/application/registration"
	"github.com/galash-uptc/backend/internal/domain"
)

type identityVerifier struct{}

func (identityVerifier) Verify(context.Context, string) (domain.Identity, error) {
	return domain.Identity{}, nil
}

type users struct{}

func (users) Upsert(context.Context, domain.User) (string, error) { return "", nil }

type healthChecker struct{}

func (healthChecker) Ping(context.Context) error { return nil }

func TestCreateWithoutSessionReturns401(t *testing.T) {
	repo := &repository{}
	handler := httpadapter.NewHandlerWithRegistrations(authapp.NewService(identityVerifier{}, users{}), healthChecker{}, nil, application.NewService(repo)).Routes()
	body := `{"lugarNacimiento":"Tunja","direccionResidencia":"Calle 1","linkCvLAC":"https://cv","linkORCID":"https://orcid","maximoNivelEstudio":"Pregrado","tituloMaximoNivelEstudio":"Ing","tituloPregrado":"Ing","vinculacion":"Estudiante","categoriaActualInvestigador":"Junior","convocatoriaMinCiencias":"2025"}`
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/registration-requests", strings.NewReader(body)))
	if response.Code != http.StatusUnauthorized || repo.hash != "" {
		t.Fatalf("status=%d hash=%q body=%s", response.Code, repo.hash, response.Body.String())
	}
}
