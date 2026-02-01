package domain

import "time"

type Pledge struct {
	PledgeID        string    `firestore:"pledgeId"`
	ShopID          string    `firestore:"shopId"`
	CreatedByUserID string    `firestore:"createdByUserId"`
	Status          string    `firestore:"status"`
	CreatedAt       time.Time `firestore:"createdAt"`
	UpdatedAt       time.Time `firestore:"updatedAt"`
}
