package audit

import (
	"context"

	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
)

type Service struct{ repo ports.AuditRepository }

func NewService(repo ports.AuditRepository) *Service { return &Service{repo: repo} }

func (s *Service) Record(ctx context.Context, item domain.AuditLog) error {
	return s.repo.Append(ctx, item)
}

func (s *Service) List(ctx context.Context, filter ports.AuditFilter) ([]domain.AuditLog, error) {
	if filter.From != nil && filter.To != nil && filter.From.After(*filter.To) {
		return nil, domain.ErrInvalidAuditPeriod
	}
	return s.repo.List(ctx, filter)
}
