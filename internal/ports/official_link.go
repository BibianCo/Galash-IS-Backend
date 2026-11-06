package ports

import (
	"context"
	"github.com/galash-uptc/backend/internal/domain"
)

type OfficialLinkRepository interface {
	ListVisible(ctx context.Context, category string, includeAll bool) ([]domain.OfficialLink, error)
	Find(ctx context.Context, id string) (domain.OfficialLink, error)
	Create(ctx context.Context, link domain.OfficialLink) (domain.OfficialLink, error)
	Update(ctx context.Context, id, actor string, link domain.OfficialLink) (domain.OfficialLink, error)
	SetStatus(ctx context.Context, id, actor, status string) (domain.OfficialLink, error)
}
