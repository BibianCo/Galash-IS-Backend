package file

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"path/filepath"
	"slices"
	"strings"

	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
)

const MaxSize int64 = 10 << 20

// Explicit extensions: the alpine runtime image has no /etc/mime.types, so
// mime.ExtensionsByType cannot be trusted for Office formats (D028).
var allowedTypes = map[string][]string{
	"application/pdf":    {".pdf"},
	"application/msword": {".doc"},
	"application/vnd.openxmlformats-officedocument.wordprocessingml.document": {".docx"},
	"application/vnd.ms-excel": {".xls"},
	"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet": {".xlsx"},
	"image/jpeg": {".jpg", ".jpeg"},
	"image/png":  {".png"},
}

type Service struct{ repo ports.FileRepository }

func NewService(repo ports.FileRepository) *Service { return &Service{repo: repo} }

func validEntity(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "announcement", "event", "activity", "project", "proposal", "evidence":
		return true
	default:
		return false
	}
}

func validFile(file domain.FileAttachment, content []byte) bool {
	if !validEntity(file.EntityType) || uuid.Validate(file.EntityID) != nil || strings.TrimSpace(file.Name) == "" || filepath.Base(file.Name) != file.Name || len(content) == 0 || int64(len(content)) > MaxSize {
		return false
	}
	typeName := strings.ToLower(strings.TrimSpace(file.ContentType))
	return file.Size == int64(len(content)) && slices.Contains(allowedTypes[typeName], strings.ToLower(filepath.Ext(file.Name)))
}

func (s *Service) Create(ctx context.Context, file domain.FileAttachment, content []byte) (domain.FileAttachment, error) {
	if !validFile(file, content) {
		return file, domain.ErrInvalidFile
	}
	return s.repo.Create(ctx, file, content)
}

func (s *Service) Find(ctx context.Context, id, actorID, role string) (domain.FileAttachment, []byte, error) {
	if strings.TrimSpace(id) == "" || strings.TrimSpace(actorID) == "" {
		return domain.FileAttachment{}, nil, domain.ErrFileNotFound
	}
	file, content, err := s.repo.Find(ctx, id, actorID, role)
	if errors.Is(err, domain.ErrForbidden) || errors.Is(err, domain.ErrFileNotFound) {
		return file, content, err
	}
	return file, content, err
}
