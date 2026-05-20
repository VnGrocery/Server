package dto

import "time"

type BuyerCheckResponse struct {
	CheckID           string     `json:"checkId,omitempty"`
	ShopID            string     `json:"shopId,omitempty"`
	ProductID         string     `json:"productId,omitempty"`
	BatchID           string     `json:"batchId,omitempty"`
	BundleID          string     `json:"bundleId,omitempty"`
	BuyerUserID       string     `json:"buyerUserId,omitempty"`
	Status            string     `json:"status,omitempty"`
	Version           int        `json:"version,omitempty"`
	PolicyVersion     string     `json:"policyVersion"`
	HasPledge         bool       `json:"hasPledge"`
	PledgeID          string     `json:"pledgeId,omitempty"`
	Trusted           bool       `json:"trusted"`
	Verdict           string     `json:"verdict"`
	PledgedScore      float64    `json:"pledgedScore,omitempty"`
	ActualScore       float64    `json:"actualScore"`
	ScoreDelta        float64    `json:"scoreDelta,omitempty"`
	ScoreDeltaAbs     float64    `json:"scoreDeltaAbs,omitempty"`
	PledgedCategory   string     `json:"pledgedCategory,omitempty"`
	ActualCategory    string     `json:"actualCategory"`
	ActualConfidence  float64    `json:"actualConfidence"`
	LocationStatus    string     `json:"locationStatus,omitempty"`
	CategoryMatch     bool       `json:"categoryMatch"`
	ImageHash         string     `json:"imageHash,omitempty"`
	ImageCID          string     `json:"imageCid,omitempty"`
	Reasons           []string   `json:"reasons"`
	ModeratedByUserID string     `json:"moderatedByUserId,omitempty"`
	ModerationNote    string     `json:"moderationNote,omitempty"`
	ModeratedAt       *time.Time `json:"moderatedAt,omitempty"`
	CreatedAt         time.Time  `json:"createdAt,omitempty"`
	UpdatedAt         time.Time  `json:"updatedAt,omitempty"`
}

type ModerateBuyerCheckRequest struct {
	ExpectedVersion int    `json:"expectedVersion"`
	Status          string `json:"status"`
	ModerationNote  string `json:"moderationNote"`
}

type BuyerCheckListResponse struct {
	Items      []BuyerCheckResponse `json:"items"`
	Pagination PaginationResponse   `json:"pagination"`
}
