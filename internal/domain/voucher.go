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
	Manual        bool      `firestore:"manual"`
	Note          string    `firestore:"note"`
	CodeFormat    string    `firestore:"codeFormat"`
	CreatedAt     time.Time `firestore:"createdAt"`
	UpdatedAt     time.Time `firestore:"updatedAt"`
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
