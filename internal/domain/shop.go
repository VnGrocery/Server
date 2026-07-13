package domain

import "time"

type Shop struct {
	ShopID                   string     `firestore:"shopId"`
	OwnerUserID              string     `firestore:"ownerUserId"`
	Name                     string     `firestore:"name"`
	Description              string     `firestore:"description"`
	Address                  string     `firestore:"address"`
	Latitude                 float64    `firestore:"latitude"`
	Longitude                float64    `firestore:"longitude"`
	Status                   string     `firestore:"status"`
	DataHash                 string     `firestore:"dataHash"`
	ChainTxHash              string     `firestore:"chainTxHash"`
	ChainBlockNumber         int64      `firestore:"chainBlockNumber"`
	ChainAnchorStatus        string     `firestore:"chainAnchorStatus"`
	ChainAnchorTime          *time.Time `firestore:"chainAnchorTime"`
	ChainAnchorOperation     string     `firestore:"chainAnchorOperation"`
	ChainAnchorAttempts      int        `firestore:"chainAnchorAttempts"`
	ChainAnchorNextAttemptAt *time.Time `firestore:"chainAnchorNextAttemptAt"`
	ChainAnchorLastError     string     `firestore:"chainAnchorLastError"`
	IntegrityStatus          string     `firestore:"integrityStatus"`
	Version                  int        `firestore:"version"`
	ModeratedByUserID        string     `firestore:"moderatedByUserId"`
	ModerationNote           string     `firestore:"moderationNote"`
	ModeratedAt              *time.Time `firestore:"moderatedAt"`
	CreatedAt                time.Time  `firestore:"createdAt"`
	UpdatedAt                time.Time  `firestore:"updatedAt"`
}
