package dto

import "time"

type UpsertProductRequest struct {
	ProductID       string   `json:"productId"`
	ExpectedVersion int      `json:"expectedVersion"`
	Name            string   `json:"name"`
	Description     string   `json:"description"`
	Category        string   `json:"category"`
	Tags            []string `json:"tags"`
	ImageURLs       []string `json:"imageUrls"`
	FreshnessNote   string   `json:"freshnessNote"`
	FreshnessScore  float64  `json:"freshnessScore"`
	Price           float64  `json:"price"`
	Currency        string   `json:"currency"`
	Status          string   `json:"status"`
}

type BulkUpsertProductRequest struct {
	Items []UpsertProductRequest `json:"items"`
}

type ModerateProductRequest struct {
	ExpectedVersion int    `json:"expectedVersion"`
	Status          string `json:"status"`
	ModerationNote  string `json:"moderationNote"`
}

type ProductResponse struct {
	ProductID         string     `json:"productId"`
	ShopID            string     `json:"shopId"`
	OwnerUserID       string     `json:"ownerUserId"`
	Name              string     `json:"name"`
	Description       string     `json:"description"`
	Category          string     `json:"category"`
	Tags              []string   `json:"tags"`
	ImageURLs         []string   `json:"imageUrls"`
	FreshnessNote     string     `json:"freshnessNote"`
	FreshnessScore    float64    `json:"freshnessScore"`
	Price             float64    `json:"price"`
	Currency          string     `json:"currency"`
	Status            string     `json:"status"`
	Version           int        `json:"version"`
	ModeratedByUserID string     `json:"moderatedByUserId,omitempty"`
	ModerationNote    string     `json:"moderationNote,omitempty"`
	ModeratedAt       *time.Time `json:"moderatedAt,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

type ProductListResponse struct {
	Items []ProductResponse `json:"items"`
}

type CreateProductFreshnessReportRequest struct {
	Score      float64 `json:"score"`
	Category   string  `json:"category"`
	Confidence float64 `json:"confidence"`
	Comment    string  `json:"comment"`
	ImageHash  string  `json:"imageHash"`
}

type ModerateProductFreshnessReportRequest struct {
	ExpectedVersion int    `json:"expectedVersion"`
	Status          string `json:"status"`
	ModerationNote  string `json:"moderationNote"`
}

type ProductFreshnessReportResponse struct {
	ReportID          string     `json:"reportId"`
	ProductID         string     `json:"productId"`
	ShopID            string     `json:"shopId"`
	ReporterUserID    string     `json:"reporterUserId"`
	Status            string     `json:"status"`
	Version           int        `json:"version"`
	Score             float64    `json:"score"`
	Category          string     `json:"category"`
	Confidence        float64    `json:"confidence"`
	Comment           string     `json:"comment"`
	ImageHash         string     `json:"imageHash"`
	ModeratedByUserID string     `json:"moderatedByUserId,omitempty"`
	ModerationNote    string     `json:"moderationNote,omitempty"`
	ModeratedAt       *time.Time `json:"moderatedAt,omitempty"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

type ProductFreshnessReportListResponse struct {
	Items []ProductFreshnessReportResponse `json:"items"`
}
