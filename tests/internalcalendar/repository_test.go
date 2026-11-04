package internalcalendar_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"testing"
	"time"

	postgresadapter "github.com/galash-uptc/backend/internal/adapters/outbound/postgres"
	calendarapp "github.com/galash-uptc/backend/internal/application/internalcalendar"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

// dbFixture: seedbed A led by director with student, delegate (student) and a
// retired student; seedbed B led by outsider (docente) with studentB.
type dbFixture struct {
	repo                                                            *postgresadapter.InternalCalendarRepository
	db                                                              *pgxpool.Pool
	admin, director, outsider, student, delegate, retired, studentB string
	seedbedA, seedbedB                                              string
	base                                                            time.Time
}

func (f dbFixture) user(id string) domain.User {
	roles := map[string]string{f.admin: "admin", f.director: "docente", f.outsider: "docente"}
	role := roles[id]
	if role == "" {
		role = "estudiante"
	}
	return domain.User{ID: id, Role: role, Status: "activo"}
}

func (f dbFixture) event(seedbed, typ string, start time.Duration, participants ...string) domain.InternalCalendarEvent {
	return domain.InternalCalendarEvent{SeedbedID: seedbed, Type: typ, Title: "Reunión " + typ, StartAt: f.base.Add(start), EndAt: f.base.Add(start + time.Hour), OrganizerID: f.director, ParticipantIDs: participants}
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
	run := fmt.Sprintf("m9-%d", time.Now().UnixNano())
	f := dbFixture{repo: postgresadapter.NewInternalCalendarRepository(db), db: db, base: time.Date(2031, 3, 3, 14, 0, 0, 0, time.UTC).Add(time.Duration(time.Now().UnixNano()%1000) * 24 * time.Hour)}
	users := []string{}
	for i, u := range []struct {
		id   *string
		role string
	}{{&f.admin, "admin"}, {&f.director, "docente"}, {&f.outsider, "docente"}, {&f.student, "estudiante"}, {&f.delegate, "estudiante"}, {&f.retired, "estudiante"}, {&f.studentB, "estudiante"}} {
		uid := fmt.Sprintf("%s-%d", run, i)
		if err := db.QueryRow(ctx, `INSERT INTO usuarios (firebase_uid,nombre,apellido,email,rol,estado) VALUES ($1,'Prueba',$2,$1||'@example.test',$2,'activo') RETURNING id`, uid, u.role).Scan(u.id); err != nil {
			t.Fatalf("seed user: %v", err)
		}
		users = append(users, *u.id)
	}
	for _, s := range []struct {
		id      *string
		name    string
		leader  string
		members []string
	}{{&f.seedbedA, run + "-A", f.director, []string{f.student, f.delegate, f.retired}}, {&f.seedbedB, run + "-B", f.outsider, []string{f.studentB}}} {
		if err := db.QueryRow(ctx, `INSERT INTO semilleros (nombre,lider_id) VALUES ($1,$2) RETURNING id`, s.name, s.leader).Scan(s.id); err != nil {
			t.Fatalf("seed seedbed: %v", err)
		}
		for _, m := range s.members {
			if _, err := db.Exec(ctx, `INSERT INTO semillero_miembros (semillero_id,usuario_id,delegado) VALUES ($1,$2,$2=$3::uuid)`, *s.id, m, f.delegate); err != nil {
				t.Fatalf("seed member: %v", err)
			}
		}
	}
	if _, err := db.Exec(ctx, `UPDATE semillero_miembros SET retirado_en=now() WHERE usuario_id=$1`, f.retired); err != nil {
		t.Fatalf("retire member: %v", err)
	}
	t.Cleanup(func() {
		for _, sql := range []string{
			`DELETE FROM cronograma_interno WHERE semillero_id = ANY($1::uuid[])`,
			`DELETE FROM semillero_miembros WHERE semillero_id = ANY($1::uuid[])`,
			`DELETE FROM semilleros WHERE id = ANY($1::uuid[])`,
		} {
			if _, err := db.Exec(ctx, sql, []string{f.seedbedA, f.seedbedB}); err != nil {
				t.Errorf("cleanup %q: %v", sql, err)
			}
		}
		// Other packages running in parallel may notify every admin, including this fixture's.
		for _, sql := range []string{`DELETE FROM notificaciones WHERE usuario_id = ANY($1::uuid[])`, `DELETE FROM usuarios WHERE id = ANY($1::uuid[])`} {
			if _, err := db.Exec(ctx, sql, users); err != nil {
				t.Errorf("cleanup %q: %v", sql, err)
			}
		}
		db.Close()
	})
	return f
}

