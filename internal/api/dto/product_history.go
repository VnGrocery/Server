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
	Reason      string    `json:"reason,omitempty"`

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

// MarketPriceResponse is what every shop selling the same product charges.
//
// Two shops are treated as selling the same product when the whole name and the
// category match once folded, so different pack sizes are never averaged into
// one misleading number.
type MarketPriceResponse struct {
	CatalogKey string `json:"catalogKey"`

	// ShopCount includes this shop. One means nobody else sells it and there is
	// nothing to compare against.
	ShopCount int `json:"shopCount"`

	CurrentAverage float64 `json:"currentAverage"`
	CurrentLowest  float64 `json:"currentLowest"`
	CurrentHighest float64 `json:"currentHighest"`

	// History is the average price in effect across those shops over the same
	// window as the product's own series.
	History []PricePointResponse `json:"history"`
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

	// Market is omitted when no other shop sells the same product, so the
	// client shows no comparison rather than one against itself.
	Market *MarketPriceResponse `json:"market,omitempty"`
}
