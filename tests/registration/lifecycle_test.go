package registration_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	postgresadapter "github.com/galash-uptc/backend/internal/adapters/outbound/postgres"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type lifecycle struct {
	repo     *postgresadapter.RegistrationRepository
	db       *pgxpool.Pool
	run      string
	seedbeds []string
	n        int
}

// newLifecycle seeds two seedbeds; request() adds an applicant with a complete profile and files a request.
func newLifecycle(t *testing.T) *lifecycle {
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
	l := &lifecycle{repo: postgresadapter.NewRegistrationRepository(db), db: db, run: fmt.Sprintf("life-%d", time.Now().UnixNano())}
	for i := 0; i < 2; i++ {
		var id string
		if err := db.QueryRow(ctx, `INSERT INTO semilleros (nombre) VALUES ($1) RETURNING id`, fmt.Sprintf("%s-s%d", l.run, i)).Scan(&id); err != nil {
			t.Fatalf("seed seedbed: %v", err)
		}
		l.seedbeds = append(l.seedbeds, id)
	}
	t.Cleanup(func() {
		users := `(SELECT id FROM usuarios WHERE firebase_uid LIKE '` + l.run + `%')`
		for _, q := range []string{
			`DELETE FROM semillero_miembros WHERE usuario_id IN ` + users,
			`DELETE FROM notificaciones WHERE usuario_id IN ` + users,
			`DELETE FROM seguimiento_solicitud WHERE solicitud_id IN (SELECT id FROM solicitudes_registro WHERE usuario_id IN ` + users + `)`,
			`DELETE FROM solicitudes_registro WHERE usuario_id IN ` + users,
			`DELETE FROM usuarios WHERE firebase_uid LIKE '` + l.run + `%'`,
			`DELETE FROM semilleros WHERE nombre LIKE '` + l.run + `%'`,
		} {
			_, _ = db.Exec(ctx, q)
		}
		db.Close()
	})
	return l
}

func (l *lifecycle) request(t *testing.T, seedbed string) (domain.RegistrationRequest, string) {
	t.Helper()
	ctx := context.Background()
	l.n++
	uid := fmt.Sprintf("%s-u%d", l.run, l.n)
	var user string
	if err := l.db.QueryRow(ctx, `INSERT INTO usuarios (firebase_uid,nombre,apellido,email,dni,fecha_nacimiento,telefono,rol) VALUES ($1,'Ana','Prueba',$1||'@example.test',$1,'2000-01-01','300','visitante') RETURNING id`, uid).Scan(&user); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	q := validRequest()
	q.UserID, q.SeedbedID = user, seedbed
	created, err := l.repo.CreateRegistration(ctx, q, uid+"-hash")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	return created, user
}

// TDB-T04: requests walk every allowed state keeping id, applicant, seedbed and an ordered dated history.
func TestRegistrationLifecycleKeepsIdentityAndHistory(t *testing.T) {
	l := newLifecycle(t)
	ctx := context.Background()
	for name, path := range map[string][]string{
		"approved": {domain.RegistrationPending, domain.RegistrationReview, domain.RegistrationApproved},
		"rejected": {domain.RegistrationReview, domain.RegistrationRejected},
	} {
		created, user := l.request(t, l.seedbeds[0])
		if created.Status != domain.RegistrationFiled || created.ID == "" {
			t.Fatalf("%s: created %+v", name, created)
		}
		for _, state := range path {
			got, err := l.repo.UpdateRegistrationStatus(ctx, created.ID, state, "obs "+state, user)
			if err != nil || got.ID != created.ID || got.Status != state || got.UserID != user || got.SeedbedID != l.seedbeds[0] {
				t.Fatalf("%s -> %s: %+v %v", name, state, got, err)
			}
		}
		if _, err := l.repo.UpdateRegistrationStatus(ctx, created.ID, domain.RegistrationReview, "reabrir", user); !errors.Is(err, domain.ErrInvalidRegistrationTransition) {
			t.Errorf("%s: reopening a closed request err=%v", name, err)
		}
		detail, err := l.repo.GetRegistrationDetail(ctx, created.ID)
		if err != nil {
			t.Fatal(err)
		}
		want := append([]string{domain.RegistrationFiled}, path...)
		if len(detail.History) != len(want) {
			t.Fatalf("%s: history %+v, want states %v", name, detail.History, want)
		}
		for i, h := range detail.History {
			if h.Status != want[i] || h.At.IsZero() || (i > 0 && h.At.Before(detail.History[i-1].At)) {
				t.Errorf("%s: history[%d] = %+v, want %s in date order", name, i, h, want[i])
			}
		}
		var role string
		var members int
		_ = l.db.QueryRow(ctx, `SELECT rol, (SELECT count(*) FROM semillero_miembros WHERE usuario_id=u.id AND retirado_en IS NULL) FROM usuarios u WHERE id=$1`, user).Scan(&role, &members)
		if approved := name == "approved"; (role == "estudiante") != approved || (members == 1) != approved {
			t.Errorf("%s: role=%s active memberships=%d", name, role, members)
		}
	}
}