func ids(list []domain.InternalCalendarEvent) []string {
	out := []string{}
	for _, e := range list {
		out = append(out, e.ID)
	}
	return out
}

func (f dbFixture) history(t *testing.T, id string) []string {
	t.Helper()
	rows, err := f.db.Query(context.Background(), `SELECT accion||':'||actor_id::text FROM cronograma_interno_historial WHERE evento_id=$1 ORDER BY creado_en, accion`, id)
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatalf("scan history: %v", err)
		}
		out = append(out, s)
	}
	return out
}

// TDB-T01 / TDB-T08: persistence, participants, history and logical cancellation.
func TestRepositoryPersistsParticipantsHistoryAndCancel(t *testing.T) {
	f := seedDB(t)
	ctx := context.Background()
	e, err := f.repo.Create(ctx, f.event(f.seedbedA, "tutoria", 0, f.student, f.delegate))
	if err != nil || e.ID == "" || e.Status != "activo" || e.OrganizerID != f.director || len(e.ParticipantIDs) != 2 {
		t.Fatalf("create = %+v, %v", e, err)
	}
	got, err := f.repo.Find(ctx, f.student, "estudiante", e.ID)
	if err != nil || len(got.Participants) != 2 || got.Participants[0].Name != "Prueba" || got.Participants[0].Role != "estudiante" || !got.StartAt.Equal(e.StartAt) {
		t.Fatalf("find = %+v, %v", got, err)
	}
	e.Title, e.ParticipantIDs = "Tutoría 2", []string{f.student}
	if u, err := f.repo.Update(ctx, e); err != nil || u.Title != "Tutoría 2" || u.UpdatedAt.Before(e.CreatedAt) {
		t.Fatalf("update = %+v, %v", u, err)
	}
	if got, _ := f.repo.Find(ctx, f.admin, "admin", e.ID); !slices.Equal(got.ParticipantIDs, []string{f.student}) {
		t.Errorf("participants after update = %v", got.ParticipantIDs)
	}
	c, err := f.repo.Cancel(ctx, e.ID, f.admin)
	if err != nil || c.Status != "cancelado" {
		t.Fatalf("cancel = %+v, %v", c, err)
	}
	if got, err := f.repo.Find(ctx, f.student, "estudiante", e.ID); err != nil || got.Status != "cancelado" || len(got.Participants) != 1 {
		t.Errorf("cancelled event must remain queryable: %+v, %v", got, err)
	}
	want := []string{"creado:" + f.director, "actualizado:" + f.director, "cancelado:" + f.admin}
	if h := f.history(t, e.ID); !slices.Equal(h, want) {
		t.Errorf("history = %v, want %v", h, want)
	}
}

