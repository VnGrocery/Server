package domain

import "time"

// SpecItem is one row of a product's specification table.
//
// A slice rather than a map: neither Firestore nor Mongo preserves the key
// order of a map, and the order of these rows is one the seller chose.
type SpecItem struct {
	Key   string `firestore:"key"`
	Value string `firestore:"value"`
}

const (
	DescBlockHeading   = "heading"
	DescBlockParagraph = "paragraph"
	DescBlockBullets   = "bullets"
	DescBlockImage     = "image"
)

// DescBlock is one block of the long-form description.
//
// Structured blocks rather than a rich-text HTML blob: the blob would have to
// be sanitised before it reached a buyer's screen, and - the reason that
// decides it here - two versions of one cannot be diffed, so the change log
// could not show what the seller actually altered.
type DescBlock struct {
	Type    string   `firestore:"type"`
	Text    string   `firestore:"text"`
	Items   []string `firestore:"items"`
	CID     string   `firestore:"cid"`
	Caption string   `firestore:"caption"`
}

type Product struct {
	ProductID   string   `firestore:"productId"`
	ShopID      string   `firestore:"shopId"`
	OwnerUserID string   `firestore:"ownerUserId"`
	Name        string   `firestore:"name"`
	Description string   `firestore:"description"`
	Category    string   `firestore:"category"`
	Tags        []string `firestore:"tags"`
	ImageURLs   []string `firestore:"imageUrls"`

	// Fields of Product, deliberately, rather than a collection of their own:
	// every mutation logs the whole struct as the signed payload, so whatever
	// sits here is covered by the same chain as the price. Held outside it,
	// this would be an unsigned channel a seller could rewrite without
	// breaking a hash.
	//
	// Description stays alongside DescBlocks rather than being replaced by it:
	// the products already in the database carry their text there.
	Specs      []SpecItem  `firestore:"specs"`
	DescBlocks []DescBlock `firestore:"descBlocks"`

	FreshnessNote     string     `firestore:"freshnessNote"`
	FreshnessScore    float64    `firestore:"freshnessScore"`
	Price             float64    `firestore:"price"`
	Currency          string     `firestore:"currency"`
	Status            string     `firestore:"status"`
	Version           int        `firestore:"version"`
	ModeratedByUserID string     `firestore:"moderatedByUserId"`
	ModerationNote    string     `firestore:"moderationNote"`
	ModeratedAt       *time.Time `firestore:"moderatedAt"`
	CreatedAt         time.Time  `firestore:"createdAt"`
	UpdatedAt         time.Time  `firestore:"updatedAt"`
}
