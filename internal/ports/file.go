package ports

import (
	"context"
	"github.com/galash-uptc/backend/internal/domain"
)

type FileRepository interface {
	Create(ctx context.Context, file domain.FileAttachment, content []byte) (domain.FileAttachment, error)
	Find(ctx context.Context, id, actorID, role string) (domain.FileAttachment, []byte, error)
}
