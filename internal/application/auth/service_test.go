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

func TestRegisterUsesIdentityNameWhenCommandNameIsEmpty(t *testing.T) {
	repository := &userStub{}
	service := NewService(identityStub{identity: domain.Identity{UID: "firebase-id", Name: "Ana", Email: "ana@example.com"}}, repository)

	result, err := service.Register(context.Background(), "token", RegisterCommand{})
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if result.ID != "user-id" || repository.user.Name != "Ana" {
		t.Fatalf("unexpected registration result: %+v, user: %+v", result, repository.user)
	}
}
