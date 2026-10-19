package ports

import (
	"context"

	"github.com/galash-uptc/backend/internal/domain"
)

type RegistrationRepository interface {
	CreateRegistration(ctx context.Context, request domain.RegistrationRequest, tokenHash string) (domain.RegistrationRequest, error)
	GetRegistrationStatus(ctx context.Context, id, tokenHash, userID string) (domain.RegistrationRequest, error)
	ListRegistrations(ctx context.Context, status, from, to, seedbedID, responsibleID string) ([]domain.RegistrationRequest, error)
	GetRegistrationDetail(ctx context.Context, id string) (domain.RegistrationDetail, error)
	UpdateRegistrationStatus(ctx context.Context, id, status, observation, responsibleID string) (domain.RegistrationRequest, error)
}
