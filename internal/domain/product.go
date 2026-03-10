package domain

import "time"

type Product struct {
	ProductID   string    `firestore:"productId"`
	ShopID      string    `firestore:"shopId"`
	OwnerUserID string    `firestore:"ownerUserId"`
	Name        string    `firestore:"name"`
	Description string    `firestore:"description"`
	Price       float64   `firestore:"price"`
	Currency    string    `firestore:"currency"`
	Status      string    `firestore:"status"`
	Version     int       `firestore:"version"`
	CreatedAt   time.Time `firestore:"createdAt"`
	UpdatedAt   time.Time `firestore:"updatedAt"`
}
