package domain

import "time"

type AuthUser struct {
	UserID       string    `firestore:"userId"`
	EmailLower   string    `firestore:"emailLower"`
	PasswordHash string    `firestore:"passwordHash"`
	GoogleSub    string    `firestore:"googleSub"`
	Providers    []string  `firestore:"providers"`
	CreatedAt    time.Time `firestore:"createdAt"`
	UpdatedAt    time.Time `firestore:"updatedAt"`
}
