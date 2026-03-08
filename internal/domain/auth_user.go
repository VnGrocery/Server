package domain

import "time"

type AuthUser struct {
	UserID       string    `firestore:"userId"`
	EmailLower   string    `firestore:"emailLower"`
	PasswordHash string    `firestore:"passwordHash"`
	GoogleSub    string    `firestore:"googleSub"`
	Providers    []string  `firestore:"providers"`
	Status       string    `firestore:"status"`
	Version      int       `firestore:"version"`
	PublicKey    string    `firestore:"publicKey"`
	KeyAlgorithm string    `firestore:"keyAlgorithm"`
	VaultKeyPath string    `firestore:"vaultKeyPath"`
	CreatedAt    time.Time `firestore:"createdAt"`
	UpdatedAt    time.Time `firestore:"updatedAt"`
}
