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

// Verify checks the signature, expiry and revocation state of the Firebase ID
// token. ID tokens stay valid for up to an hour after the user is disabled or
// their sessions are revoked, so the revocation check is required here.
func (v *Verifier) Verify(ctx context.Context, token string) (domain.Identity, error) {
	firebaseToken, err := v.client.VerifyIDTokenAndCheckRevoked(ctx, token)
	if err != nil {
		return domain.Identity{}, err
	}
	name, _ := firebaseToken.Claims["name"].(string)
	email, _ := firebaseToken.Claims["email"].(string)
	return domain.Identity{UID: firebaseToken.UID, Name: name, Email: email}, nil
}