// PT-CRO-01 / TDB-T12: students only see their seedbed; seedbed and date filters.
func TestRepositoryVisibilityAndFilters(t *testing.T) {
	f := seedDB(t)
	ctx := context.Background()
	a1, err := f.repo.Create(ctx, f.event(f.seedbedA, "reunion", 0, f.student))
	if err != nil {
		t.Fatalf("create a1: %v", err)
	}
	a2, _ := f.repo.Create(ctx, f.event(f.seedbedA, "hito", 48*time.Hour, f.delegate))
	b1, err := f.repo.Create(ctx, f.event(f.seedbedB, "reunion", 0, f.studentB))
	if err != nil {
		t.Fatalf("create b1: %v", err)
	}
	from, to := f.base.Add(-time.Hour), f.base.Add(24*time.Hour)
	cases := []struct {
		name, actor, role, seedbed string
		from, to                   *time.Time
		want                       []string
	}{
		{"student A", f.student, "estudiante", "", nil, nil, []string{a1.ID, a2.ID}},
		{"student A asks B", f.student, "estudiante", f.seedbedB, nil, nil, []string{}},
		{"student B", f.studentB, "estudiante", "", nil, nil, []string{b1.ID}},
		{"retired member", f.retired, "estudiante", "", nil, nil, []string{}},
		{"director A", f.director, "docente", "", nil, nil, []string{a1.ID, a2.ID}},
		{"admin seedbed B", f.admin, "admin", f.seedbedB, nil, nil, []string{b1.ID}},
		{"student A range", f.student, "estudiante", "", &from, &to, []string{a1.ID}},
		{"student A from", f.student, "estudiante", f.seedbedA, &to, nil, []string{a2.ID}},
	}
	for _, c := range cases {
		list, err := f.repo.List(ctx, c.actor, c.role, c.seedbed, c.from, c.to)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		// shared DB may hold other admin-visible rows; compare only this fixture's events.
		got := slices.DeleteFunc(ids(list), func(id string) bool { return !slices.Contains([]string{a1.ID, a2.ID, b1.ID}, id) })
		if !slices.Equal(got, c.want) {
			t.Errorf("%s: list = %v, want %v", c.name, got, c.want)
		}
	}
	if list, _ := f.repo.List(ctx, f.admin, "admin", "", &from, &to); !slices.Contains(ids(list), a1.ID) || !slices.Contains(ids(list), b1.ID) {
		t.Errorf("admin range list misses events")
	}
	for name, actor := range map[string]string{"student B": f.studentB, "retired": f.retired} {
		if _, err := f.repo.Find(ctx, actor, "estudiante", a1.ID); !errors.Is(err, domain.ErrInternalCalendarNotFound) {
			t.Errorf("%s find foreign event err = %v, want not found", name, err)
		}
	}
	if _, err := f.repo.Find(ctx, f.student, "estudiante", "not-a-uuid"); !errors.Is(err, domain.ErrInternalCalendarNotFound) {
		t.Errorf("invalid uuid err = %v, want not found", err)
	}
}

// TDB-T02 / TDB-T03: participant and seedbed references, constraints and schedule conflicts.
func TestRepositoryIntegrityAndConflicts(t *testing.T) {
	f := seedDB(t)
	ctx := context.Background()
	count := func() int {
		var n int
		_ = f.db.QueryRow(ctx, `SELECT count(*) FROM cronograma_interno WHERE semillero_id=$1`, f.seedbedA).Scan(&n)
		return n
	}
	for name, p := range map[string]string{"other seedbed": f.studentB, "retired": f.retired, "unknown": "00000000-0000-0000-0000-000000000000"} {
		if _, err := f.repo.Create(ctx, f.event(f.seedbedA, "reunion", 0, f.student, p)); !errors.Is(err, domain.ErrInvalidInternalCalendar) {
			t.Errorf("participant %s err = %v, want invalid", name, err)
		}
	}
	if n := count(); n != 0 {
		t.Errorf("rejected creates left %d rows (transaction not rolled back)", n)
	}
	if _, err := f.repo.Create(ctx, f.event("00000000-0000-0000-0000-000000000000", "reunion", 0, f.student)); err == nil {
		t.Errorf("unknown seedbed accepted")
	}
	if _, err := f.db.Exec(ctx, `INSERT INTO cronograma_interno (semillero_id,tipo,titulo,inicia_en,termina_en,organizador_id) VALUES ($1,'otro','x',now(),now()+interval '1 hour',$2)`, f.seedbedA, f.director); err == nil {
		t.Errorf("tipo CHECK not enforced")
	}
	if _, err := f.db.Exec(ctx, `INSERT INTO cronograma_interno (semillero_id,tipo,titulo,inicia_en,termina_en,organizador_id,estado) VALUES ($1,'hito','x',now(),now()+interval '1 hour',$2,'borrado')`, f.seedbedA, f.director); err == nil {
		t.Errorf("estado CHECK not enforced")
	}

	e, err := f.repo.Create(ctx, f.event(f.seedbedA, "reunion", 0, f.student))
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := f.repo.Update(ctx, domain.InternalCalendarEvent{ID: e.ID, SeedbedID: f.seedbedA, Type: "reunion", Title: "x", StartAt: e.StartAt, EndAt: e.EndAt, OrganizerID: f.director, ParticipantIDs: []string{f.studentB}}); !errors.Is(err, domain.ErrInvalidInternalCalendar) {
		t.Errorf("update with foreign participant err = %v, want invalid", err)
	}
	if got, _ := f.repo.Find(ctx, f.admin, "admin", e.ID); got.Title != e.Title || !slices.Equal(got.ParticipantIDs, []string{f.student}) {
		t.Errorf("failed update not rolled back: %+v", got)
	}
	overlap := f.event(f.seedbedA, "hito", 30*time.Minute, f.delegate, f.student)
	for name, c := range map[string]struct {
		e       domain.InternalCalendarEvent
		exclude string
		want    bool
	}{
		"overlap shared participant": {overlap, "", true},
		"overlap other participant":  {f.event(f.seedbedA, "hito", 30*time.Minute, f.delegate), "", false},
		"adjacent slot":              {f.event(f.seedbedA, "hito", time.Hour, f.student), "", false},
		"same event excluded":        {e, e.ID, false},
	} {
		if got, err := f.repo.HasConflict(ctx, c.e, c.exclude); err != nil || got != c.want {
			t.Errorf("%s: conflict = %v, %v; want %v", name, got, err, c.want)
		}
	}
	if _, err := f.repo.Cancel(ctx, e.ID, f.director); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if got, err := f.repo.HasConflict(ctx, overlap, ""); err != nil || got {
		t.Errorf("cancelled event still blocks schedule: %v, %v", got, err)
	}
	if _, err := f.repo.Cancel(ctx, "00000000-0000-0000-0000-000000000000", f.director); !errors.Is(err, domain.ErrInternalCalendarNotFound) {
		t.Errorf("cancel unknown err = %v, want not found", err)
	}
}

