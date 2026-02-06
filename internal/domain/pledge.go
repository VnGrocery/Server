package domain

import "time"

type Pledge struct {
	PledgeID        string    `firestore:"pledgeId"`
	ShopID          string    `firestore:"shopId"`
	CreatedByUserID string    `firestore:"createdByUserId"`
	Status          string    `firestore:"status"`
	Score           float64   `firestore:"score"`
	Category        string    `firestore:"category"`
	Confidence      float64   `firestore:"confidence"`
	CreatedAt       time.Time `firestore:"createdAt"`
	UpdatedAt       time.Time `firestore:"updatedAt"`
}
