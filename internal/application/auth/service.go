package auth

import (
	"context"
	"errors"

	"github.com/galash-uptc/backend/internal/domain"
	"github.com/galash-uptc/backend/internal/ports"
)

type RegisterCommand struct {
	Name     string
	LastName string
}

type RegisterResult struct {
	ID          string `json:"id"`
	FirebaseUID string `json:"firebase_uid"`
}

type Service struct {
	identities ports.IdentityVerifier
	users      ports.UserRepository
}

func NewService(identities ports.IdentityVerifier, users ports.UserRepository) *Service {
	return &Service{identities: identities, users: users}
}

func (s *Service) Register(ctx context.Context, token string, command RegisterCommand) (RegisterResult, error) {
	identity, err := s.identities.Verify(ctx, token)
	if err != nil {
		return RegisterResult{}, errors.Join(domain.ErrInvalidIdentity, err)
	}

	name := command.Name
	if name == "" {
		name = identity.Name
	}
	if name == "" || identity.Email == "" {
		return RegisterResult{}, domain.ErrIncompleteIdentity
	}

	id, err := s.users.Upsert(ctx, domain.User{
		FirebaseUID: identity.UID,
		Name:        name,
		LastName:    command.LastName,
		Email:       identity.Email,
	})
	if err != nil {
		return RegisterResult{}, err
	}
	return RegisterResult{ID: id, FirebaseUID: identity.UID}, nil
}
