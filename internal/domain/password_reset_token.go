package domain

import "time"

type PasswordResetToken struct {
	TokenID   string     `firestore:"tokenId"`
	UserID    string     `firestore:"userId"`
	TokenHash string     `firestore:"tokenHash"`
	Status    string     `firestore:"status"`
	ExpiresAt time.Time  `firestore:"expiresAt"`
	UsedAt    *time.Time `firestore:"usedAt"`
	CreatedAt time.Time  `firestore:"createdAt"`
	UpdatedAt time.Time  `firestore:"updatedAt"`
}
