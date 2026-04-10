package dto

type BuyerCheckResponse struct {
	CheckID          string   `json:"checkId,omitempty"`
	ShopID           string   `json:"shopId,omitempty"`
	ProductID        string   `json:"productId,omitempty"`
	Status           string   `json:"status,omitempty"`
	Version          int      `json:"version,omitempty"`
	PolicyVersion    string   `json:"policyVersion"`
	HasPledge        bool     `json:"hasPledge"`
	PledgeID         string   `json:"pledgeId,omitempty"`
	Trusted          bool     `json:"trusted"`
	Verdict          string   `json:"verdict"`
	PledgedScore     float64  `json:"pledgedScore,omitempty"`
	ActualScore      float64  `json:"actualScore"`
	ScoreDelta       float64  `json:"scoreDelta,omitempty"`
	ScoreDeltaAbs    float64  `json:"scoreDeltaAbs,omitempty"`
	PledgedCategory  string   `json:"pledgedCategory,omitempty"`
	ActualCategory   string   `json:"actualCategory"`
	ActualConfidence float64  `json:"actualConfidence"`
	CategoryMatch    bool     `json:"categoryMatch"`
	ImageHash        string   `json:"imageHash,omitempty"`
	ImageCID         string   `json:"imageCid,omitempty"`
	Reasons          []string `json:"reasons"`
}

type ModerateBuyerCheckRequest struct {
	ExpectedVersion int    `json:"expectedVersion"`
	Status          string `json:"status"`
	ModerationNote  string `json:"moderationNote"`
}
