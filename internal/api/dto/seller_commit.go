package dto

import "time"

type SellerCommitRequest struct {
	ShopID     string  `json:"shopId"`
	Score      float64 `json:"score"`
	Category   string  `json:"category"`
	Confidence float64 `json:"confidence"`
	ImageHash  string  `json:"imageHash"`
}

type SellerCommitResponse struct {
	PledgeID        string    `json:"pledgeId"`
	ShopID          string    `json:"shopId"`
	CreatedByUserID string    `json:"createdByUserId"`
	Status          string    `json:"status"`
	Score           float64   `json:"score"`
	Category        string    `json:"category"`
	Confidence      float64   `json:"confidence"`
	ImageHash       string    `json:"imageHash"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}
