package domain

import "time"

type Voucher struct {
	VoucherID     string    `firestore:"voucherId"`
	ShopID        string    `firestore:"shopId"`
	Code          string    `firestore:"code"`
	Title         string    `firestore:"title"`
	DiscountValue int       `firestore:"discountValue"`
	IsPercent     bool      `firestore:"isPercent"`
	MinSpend      int       `firestore:"minSpend"`
	ExpiresAt     time.Time `firestore:"expiresAt"`
	Active        bool      `firestore:"active"`

	// TotalQuantity is how many claims the shop is offering. Zero means the
	// offer is not rationed, which is what every voucher created before this
	// field existed carries.
	TotalQuantity int `firestore:"totalQuantity"`

	// ClaimedCount only ever moves up, and only through the atomic claim in
	// the repository: two buyers reaching for the last voucher at the same
	// moment must not both get it.
	ClaimedCount int       `firestore:"claimedCount"`
	Manual       bool      `firestore:"manual"`
	Note         string    `firestore:"note"`
	CodeFormat   string    `firestore:"codeFormat"`
	CreatedAt    time.Time `firestore:"createdAt"`
	UpdatedAt    time.Time `firestore:"updatedAt"`
}

type UserVoucher struct {
	UserVoucherID string     `firestore:"userVoucherId"`
	UserID        string     `firestore:"userId"`
	VoucherID     string     `firestore:"voucherId"`
	Used          bool       `firestore:"used"`
	UsedAt        *time.Time `firestore:"usedAt"`
	CreatedAt     time.Time  `firestore:"createdAt"`
	UpdatedAt     time.Time  `firestore:"updatedAt"`
}
