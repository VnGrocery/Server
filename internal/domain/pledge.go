package domain

import "time"

type Pledge struct {
	PledgeID          string     `firestore:"pledgeId"`
	ShopID            string     `firestore:"shopId"`
	ProductID         string     `firestore:"productId"`
	CreatedByUserID   string     `firestore:"createdByUserId"`
	Status            string     `firestore:"status"`
	Version           int        `firestore:"version"`
	Score             float64    `firestore:"score"`
	Category          string     `firestore:"category"`
	Confidence        float64    `firestore:"confidence"`
	ImageHash         string     `firestore:"imageHash"`
	ImageCID          string     `firestore:"imageCid"`
	DataHash          string     `firestore:"dataHash"`
	ChainTxHash       string     `firestore:"chainTxHash"`
	ChainBlockNumber  int64      `firestore:"chainBlockNumber"`
	ChainAnchorStatus string     `firestore:"chainAnchorStatus"`
	ChainAnchorTime   *time.Time `firestore:"chainAnchorTime"`
	IntegrityStatus   string     `firestore:"integrityStatus"`
	CreatedAt         time.Time  `firestore:"createdAt"`
	UpdatedAt         time.Time  `firestore:"updatedAt"`
}
