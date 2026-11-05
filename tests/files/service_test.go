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
