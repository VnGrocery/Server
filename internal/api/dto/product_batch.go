package dto

import "time"

type CreateProductBatchRequest struct {
	BatchCode        string     `json:"batchCode"`
	OriginName       string     `json:"originName"`
	OriginAddress    string     `json:"originAddress"`
	SupplierName     string     `json:"supplierName"`
	SlaughteredAt    *time.Time `json:"slaughteredAt"`
	PackedAt         *time.Time `json:"packedAt"`
	ReceivedAt       *time.Time `json:"receivedAt"`
	ExpiresAt        *time.Time `json:"expiresAt"`
	Quantity         float64    `json:"quantity"`
	QuantityUnit     string     `json:"quantityUnit"`
	StorageTempMin   float64    `json:"storageTempMin"`
	StorageTempMax   float64    `json:"storageTempMax"`
	CurrentFreshness float64    `json:"currentFreshness"` // accepts 0-100 percent; 0-10 scores are normalized to percent
	CurrentCategory  string     `json:"currentCategory"`
	Status           string     `json:"status"`
}

type UpdateProductBatchRequest struct {
	ExpectedVersion  int        `json:"expectedVersion"`
	BatchCode        string     `json:"batchCode"`
	OriginName       string     `json:"originName"`
	OriginAddress    string     `json:"originAddress"`
	SupplierName     string     `json:"supplierName"`
	SlaughteredAt    *time.Time `json:"slaughteredAt"`
	PackedAt         *time.Time `json:"packedAt"`
	ReceivedAt       *time.Time `json:"receivedAt"`
	ExpiresAt        *time.Time `json:"expiresAt"`
	Quantity         float64    `json:"quantity"`
	QuantityUnit     string     `json:"quantityUnit"`
	StorageTempMin   float64    `json:"storageTempMin"`
	StorageTempMax   float64    `json:"storageTempMax"`
	CurrentFreshness float64    `json:"currentFreshness"` // accepts 0-100 percent; 0-10 scores are normalized to percent
	CurrentCategory  string     `json:"currentCategory"`
	Status           string     `json:"status"`
}

type ProductBatchResponse struct {
	BatchID          string     `json:"batchId"`
	ProductID        string     `json:"productId"`
	ShopID           string     `json:"shopId"`
	OwnerUserID      string     `json:"ownerUserId"`
	BatchCode        string     `json:"batchCode"`
	OriginName       string     `json:"originName"`
	OriginAddress    string     `json:"originAddress"`
	SupplierName     string     `json:"supplierName"`
	SlaughteredAt    *time.Time `json:"slaughteredAt,omitempty"`
	PackedAt         *time.Time `json:"packedAt,omitempty"`
	ReceivedAt       *time.Time `json:"receivedAt,omitempty"`
	ExpiresAt        *time.Time `json:"expiresAt,omitempty"`
	Quantity         float64    `json:"quantity"`
	QuantityUnit     string     `json:"quantityUnit"`
	StorageTempMin   float64    `json:"storageTempMin"`
	StorageTempMax   float64    `json:"storageTempMax"`
	CurrentFreshness float64    `json:"currentFreshness"` // percent, 0-100
	CurrentCategory  string     `json:"currentCategory"`
	Status           string     `json:"status"`
	Version          int        `json:"version"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

type ProductBatchListResponse struct {
	Items []ProductBatchResponse `json:"items"`
}
