package ports

import (
	"context"
	"github.com/galash-uptc/backend/internal/domain"
)

type AnnouncementRepository interface {
	List(ctx context.Context, userID string, includeAllDrafts bool) ([]domain.Announcement, error)
	Find(ctx context.Context, id, userID string) (domain.Announcement, error)
	Create(ctx context.Context, announcement domain.Announcement) (domain.Announcement, error)
	Update(ctx context.Context, id string, title, description string, attachments []domain.AnnouncementAttachment) (domain.Announcement, error)
	Publish(ctx context.Context, id string) (domain.Announcement, error)
	ConfirmRead(ctx context.Context, announcementID, userID string) error
	ReadStatus(ctx context.Context, announcementID string) ([]domain.AnnouncementReadStatus, error)
}
