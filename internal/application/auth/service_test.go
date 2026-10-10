package auth

import (
	"context"
	"testing"

	"github.com/galash-uptc/backend/internal/domain"
)

type identityStub struct{ identity domain.Identity }

func (s identityStub) Verify(context.Context, string) (domain.Identity, error) {
	return s.identity, nil
}

type userStub struct{ user domain.User }

func (s *userStub) Upsert(_ context.Context, user domain.User) (string, error) {
	s.user = user
	return "user-id", nil
}

func TestRegisterPersistsRequestedIdentityFields(t *testing.T) {
	repository := &userStub{}
	service := NewService(identityStub{identity: domain.Identity{UID: "firebase-id", Name: "Ana", Email: "ana@example.com"}}, repository)

	result, err := service.Register(context.Background(), "token", RegisterCommand{
		Name: "Ana", LastName: "Pérez", DocumentType: "CC", DocumentNumber: "123", Email: "ana@example.com", Role: "estudiante",
	})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if result.ID != "user-id" || repository.user.Name != "Ana" || repository.user.LastName != "Pérez" || repository.user.DocumentType != "CC" || repository.user.DocumentNumber != "123" || repository.user.Email != "ana@example.com" || repository.user.Role != "estudiante" {
		t.Fatalf("unexpected registration result: %+v, user: %+v", result, repository.user)
	}
}
