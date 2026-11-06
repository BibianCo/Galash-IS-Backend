package ports

import (
	"context"
	"time"

	"github.com/galash-uptc/backend/internal/domain"
)

type AuditFilter struct {
	ActorID string
	Action  string
	Entity  string
	From    *time.Time
	To      *time.Time
}

type AuditRepository interface {
	Append(ctx context.Context, log domain.AuditLog) error
	List(ctx context.Context, filter AuditFilter) ([]domain.AuditLog, error)
}