// PT-TR-01 / PT-CRO-01: contract M9 limits writes to the seedbed director, its delegates or Admin.
func TestServiceRestrictsWritesToSeedbedManagers(t *testing.T) {
	f := seedDB(t)
	ctx := context.Background()
	s := calendarapp.NewService(f.repo)
	e := f.event(f.seedbedA, "reunion", 0, f.student)
	if _, err := s.Create(ctx, e, f.user(f.outsider)); !errors.Is(err, domain.ErrForbidden) {
		t.Errorf("docente outside seedbed A creates event: err = %v, want forbidden", err)
	}
	own, err := s.Create(ctx, f.event(f.seedbedA, "tutoria", 2*time.Hour, f.student), f.user(f.director))
	if err != nil {
		t.Fatalf("director create: %v", err)
	}
	if _, err := s.Create(ctx, f.event(f.seedbedA, "hito", 4*time.Hour, f.student), f.user(f.admin)); err != nil {
		t.Errorf("admin create: %v", err)
	}
	for _, actor := range []string{f.student, f.studentB, f.outsider} {
		if _, err := s.Update(ctx, own.ID, own, f.user(actor)); err == nil {
			t.Errorf("non-manager %s updated event", f.user(actor).Role)
		}
		if _, err := s.Cancel(ctx, own.ID, f.user(actor)); err == nil {
			t.Errorf("non-manager %s cancelled event", f.user(actor).Role)
		}
	}
	if got, _ := f.repo.Find(ctx, f.admin, "admin", own.ID); got.Status != "activo" {
		t.Fatalf("rejected writes changed event: %+v", got)
	}
	// Delegates are managers per contract (seedbed relation, any active member role).
	if _, err := s.Create(ctx, f.event(f.seedbedA, "reunion", 6*time.Hour, f.student), f.user(f.delegate)); err != nil {
		t.Errorf("delegate create: err = %v, want success", err)
	}
	own.Title = "Tutoría por delegado"
	if _, err := s.Update(ctx, own.ID, own, f.user(f.delegate)); err != nil {
		t.Errorf("delegate update: err = %v, want success", err)
	}
	if _, err := s.Cancel(ctx, own.ID, f.user(f.delegate)); err != nil {
		t.Errorf("delegate cancel: err = %v, want success", err)
	}
}
