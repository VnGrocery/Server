package domain

import "time"

type Shop struct {
	ShopID      string    `firestore:"shopId"`
	OwnerUserID string    `firestore:"ownerUserId"`
	Name        string    `firestore:"name"`
	Description string    `firestore:"description"`
	Address     string    `firestore:"address"`
	Latitude    float64   `firestore:"latitude"`
	Longitude   float64   `firestore:"longitude"`
	Status      string    `firestore:"status"`
	CreatedAt   time.Time `firestore:"createdAt"`
	UpdatedAt   time.Time `firestore:"updatedAt"`
}
