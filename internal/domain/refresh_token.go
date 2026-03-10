package domain

import "time"

type RefreshToken struct {
	TokenID   string     `firestore:"tokenId"`
	UserID    string     `firestore:"userId"`
	TokenHash string     `firestore:"tokenHash"`
	Status    string     `firestore:"status"`
	ExpiresAt time.Time  `firestore:"expiresAt"`
	RevokedAt *time.Time `firestore:"revokedAt"`
	CreatedAt time.Time  `firestore:"createdAt"`
	UpdatedAt time.Time  `firestore:"updatedAt"`
}
