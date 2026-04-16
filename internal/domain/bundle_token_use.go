package domain

import "time"

type BundleTokenUse struct {
	UseID       string    `firestore:"useId"`
	NonceHash   string    `firestore:"nonceHash"`
	ShopID      string    `firestore:"shopId"`
	ProductID   string    `firestore:"productId"`
	BundleID    string    `firestore:"bundleId"`
	PledgeID    string    `firestore:"pledgeId"`
	BuyerUserID string    `firestore:"buyerUserId"`
	UsedAt      time.Time `firestore:"usedAt"`
	ExpiresAt   time.Time `firestore:"expiresAt"`
}
