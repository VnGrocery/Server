package auth

import (
	"context"
	"errors"
	"testing"

	firebaseauth "firebase.google.com/go/v4/auth"
)

type tokenVerifierStub struct {
	verify func(ctx context.Context, idToken string) (*firebaseauth.Token, error)
}

func (s tokenVerifierStub) VerifyIDToken(ctx context.Context, idToken string) (*firebaseauth.Token, error) {
	return s.verify(ctx, idToken)
}

func TestVerifyMapsPrincipal(t *testing.T) {
	service := NewVerifier(tokenVerifierStub{
		verify: func(ctx context.Context, idToken string) (*firebaseauth.Token, error) {
			return &firebaseauth.Token{
				UID: "user-123",
				Claims: map[string]any{
					"email":        "seller@example.com",
					"phone_number": "0123456789",
				},
			}, nil
		},
	})

	principal, err := service.Verify(context.Background(), "good")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	if principal.UserID != "user-123" {
		t.Fatalf("expected user-123, got %s", principal.UserID)
	}
	if principal.Email != "seller@example.com" {
		t.Fatalf("expected email mapped")
	}
	if principal.PhoneNumber != "0123456789" {
		t.Fatalf("expected phone mapped")
	}
}

func TestVerifyReturnsUnauthorizedOnInvalidToken(t *testing.T) {
	service := NewVerifier(tokenVerifierStub{
		verify: func(ctx context.Context, idToken string) (*firebaseauth.Token, error) {
			return nil, ErrUnauthorized
		},
	})

	_, err := service.Verify(context.Background(), "bad")
	if !errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected unauthorized, got %v", err)
	}
}

func TestVerifyReturnsInternalError(t *testing.T) {
	service := NewVerifier(tokenVerifierStub{
		verify: func(ctx context.Context, idToken string) (*firebaseauth.Token, error) {
			return nil, errors.New("firebase down")
		},
	})

	_, err := service.Verify(context.Background(), "bad")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if errors.Is(err, ErrUnauthorized) {
		t.Fatalf("expected internal error, got unauthorized")
	}
}
