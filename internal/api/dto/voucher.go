package dto

import "time"

type VoucherResponse struct {
	VoucherID     string    `json:"voucherId"`
	ShopID        string    `json:"shopId"`
	Code          string    `json:"code"`
	Title         string    `json:"title"`
	DiscountValue int       `json:"discountValue"`
	IsPercent     bool      `json:"isPercent"`
	MinSpend      int       `json:"minSpend"`
	ExpiresAt     time.Time `json:"expiresAt"`
	Active        bool      `json:"active"`
	Manual        bool      `json:"manual"`
	Note          string    `json:"note"`
	CodeFormat    string    `json:"codeFormat"`
}

type CreateVoucherRequest struct {
	Code          string    `json:"code"`
	Title         string    `json:"title"`
	DiscountValue int       `json:"discountValue"`
	IsPercent     bool      `json:"isPercent"`
	MinSpend      int       `json:"minSpend"`
	ExpiresAt     time.Time `json:"expiresAt"`
	Note          string    `json:"note"`
	CodeFormat    string    `json:"codeFormat"`
}

type ManualVoucherRequest struct {
	ShopID     string    `json:"shopId"`
	Code       string    `json:"code"`
	Title      string    `json:"title"`
	Note       string    `json:"note"`
	CodeFormat string    `json:"codeFormat"`
	ExpiresAt  time.Time `json:"expiresAt"`
}

type SaveVoucherRequest struct {
	VoucherID string `json:"voucherId"`
}
type CheckVoucherRequest struct {
	Code       string `json:"code"`
	ShopID     string `json:"shopId"`
	OrderValue int    `json:"orderValue"`
}

type VoucherCheckResponse struct {
	Voucher        *VoucherResponse `json:"voucher,omitempty"`
	Valid          bool             `json:"valid"`
	Message        string           `json:"message"`
	DiscountAmount int              `json:"discountAmount"`
	FinalPrice     int              `json:"finalPrice"`
}

type UserVoucherResponse struct {
	UserVoucherID string          `json:"userVoucherId"`
	VoucherID     string          `json:"voucherId"`
	Used          bool            `json:"used"`
	UsedAt        *time.Time      `json:"usedAt,omitempty"`
	Voucher       VoucherResponse `json:"voucher"`
}

type VoucherListResponse struct {
	Items []VoucherResponse `json:"items"`
}
type UserVoucherListResponse struct {
	Items []UserVoucherResponse `json:"items"`
}
