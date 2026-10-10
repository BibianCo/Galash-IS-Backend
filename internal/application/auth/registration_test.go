package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/galash-uptc/backend/internal/domain"
)

type registrationIdentityVerifier struct {
	identity domain.Identity
	err      error
}

func (v registrationIdentityVerifier) Verify(context.Context, string) (domain.Identity, error) {
	return v.identity, v.err
}

func validRegisterCommand() RegisterCommand {
	return RegisterCommand{
		Name: "Ana", LastName: "Pérez", DocumentType: "CC", DocumentNumber: "123",
		Email: "ana@example.com", Role: "estudiante",
	}
}

func TestRegisterRejectsInvalidFirebaseToken(t *testing.T) {
	service := NewService(
		registrationIdentityVerifier{err: errors.New("invalid token")},
		&userStub{},
	)

	_, err := service.Register(context.Background(), "bad-token", validRegisterCommand())
	if !errors.Is(err, domain.ErrInvalidIdentity) {
		t.Fatalf("Register() error = %v, want ErrInvalidIdentity", err)
	}
}

func TestRegisterRejectsInvalidInput(t *testing.T) {
	identity := registrationIdentityVerifier{identity: domain.Identity{UID: "firebase-id", Email: "ana@example.com"}}
	tests := []struct {
		name    string
		mutate  func(*RegisterCommand)
		wantErr error
	}{
		{name: "missing name", mutate: func(command *RegisterCommand) { command.Name = "" }, wantErr: domain.ErrIncompleteIdentity},
		{name: "missing last name", mutate: func(command *RegisterCommand) { command.LastName = "" }, wantErr: domain.ErrIncompleteIdentity},
		{name: "missing document type", mutate: func(command *RegisterCommand) { command.DocumentType = "" }, wantErr: domain.ErrIncompleteIdentity},
		{name: "missing document number", mutate: func(command *RegisterCommand) { command.DocumentNumber = "" }, wantErr: domain.ErrIncompleteIdentity},
		{name: "missing email", mutate: func(command *RegisterCommand) { command.Email = "" }, wantErr: domain.ErrIncompleteIdentity},
		{name: "email mismatch", mutate: func(command *RegisterCommand) { command.Email = "other@example.com" }, wantErr: domain.ErrIncompleteIdentity},
		{name: "invalid role", mutate: func(command *RegisterCommand) { command.Role = "administrador" }, wantErr: domain.ErrIncompleteIdentity},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			command := validRegisterCommand()
			test.mutate(&command)
			service := NewService(identity, &userStub{})

			_, err := service.Register(context.Background(), "firebase-token", command)
			if !errors.Is(err, test.wantErr) {
				t.Fatalf("Register() error = %v, want %v", err, test.wantErr)
			}
		})
	}
}
