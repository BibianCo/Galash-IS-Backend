package member_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	postgresadapter "github.com/galash-uptc/backend/internal/adapters/outbound/postgres"
	memberapp "github.com/galash-uptc/backend/internal/application/member"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

func cvlacService(t *testing.T) (*memberapp.Service, string) {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}
	ctx := context.Background()
	db, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	uid := fmt.Sprintf("cvlac-%d", time.Now().UnixNano())
	var id string
	if err := db.QueryRow(ctx, `INSERT INTO usuarios (firebase_uid,nombre,email,rol) VALUES ($1,'Docente',$1||'@example.test','docente') RETURNING id`, uid).Scan(&id); err != nil {
		t.Fatalf("seed: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.Exec(ctx, `DELETE FROM perfil_investigador WHERE usuario_id=$1`, id)
		_, _ = db.Exec(ctx, `DELETE FROM usuarios WHERE id=$1`, id)
		db.Close()
	})
	return memberapp.NewService(postgresadapter.NewUserRepository(db)), id
}

// PT-INT-03: a valid CvLAC link is stored, shown on the public profile and replaced; invalid links are rejected.
func TestTeacherCvLACLinkIsValidatedAndShown(t *testing.T) {
	service, id := cvlacService(t)
	ctx := context.Background()
	if p, err := service.Get(ctx, id, false); err != nil || p.CvLAC != "" {
		t.Fatalf("without link: cvlac=%q err=%v", p.CvLAC, err)
	}
	for _, bad := range []string{"javascript:alert(1)", "ftp://cv.example/x", "cvlac", "https://"} {
		if _, err := service.Update(ctx, id, memberapp.UpdateCommand{CvLAC: &bad}); !errors.Is(err, domain.ErrInvalidMember) {
			t.Errorf("%q: err=%v, want ErrInvalidMember", bad, err)
		}
	}
	for _, link := range []string{"https://scienti.minciencias.gov.co/cvlac/visualizador/generarCurriculoCv.do?cod_rh=0000000001", "https://scienti.minciencias.gov.co/cvlac/visualizador/generarCurriculoCv.do?cod_rh=0000000002"} {
		if _, err := service.Update(ctx, id, memberapp.UpdateCommand{CvLAC: &link}); err != nil {
			t.Fatalf("update %q: %v", link, err)
		}
		if p, err := service.Get(ctx, id, false); err != nil || p.CvLAC != link {
			t.Fatalf("public profile cvlac=%q err=%v, want %q", p.CvLAC, err, link)
		}
	}
}

// PT-INT-03: removing the link hides it from the profile.
func TestTeacherCvLACLinkCanBeRemoved(t *testing.T) {
	service, id := cvlacService(t)
	ctx := context.Background()
	link, empty := "https://cv.example/1", ""
	if _, err := service.Update(ctx, id, memberapp.UpdateCommand{CvLAC: &link}); err != nil {
		t.Fatalf("set: %v", err)
	}
	// Omitting cvlac keeps the link.
	if _, err := service.Update(ctx, id, memberapp.UpdateCommand{Area: "IA"}); err != nil {
		t.Fatalf("other field: %v", err)
	}
	if p, _ := service.Get(ctx, id, false); p.CvLAC != link {
		t.Fatalf("cvlac after unrelated update = %q, want %q", p.CvLAC, link)
	}
	if _, err := service.Update(ctx, id, memberapp.UpdateCommand{CvLAC: &empty}); err != nil {
		t.Fatalf("clear: %v", err)
	}
	if p, _ := service.Get(ctx, id, false); p.CvLAC != "" {
		t.Fatalf("cvlac after removal = %q, want empty", p.CvLAC)
	}
}
