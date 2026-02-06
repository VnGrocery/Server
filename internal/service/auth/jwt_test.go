package auth

import (
	"testing"
	"time"
)

func TestJWTServiceIssuesAndVerifiesToken(t *testing.T) {
	service := NewJWTService("test-secret", "vngrocery")

	token, err := service.IssueToken(Principal{UserID: "user-1", Email: "u@example.com"}, 2*time.Hour)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	principal, err := service.Verify(nil, token)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if principal.UserID != "user-1" {
		t.Fatalf("unexpected user id: %s", principal.UserID)
	}
	if principal.Email != "u@example.com" {
		t.Fatalf("unexpected email: %s", principal.Email)
	}
}

func TestJWTServiceRejectsInvalidToken(t *testing.T) {
	service := NewJWTService("test-secret", "vngrocery")

	_, err := service.Verify(nil, "not-a-token")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
