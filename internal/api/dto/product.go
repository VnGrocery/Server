package dto

import "time"

type UpsertProductRequest struct {
	ExpectedVersion int     `json:"expectedVersion"`
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	Price           float64 `json:"price"`
	Currency        string  `json:"currency"`
}

type ProductResponse struct {
	ProductID   string    `json:"productId"`
	ShopID      string    `json:"shopId"`
	OwnerUserID string    `json:"ownerUserId"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Price       float64   `json:"price"`
	Currency    string    `json:"currency"`
	Status      string    `json:"status"`
	Version     int       `json:"version"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type ProductListResponse struct {
	Items []ProductResponse `json:"items"`
}