// TDB-T12: list filters return exactly the prepared dataset counts.
func TestRegistrationFiltersMatchPreparedCounts(t *testing.T) {
	l := newLifecycle(t)
	ctx := context.Background()
	// Seedbed 0: 3 Radicada + 2 En revisión; seedbed 1: 1 Radicada.
	for i := 0; i < 5; i++ {
		created, user := l.request(t, l.seedbeds[0])
		if i >= 3 {
			if _, err := l.repo.UpdateRegistrationStatus(ctx, created.ID, domain.RegistrationReview, "ok", user); err != nil {
				t.Fatal(err)
			}
		}
	}
	l.request(t, l.seedbeds[1])
	today := time.Now().Format("2006-01-02")
	for name, c := range map[string]struct {
		status, from, seedbed string
		scope                 []string
		want                  int
	}{
		"both seedbeds":        {"", "", "", l.seedbeds, 6},
		"seedbed 0":            {"", "", l.seedbeds[0], l.seedbeds, 5},
		"filed in both":        {domain.RegistrationFiled, "", "", l.seedbeds, 4},
		"in review seedbed 0":  {domain.RegistrationReview, "", l.seedbeds[0], l.seedbeds, 2},
		"approved":             {domain.RegistrationApproved, "", "", l.seedbeds, 0},
		"manager of seedbed 1": {"", "", "", l.seedbeds[1:], 1},
		"from today":           {"", today, "", l.seedbeds, 6},
		"from tomorrow":        {"", time.Now().AddDate(0, 0, 1).Format("2006-01-02"), "", l.seedbeds, 0},
	} {
		got, err := l.repo.ListRegistrations(ctx, c.status, c.from, "", c.seedbed, "", c.scope)
		if err != nil || len(got) != c.want {
			t.Errorf("%s: %d rows (%v), want %d", name, len(got), err, c.want)
		}
	}
}

// TDB-T11: two concurrent status changes on one request: exactly one wins and history has no partial rows.
func TestConcurrentRegistrationUpdatesHaveOneWinner(t *testing.T) {
	l := newLifecycle(t)
	ctx := context.Background()
	for round := 0; round < 20; round++ {
		created, user := l.request(t, l.seedbeds[0])
		var wg sync.WaitGroup
		errs := make([]error, 2)
		// Same target from two sessions: Review -> Review is not a transition, so a lost update would show as two winners.
		for i := range errs {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				_, errs[i] = l.repo.UpdateRegistrationStatus(ctx, created.ID, domain.RegistrationReview, fmt.Sprintf("sesión %d", i), user)
			}(i)
		}
		wg.Wait()
		ok := 0
		for _, err := range errs {
			if err == nil {
				ok++
			} else if !errors.Is(err, domain.ErrInvalidRegistrationTransition) {
				t.Fatalf("round %d: unexpected error %v", round, err)
			}
		}
		detail, err := l.repo.GetRegistrationDetail(ctx, created.ID)
		if err != nil {
			t.Fatal(err)
		}
		if ok != 1 || len(detail.History) != 2 || detail.History[1].Observation != detail.Observation {
			t.Fatalf("round %d: winners=%d status=%s history=%+v", round, ok, detail.Status, detail.History)
		}
	}
}
