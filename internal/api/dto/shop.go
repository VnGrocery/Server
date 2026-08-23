package dto

import "time"

type UpsertShopRequest struct {
	ExpectedVersion int `json:"expectedVersion"`

	// Why this change is being made. Required on update and delete; signed
	// into the event log with everything else.
	ChangeReason string `json:"changeReason"`

	Name        string  `json:"name"`
	Description string  `json:"description"`
	Address     string  `json:"address"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

type ModerateShopRequest struct {
	ExpectedVersion int    `json:"expectedVersion"`
	Status          string `json:"status"`
	ModerationNote  string `json:"moderationNote"`
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
	Score              float64    `json:"score"`
	Grade              string     `json:"grade"`
	FormulaVersion     string     `json:"formulaVersion"`
	PledgeScore        float64    `json:"pledgeScore"`
	ReviewScore        float64    `json:"reviewScore"`
	BuyerCheckScore    float64    `json:"buyerCheckScore"`
	ConsistencyScore   float64    `json:"consistencyScore"`
	RecencyScore       float64    `json:"recencyScore"`
	CoverageScore      float64    `json:"coverageScore"`
	BuyerCheckCount    int        `json:"buyerCheckCount"`
	TrustedCheckCount  int        `json:"trustedCheckCount"`
	HighRiskCheckCount int        `json:"highRiskCheckCount"`
	Reasons            []string   `json:"reasons"`
}

type ShopRatingSummaryResponse struct {
	RatingCount   int     `json:"ratingCount"`
	AverageRating float64 `json:"averageRating"`
}

type CreateShopReviewRequest struct {
	ExpectedVersion int      `json:"expectedVersion"`
	Rating          int      `json:"rating"`
	Comment         string   `json:"comment"`
	ImageURLs       []string `json:"imageUrls"`
}

type ShopReviewResponse struct {
	ReviewID       string `json:"reviewId"`
	ShopID         string `json:"shopId"`
	ReviewerUserID string `json:"reviewerUserId"`
	// Empty when the account has no display name; the client shows a generic
	// label rather than printing the raw user id.
	ReviewerName string    `json:"reviewerName"`
	Rating       int       `json:"rating"`
	Comment      string    `json:"comment"`
	ImageURLs    []string  `json:"imageUrls"`
	Status       string    `json:"status"`
	Version      int       `json:"version"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type ShopResponse struct {
	// DistanceKm from the point in a lat/lng/radiusKm query. Omitted when the
	// listing was not narrowed by location, so a client can tell "right here"
	// from "not measured".
	DistanceKm        *float64                  `json:"distanceKm,omitempty"`
	ShopID            string                    `json:"shopId"`
	OwnerUserID       string                    `json:"ownerUserId"`
	Version           int                       `json:"version"`
	Name              string                    `json:"name"`
	Description       string                    `json:"description"`
	Address           string                    `json:"address"`
	Latitude          float64                   `json:"latitude"`
	Longitude         float64                   `json:"longitude"`
	Status            string                    `json:"status"`
	DataHash          string                    `json:"dataHash,omitempty"`
	ChainTxHash       string                    `json:"chainTxHash,omitempty"`
	ChainBlockNumber  int64                     `json:"chainBlockNumber,omitempty"`
	ChainAnchorStatus string                    `json:"chainAnchorStatus,omitempty"`
	ChainAnchorTime   *time.Time                `json:"chainAnchorTime,omitempty"`
	IntegrityStatus   string                    `json:"integrityStatus,omitempty"`
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
