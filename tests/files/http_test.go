package files_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"strings"
	"testing"

	httpadapter "github.com/galash-uptc/backend/internal/adapters/inbound/http"
	auditapp "github.com/galash-uptc/backend/internal/application/audit"
	authapp "github.com/galash-uptc/backend/internal/application/auth"
	fileapp "github.com/galash-uptc/backend/internal/application/file"
	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
)

type healthChecker struct{}

func (healthChecker) Ping(context.Context) error { return nil }

type verifier struct{}

func (verifier) Verify(_ context.Context, token string) (domain.Identity, error) {
	return domain.Identity{UID: token, Email: token + "@example.com"}, nil
}

type authStore struct {
	users    map[string]domain.User
	sessions map[string]string
}

func (a *authStore) FindByID(_ context.Context, id string) (domain.User, error) {
	u, ok := a.users[id]
	if !ok {
		return domain.User{}, errors.New("user not found")
	}
	return u, nil
}
func (a *authStore) FindByEmail(context.Context, string) (domain.User, error) {
	return domain.User{}, errors.New("not used")
}
func (a *authStore) FindByFirebaseUID(ctx context.Context, uid string) (domain.User, error) {
	return a.FindByID(ctx, uid)
}
func (*authStore) SetPassword(context.Context, string, string) error                { return nil }
func (*authStore) Search(context.Context, string, int, bool) ([]domain.User, error) { return nil, nil }
func (*authStore) UpdateRole(context.Context, string, string) error                 { return nil }
func (*authStore) Upsert(context.Context, domain.User) (string, error)              { return "", nil }
func (a *authStore) Create(_ context.Context, userID, hash string) error {
	a.sessions[hash] = userID
	return nil
}
func (a *authStore) FindUserID(_ context.Context, hash string) (string, error) {
	if id, ok := a.sessions[hash]; ok {
		return id, nil
	}
	return "", errors.New("session not found")
}
func (*authStore) Revoke(context.Context, string) error           { return nil }
func (*authStore) RevokeAllForUser(context.Context, string) error { return nil }
func (*authStore) DeleteExpiredSessions(context.Context) error    { return nil }

// memoryFiles mirrors the adapter rule used here: Admin or uploader can read.
type memoryFiles struct {
	items   map[string]domain.FileAttachment
	content map[string][]byte
}

func (r *memoryFiles) Create(_ context.Context, f domain.FileAttachment, content []byte) (domain.FileAttachment, error) {
	if f.EntityID == inaccessibleEntityID {
		return f, domain.ErrForbidden
	}
	f.ID = fmt.Sprint(len(r.items) + 1)
	r.items[f.ID], r.content[f.ID] = f, content
	return f, nil
}
func (r *memoryFiles) Find(_ context.Context, id, actorID, role string) (domain.FileAttachment, []byte, error) {
	f, ok := r.items[id]
	if !ok || f.UploadedBy != actorID && !strings.EqualFold(role, "admin") {
		return domain.FileAttachment{}, nil, domain.ErrFileNotFound
	}
	return f, r.content[id], nil
}

type noAudit struct{}

func (noAudit) Append(context.Context, domain.AuditLog) error { return nil }
func (noAudit) List(context.Context, ports.AuditFilter) ([]domain.AuditLog, error) {
	return nil, nil
}

type fixture struct {
	entity  string
	server  http.Handler
	repo    *memoryFiles
	cookies map[string]*http.Cookie
}

const (
	entityID             = "00000000-0000-0000-0000-000000000001"
	inaccessibleEntityID = "00000000-0000-0000-0000-000000000002"
)

func newFixture(t *testing.T) fixture {
	t.Helper()
	store := &authStore{sessions: map[string]string{}, users: map[string]domain.User{
		"admin":     {ID: "admin", Role: "Admin", Status: "activo"},
		"student":   {ID: "student", Role: "estudiante", Status: "activo"},
		"teacher":   {ID: "teacher", Role: "docente", Status: "activo"},
		"visitante": {ID: "visitante", Role: "visitante", Status: "activo"},
	}}
	repo := &memoryFiles{items: map[string]domain.FileAttachment{}, content: map[string][]byte{}}
	auth := authapp.NewService(verifier{}, store)
	server := httpadapter.NewHandlerWithOfficialLinksAndAudit(auth, healthChecker{}, nil, nil, nil, nil, nil, nil, nil, nil, nil, auditapp.NewService(noAudit{}), fileapp.NewService(repo)).Routes()
	f := fixture{entity: entityID, server: server, repo: repo, cookies: map[string]*http.Cookie{}}
	for id := range store.users {
		req := httptest.NewRequest(http.MethodPost, "/auth/sessions", strings.NewReader(`{"token":"`+id+`"}`))
		res := httptest.NewRecorder()
		server.ServeHTTP(res, req)
		if res.Code != http.StatusCreated {
			t.Fatalf("login %s status=%d", id, res.Code)
		}
		f.cookies[id] = res.Result().Cookies()[0]
	}
	return f
}

func (f fixture) upload(t *testing.T, user, entityType, name, declared string, content []byte) *httptest.ResponseRecorder {
	t.Helper()
	var body bytes.Buffer
	w := multipart.NewWriter(&body)
	_ = w.WriteField("entityType", entityType)
	_ = w.WriteField("entityId", f.entity)
	if name != "" {
		h := textproto.MIMEHeader{}
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, name))
		h.Set("Content-Type", declared)
		part, _ := w.CreatePart(h)
		_, _ = part.Write(content)
	}
	_ = w.Close()
	req := httptest.NewRequest(http.MethodPost, "/files", &body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	if c := f.cookies[user]; c != nil {
		req.AddCookie(c)
	}
	res := httptest.NewRecorder()
	f.server.ServeHTTP(res, req)
	return res
}

