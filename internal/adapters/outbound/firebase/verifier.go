package firebase

import (
	"context"

	"firebase.google.com/go/v4/auth"
	"github.com/galash-uptc/backend/internal/domain"
)

type Verifier struct {
	client *auth.Client
}

func NewVerifier(client *auth.Client) *Verifier {
	return &Verifier{client: client}
}

func (v *Verifier) Verify(ctx context.Context, token string) (domain.Identity, error) {
	firebaseToken, err := v.client.VerifyIDToken(ctx, token)
	if err != nil {
		return domain.Identity{}, err
	}
	name, _ := firebaseToken.Claims["name"].(string)
	email, _ := firebaseToken.Claims["email"].(string)
	return domain.Identity{UID: firebaseToken.UID, Name: name, Email: email}, nil
}
