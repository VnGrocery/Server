package domain

import "time"

type Shop struct {
	ShopID      string    `firestore:"shopId"`
	OwnerUserID string    `firestore:"ownerUserId"`
	Name        string    `firestore:"name"`
	Status      string    `firestore:"status"`
	CreatedAt   time.Time `firestore:"createdAt"`
	UpdatedAt   time.Time `firestore:"updatedAt"`
}