func (f fixture) get(user, id string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, "/files/"+id, nil)
	if c := f.cookies[user]; c != nil {
		req.AddCookie(c)
	}
	res := httptest.NewRecorder()
	f.server.ServeHTTP(res, req)
	return res
}

var (
	pdf  = []byte("%PDF-1.4\n%test\n")
	png  = []byte("\x89PNG\r\n\x1a\n0000")
	docx = append([]byte("PK\x03\x04"), make([]byte, 32)...)
)

// PT-COM-01 / PT-TR-01: upload requires an active Estudiante, Docente or Admin.
func TestUploadAccessControl(t *testing.T) {
	f := newFixture(t)
	if res := f.upload(t, "", "event", "a.pdf", "application/pdf", pdf); res.Code != http.StatusUnauthorized {
		t.Fatalf("without session status=%d", res.Code)
	}
	if res := f.upload(t, "visitante", "event", "a.pdf", "application/pdf", pdf); res.Code != http.StatusForbidden {
		t.Fatalf("visitante status=%d", res.Code)
	}
	for _, user := range []string{"student", "teacher", "admin"} {
		res := f.upload(t, user, "event", "a.pdf", "application/pdf", pdf)
		var got domain.FileAttachment
		if res.Code != http.StatusCreated || json.Unmarshal(res.Body.Bytes(), &got) != nil {
			t.Fatalf("%s status=%d body=%s", user, res.Code, res.Body.String())
		}
		if got.ID == "" || got.UploadedBy != user || got.ContentType != "application/pdf" || got.Size != int64(len(pdf)) || got.EntityID != entityID {
			t.Fatalf("%s uploaded %+v", user, got)
		}
	}
}

// PT-COM-01 / PT-SEM-06: size, content sniffing and multipart validation.
func TestUploadValidation(t *testing.T) {
	f := newFixture(t)
	cases := []struct {
		name, entity, file, declared string
		content                      []byte
	}{
		{"missing file", "event", "", "", nil},
		{"text disguised as pdf", "event", "a.pdf", "application/pdf", []byte("hola mundo")},
		{"html disguised as png", "event", "a.png", "image/png", []byte("<html><script>x</script></html>")},
		{"png named pdf", "event", "a.pdf", "application/pdf", png},
		{"zip declared pdf", "event", "a.pdf", "application/pdf", docx},
		{"unknown entity", "user", "a.pdf", "application/pdf", pdf},
		{"empty file", "event", "a.pdf", "application/pdf", []byte{}},
		{"over 10 MB", "event", "a.pdf", "application/pdf", append(append([]byte{}, pdf...), make([]byte, fileapp.MaxSize)...)},
	}
	for _, c := range cases {
		if res := f.upload(t, "student", c.entity, c.file, c.declared, c.content); res.Code != http.StatusBadRequest {
			t.Fatalf("%s status=%d body=%s", c.name, res.Code, res.Body.String())
		}
	}
	req := httptest.NewRequest(http.MethodPost, "/files", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(f.cookies["student"])
	res := httptest.NewRecorder()
	f.server.ServeHTTP(res, req)
	if res.Code != http.StatusBadRequest {
		t.Fatalf("non-multipart status=%d", res.Code)
	}
	if len(f.repo.items) != 0 {
		t.Fatalf("invalid uploads were stored: %+v", f.repo.items)
	}
	for name, c := range map[string]struct {
		declared string
		content  []byte
	}{"a.png": {"image/png", png}, "a.docx": {"application/vnd.openxmlformats-officedocument.wordprocessingml.document", docx}} {
		if res := f.upload(t, "teacher", "announcement", name, c.declared, c.content); res.Code != http.StatusCreated {
			t.Fatalf("%s status=%d body=%s", name, res.Code, res.Body.String())
		}
	}
}

// PT-COM-01: authorized download returns content as attachment; others get 404.
func TestDownloadAuthorization(t *testing.T) {
	f := newFixture(t)
	res := f.upload(t, "student", "evidence", "informe.pdf", "application/pdf", pdf)
	var up domain.FileAttachment
	if res.Code != http.StatusCreated || json.Unmarshal(res.Body.Bytes(), &up) != nil {
		t.Fatalf("upload status=%d", res.Code)
	}
	if res := f.get("", up.ID); res.Code != http.StatusUnauthorized {
		t.Fatalf("without session status=%d", res.Code)
	}
	if res := f.get("teacher", up.ID); res.Code != http.StatusNotFound {
		t.Fatalf("unrelated user status=%d", res.Code)
	}
	if res := f.get("admin", "999"); res.Code != http.StatusNotFound {
		t.Fatalf("missing file status=%d", res.Code)
	}
	for _, user := range []string{"student", "admin"} {
		res := f.get(user, up.ID)
		if res.Code != http.StatusOK || !bytes.Equal(res.Body.Bytes(), pdf) || res.Header().Get("Content-Type") != "application/pdf" || res.Header().Get("Content-Disposition") != `attachment; filename="informe.pdf"` {
			t.Fatalf("%s download status=%d headers=%v", user, res.Code, res.Header())
		}
	}
}

// D032: the repository's entity access decision surfaces as 403 and nothing is stored.
func TestUploadRejectsInaccessibleEntity(t *testing.T) {
	f := newFixture(t)
	f.entity = inaccessibleEntityID
	if res := f.upload(t, "student", "proposal", "a.pdf", "application/pdf", pdf); res.Code != http.StatusForbidden {
		t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
	}
	if len(f.repo.items) != 0 {
		t.Fatalf("stored %+v", f.repo.items)
	}
}
