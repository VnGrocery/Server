package domain

import (
	"fmt"
	"strings"
	"time"
)

const (
	RoleAdmin = "admin"
	RoleUser  = "user"

	UserStatusActive    = "active"
	UserStatusSuspended = "suspended"
	UserStatusDeleted   = "deleted"
)

func NormalizeRole(role string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(role))
	switch normalized {
	case RoleAdmin, RoleUser:
		return normalized, nil
	default:
		return "", fmt.Errorf("unsupported role")
	}
}

func NormalizeUserStatus(status string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(status))
	switch normalized {
	case UserStatusActive, UserStatusSuspended, UserStatusDeleted:
		return normalized, nil
	default:
		return "", fmt.Errorf("unsupported user status")
	}
}

func IsActiveUser(user User) bool {
	return strings.EqualFold(strings.TrimSpace(user.Status), UserStatusActive)
}

type User struct {
	UserID      string    `firestore:"userId"`
	Email       string    `firestore:"email"`
	DisplayName string    `firestore:"displayName"`
	FirstName   string    `firestore:"firstName"`
	LastName    string    `firestore:"lastName"`
	Role        string    `firestore:"role"`
	Status      string    `firestore:"status"`
	Version     int       `firestore:"version"`
	CreatedAt   time.Time `firestore:"createdAt"`
	UpdatedAt   time.Time `firestore:"updatedAt"`
}
