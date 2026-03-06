package dto

import "time"

type UpsertShopRequest struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Address     string  `json:"address"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

type ModerateShopRequest struct {
	Status         string `json:"status"`
	ModerationNote string `json:"moderationNote"`
}

type ShopTrustSummaryResponse struct {
	HasPledges         bool       `json:"hasPledges"`
	PledgeCount        int        `json:"pledgeCount"`
	LatestPledgeID     string     `json:"latestPledgeId,omitempty"`
	LatestPledgeStatus string     `json:"latestPledgeStatus,omitempty"`
	LatestScore        float64    `json:"latestScore,omitempty"`
	LatestCategory     string     `json:"latestCategory,omitempty"`
	LatestConfidence   float64    `json:"latestConfidence,omitempty"`
	LastCommittedAt    *time.Time `json:"lastCommittedAt,omitempty"`
}

type ShopRatingSummaryResponse struct {
	RatingCount   int     `json:"ratingCount"`
	AverageRating float64 `json:"averageRating"`
}

type CreateShopReviewRequest struct {
	Rating  int    `json:"rating"`
	Comment string `json:"comment"`
}

type ShopReviewResponse struct {
	ReviewID       string    `json:"reviewId"`
	ShopID         string    `json:"shopId"`
	ReviewerUserID string    `json:"reviewerUserId"`
	Rating         int       `json:"rating"`
	Comment        string    `json:"comment"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"createdAt"`
	UpdatedAt      time.Time `json:"updatedAt"`
}

type ShopResponse struct {
	ShopID            string                    `json:"shopId"`
	OwnerUserID       string                    `json:"ownerUserId"`
	Name              string                    `json:"name"`
	Description       string                    `json:"description"`
	Address           string                    `json:"address"`
	Latitude          float64                   `json:"latitude"`
	Longitude         float64                   `json:"longitude"`
	Status            string                    `json:"status"`
	ModeratedByUserID string                    `json:"moderatedByUserId,omitempty"`
	ModerationNote    string                    `json:"moderationNote,omitempty"`
	ModeratedAt       *time.Time                `json:"moderatedAt,omitempty"`
	TrustSummary      ShopTrustSummaryResponse  `json:"trustSummary"`
	RatingSummary     ShopRatingSummaryResponse `json:"ratingSummary"`
	CreatedAt         time.Time                 `json:"createdAt"`
	UpdatedAt         time.Time                 `json:"updatedAt"`
}

type ShopListResponse struct {
	Items    []ShopResponse `json:"items"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
	Total    int            `json:"total"`
	HasNext  bool           `json:"hasNext"`
}
