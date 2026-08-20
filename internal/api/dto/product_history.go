package dto

import "time"

// FieldChangeResponse is one field a change altered.
type FieldChangeResponse struct {
	Field  string `json:"field"`
	Before string `json:"before"`
	After  string `json:"after"`
}

// ProductHistoryEntryResponse is one recorded change, shaped like a commit: a
// hash of its own content, the hash before it, who made it and when.
type ProductHistoryEntryResponse struct {
	SHA         string `json:"sha"`
	ShortSHA    string `json:"shortSha"`
	PreviousSHA string `json:"previousSha,omitempty"`

	Sequence    int       `json:"sequence"`
	Action      string    `json:"action"`
	Status      string    `json:"status"`
	ActorUserID string    `json:"actorUserId"`
	ActorName   string    `json:"actorName,omitempty"`
	OccurredAt  time.Time `json:"occurredAt"`

	Verified         bool `json:"verified"`
	ContentHashValid bool `json:"contentHashValid"`
	SignatureValid   bool `json:"signatureValid"`
	ChainLinkValid   bool `json:"chainLinkValid"`

	PriceAfter *float64              `json:"priceAfter,omitempty"`
	Changes    []FieldChangeResponse `json:"changes"`
}

// PricePointResponse is the price in effect at a moment.
type PricePointResponse struct {
	At    time.Time `json:"at"`
	Price float64   `json:"price"`
}

// ProductHistoryResponse is a product's change history and the price series
// derived from it.
type ProductHistoryResponse struct {
	ProductID string `json:"productId"`

	// Newest first, the way a commit list reads.
	Entries []ProductHistoryEntryResponse `json:"entries"`

	// False when any entry failed verification, which means the record has been
	// altered since it was written.
	ChainVerified bool `json:"chainVerified"`

	PriceHistory []PricePointResponse `json:"priceHistory"`
	WindowDays   int                  `json:"windowDays"`
}
