package files_test

import (
	"context"
	"testing"

	fileapp "github.com/galash-uptc/backend/internal/application/file"
	"github.com/galash-uptc/backend/internal/domain"
)

type repository struct{}

func (repository) Create(context.Context, domain.FileAttachment, []byte) (domain.FileAttachment, error) {
	return domain.FileAttachment{ID: "file-1"}, nil
}
func (repository) Find(context.Context, string, string, string) (domain.FileAttachment, []byte, error) {
	return domain.FileAttachment{}, nil, nil
}

func TestCreateRejectsOversizedFile(t *testing.T) {
	s := fileapp.NewService(repository{})
	file := domain.FileAttachment{EntityType: "event", EntityID: "00000000-0000-0000-0000-000000000001", Name: "evidence.pdf", ContentType: "application/pdf", Size: fileapp.MaxSize + 1}
	_, err := s.Create(context.Background(), file, make([]byte, fileapp.MaxSize+1))
	if err != domain.ErrInvalidFile {
		t.Fatalf("error=%v", err)
	}
}

func TestCreateRejectsMismatchedTypeAndExtension(t *testing.T) {
	s := fileapp.NewService(repository{})
	file := domain.FileAttachment{EntityType: "proposal", EntityID: "00000000-0000-0000-0000-000000000001", Name: "evidence.exe", ContentType: "application/pdf", Size: 4}
	_, err := s.Create(context.Background(), file, []byte("test"))
	if err != domain.ErrInvalidFile {
		t.Fatalf("error=%v", err)
	}
}

// PT-COM-01 / PT-SEM-06: every allowed format is accepted with its usual
// extension regardless of the host MIME tables (runtime image is alpine).
func TestCreateAcceptsAllowedFormats(t *testing.T) {
	s := fileapp.NewService(repository{})
	for name, typ := range map[string]string{
		"a.pdf": "application/pdf", "a.doc": "application/msword", "a.xls": "application/vnd.ms-excel",
		"a.docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"a.xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"a.jpg":  "image/jpeg", "a.JPEG": "image/jpeg", "a.png": "image/png",
	} {
		file := domain.FileAttachment{EntityType: "announcement", EntityID: "00000000-0000-0000-0000-000000000001", Name: name, ContentType: typ, Size: 4}
		if _, err := s.Create(context.Background(), file, []byte("test")); err != nil {
			t.Fatalf("%s (%s): %v", name, typ, err)
		}
	}
}

// PT-COM-01: rejects unknown types, entities, IDs, empty or inconsistent content.
func TestCreateRejectsInvalidMetadata(t *testing.T) {
	s := fileapp.NewService(repository{})
	valid := domain.FileAttachment{EntityType: "event", EntityID: "00000000-0000-0000-0000-000000000001", Name: "a.pdf", ContentType: "application/pdf", Size: 4}
	cases := map[string]func(*domain.FileAttachment){
		"html type":       func(f *domain.FileAttachment) { f.Name, f.ContentType = "a.html", "text/html" },
		"svg type":        func(f *domain.FileAttachment) { f.Name, f.ContentType = "a.svg", "image/svg+xml" },
		"excel macro ext": func(f *domain.FileAttachment) { f.Name, f.ContentType = "a.xlm", "application/vnd.ms-excel" },
		"unknown entity":  func(f *domain.FileAttachment) { f.EntityType = "user" },
		"bad entity id":   func(f *domain.FileAttachment) { f.EntityID = "1" },
		"path in name":    func(f *domain.FileAttachment) { f.Name = "../a.pdf" },
		"blank name":      func(f *domain.FileAttachment) { f.Name = " " },
		"size mismatch":   func(f *domain.FileAttachment) { f.Size = 9 },
	}
	for name, mutate := range cases {
		file := valid
		mutate(&file)
		if _, err := s.Create(context.Background(), file, []byte("test")); err != domain.ErrInvalidFile {
			t.Fatalf("%s: error=%v", name, err)
		}
	}
	empty := valid
	empty.Size = 0
	if _, err := s.Create(context.Background(), empty, nil); err != domain.ErrInvalidFile {
		t.Fatalf("empty content: error=%v", err)
	}
}
