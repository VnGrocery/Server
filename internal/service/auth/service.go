package auth

import (
	"context"
	"errors"
	"fmt"

	firebaseauth "firebase.google.com/go/v4/auth"
)

var ErrUnauthorized = errors.New("missing or invalid authentication token")

type Principal struct {
	UserID      string `json:"userId"`
	Email       string `json:"email,omitempty"`
	PhoneNumber string `json:"phoneNumber,omitempty"`
}

type TokenVerifier interface {
	VerifyIDToken(ctx context.Context, idToken string) (*firebaseauth.Token, error)
}

type Verifier interface {
	Verify(ctx context.Context, token string) (Principal, error)
}

type Service struct {
	tokenVerifier TokenVerifier
}

func NewVerifier(tokenVerifier TokenVerifier) *Service {
	return &Service{tokenVerifier: tokenVerifier}
}

func (s *Service) Verify(ctx context.Context, token string) (Principal, error) {
	verifiedToken, err := s.tokenVerifier.VerifyIDToken(ctx, token)
	if err != nil {
		if errors.Is(err, ErrUnauthorized) || firebaseauth.IsIDTokenInvalid(err) || firebaseauth.IsIDTokenExpired(err) || firebaseauth.IsUserDisabled(err) {
			return Principal{}, ErrUnauthorized
		}

		return Principal{}, fmt.Errorf("failed to verify Firebase ID token: %w", err)
	}

	return Principal{
		UserID:      verifiedToken.UID,
		Email:       readStringClaim(verifiedToken.Claims, "email"),
		PhoneNumber: readStringClaim(verifiedToken.Claims, "phone_number"),
	}, nil
}

func readStringClaim(claims map[string]any, key string) string {
	value, ok := claims[key]
	if !ok {
		return ""
	}

	text, ok := value.(string)
	if !ok {
		return ""
	}

	return text
}
