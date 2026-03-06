package domain

import "time"

type ShopReview struct {
	ReviewID       string    `firestore:"reviewId"`
	ShopID         string    `firestore:"shopId"`
	ReviewerUserID string    `firestore:"reviewerUserId"`
	Rating         int       `firestore:"rating"`
	Comment        string    `firestore:"comment"`
	Status         string    `firestore:"status"`
	CreatedAt      time.Time `firestore:"createdAt"`
	UpdatedAt      time.Time `firestore:"updatedAt"`
}
