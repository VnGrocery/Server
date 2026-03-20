package domain

import "time"

type Product struct {
	ProductID         string     `firestore:"productId"`
	ShopID            string     `firestore:"shopId"`
	OwnerUserID       string     `firestore:"ownerUserId"`
	Name              string     `firestore:"name"`
	Description       string     `firestore:"description"`
	Category          string     `firestore:"category"`
	Tags              []string   `firestore:"tags"`
	ImageURLs         []string   `firestore:"imageUrls"`
	FreshnessNote     string     `firestore:"freshnessNote"`
	FreshnessScore    float64    `firestore:"freshnessScore"`
	Price             float64    `firestore:"price"`
	Currency          string     `firestore:"currency"`
	Status            string     `firestore:"status"`
	Version           int        `firestore:"version"`
	ModeratedByUserID string     `firestore:"moderatedByUserId"`
	ModerationNote    string     `firestore:"moderationNote"`
	ModeratedAt       *time.Time `firestore:"moderatedAt"`
	CreatedAt         time.Time  `firestore:"createdAt"`
	UpdatedAt         time.Time  `firestore:"updatedAt"`
}
