package domain

import "time"

type ShopReview struct {
	ReviewID       string    `firestore:"reviewId"`
	ShopID         string    `firestore:"shopId"`
	ReviewerUserID string    `firestore:"reviewerUserId"`
	Rating         int       `firestore:"rating"`
	Comment        string    `firestore:"comment"`
	ImageURLs      []string  `firestore:"imageUrls"`
	Status         string    `firestore:"status"`
	Version        int       `firestore:"version"`
	CreatedAt      time.Time `firestore:"createdAt"`
	UpdatedAt      time.Time `firestore:"updatedAt"`
}
