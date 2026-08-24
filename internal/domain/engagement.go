package domain

import "time"

// Engagement kinds. A shop is followed; a product is liked or loved.
const (
	EngagementFollow = "follow"
	EngagementLike   = "like"
	EngagementLove   = "love"

	EngagementTargetShop    = "shop"
	EngagementTargetProduct = "product"
)

// Engagement is one person's mark on a shop or a product. There is at most one
// row per person per target per kind, so tapping twice takes the mark back
// rather than counting it again.
type Engagement struct {
	EngagementID string    `firestore:"engagementId"`
	UserID       string    `firestore:"userId"`
	TargetType   string    `firestore:"targetType"`
	TargetID     string    `firestore:"targetId"`
	Kind         string    `firestore:"kind"`
	CreatedAt    time.Time `firestore:"createdAt"`
}

// EngagementCount is the running total for one target, and the record that
// actually goes on chain.
//
// The individual taps are not anchored: one transaction per heart would cost
// more than the fact is worth, and would still only prove the same thing this
// record proves - that at a block time, the shop had this many followers and
// the product this many hearts, and nobody has quietly rewritten the figure
// since.
type EngagementCount struct {
	CountID    string `firestore:"countId"`
	TargetType string `firestore:"targetType"`
	TargetID   string `firestore:"targetId"`

	Follows int `firestore:"follows"`
	Likes   int `firestore:"likes"`
	Loves   int `firestore:"loves"`

	DataHash                 string     `firestore:"dataHash"`
	ChainTxHash              string     `firestore:"chainTxHash"`
	ChainBlockNumber         int64      `firestore:"chainBlockNumber"`
	ChainAnchorStatus        string     `firestore:"chainAnchorStatus"`
	ChainAnchorTime          *time.Time `firestore:"chainAnchorTime"`
	ChainAnchorAttempts      int        `firestore:"chainAnchorAttempts"`
	ChainAnchorNextAttemptAt *time.Time `firestore:"chainAnchorNextAttemptAt"`
	ChainAnchorLastError     string     `firestore:"chainAnchorLastError"`
	IntegrityStatus          string     `firestore:"integrityStatus"`

	Version   int       `firestore:"version"`
	CreatedAt time.Time `firestore:"createdAt"`
	UpdatedAt time.Time `firestore:"updatedAt"`
}
