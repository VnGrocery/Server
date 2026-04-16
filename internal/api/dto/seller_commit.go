package dto

import "time"

type SellerCommitRequest struct {
	ShopID     string  `json:"shopId"`
	ProductID  string  `json:"productId"`
	BundleID   string  `json:"bundleId"`
	Score      float64 `json:"score"`
	Category   string  `json:"category"`
	Confidence float64 `json:"confidence"`
	ImageHash  string  `json:"imageHash"`
	ImageCID   string  `json:"imageCid"`
}

type SellerCommitResponse struct {
	PledgeID          string     `json:"pledgeId"`
	ShopID            string     `json:"shopId"`
	ProductID         string     `json:"productId,omitempty"`
	BundleID          string     `json:"bundleId"`
	CreatedByUserID   string     `json:"createdByUserId"`
	Status            string     `json:"status"`
	Score             float64    `json:"score"`
	Category          string     `json:"category"`
	Confidence        float64    `json:"confidence"`
	ImageHash         string     `json:"imageHash"`
	ImageCID          string     `json:"imageCid,omitempty"`
	DataHash          string     `json:"dataHash"`
	ChainTxHash       string     `json:"chainTxHash,omitempty"`
	ChainBlockNumber  int64      `json:"chainBlockNumber,omitempty"`
	ChainAnchorStatus string     `json:"chainAnchorStatus"`
	ChainAnchorTime   *time.Time `json:"chainAnchorTime,omitempty"`
	IntegrityStatus   string     `json:"integrityStatus"`
	QRVersion         string     `json:"qrVersion,omitempty"`
	BundleToken       string     `json:"bundleToken,omitempty"`
	BundleTokenExp    *time.Time `json:"bundleTokenExpiresAt,omitempty"`
	CommittedAt       time.Time  `json:"committedAt"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

type BundleTokenResponse struct {
	QRVersion            string     `json:"qrVersion,omitempty"`
	BundleToken          string     `json:"bundleToken"`
	BundleTokenExpiresAt *time.Time `json:"bundleTokenExpiresAt,omitempty"`
}

type PledgeHistoryResponse struct {
	Items []PledgeResponse `json:"items"`
}

type PledgeResponse struct {
	PledgeID          string     `json:"pledgeId"`
	ShopID            string     `json:"shopId"`
	ProductID         string     `json:"productId,omitempty"`
	BundleID          string     `json:"bundleId"`
	CreatedByUserID   string     `json:"createdByUserId"`
	Status            string     `json:"status"`
	Version           int        `json:"version"`
	Score             float64    `json:"score"`
	Category          string     `json:"category"`
	Confidence        float64    `json:"confidence"`
	ImageHash         string     `json:"imageHash"`
	ImageCID          string     `json:"imageCid,omitempty"`
	DataHash          string     `json:"dataHash"`
	ChainTxHash       string     `json:"chainTxHash,omitempty"`
	ChainBlockNumber  int64      `json:"chainBlockNumber,omitempty"`
	ChainAnchorStatus string     `json:"chainAnchorStatus"`
	ChainAnchorTime   *time.Time `json:"chainAnchorTime,omitempty"`
	IntegrityStatus   string     `json:"integrityStatus"`
	CommittedAt       time.Time  `json:"committedAt"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

type PledgeIntegrityResponse struct {
	PledgeID          string     `json:"pledgeId"`
	ShopID            string     `json:"shopId"`
	DataHash          string     `json:"dataHash"`
	ProvidedDataHash  string     `json:"providedDataHash,omitempty"`
	ChainTxHash       string     `json:"chainTxHash,omitempty"`
	ChainBlockNumber  int64      `json:"chainBlockNumber,omitempty"`
	ChainAnchorStatus string     `json:"chainAnchorStatus"`
	ChainAnchorTime   *time.Time `json:"chainAnchorTime,omitempty"`
	IntegrityStatus   string     `json:"integrityStatus"`
	OnChainMatch      bool       `json:"onChainMatch"`
	ProvidedHashMatch bool       `json:"providedHashMatch"`
	OnChainDataHash   string     `json:"onChainDataHash,omitempty"`
	OnChainVersion    int        `json:"onChainVersion,omitempty"`
	OnChainTimestamp  *time.Time `json:"onChainTimestamp,omitempty"`
	OnChainPresent    bool       `json:"onChainPresent"`
	MismatchReason    string     `json:"mismatchReason,omitempty"`
	LastCheckedAt     *time.Time `json:"lastCheckedAt,omitempty"`
	CanReanchor       bool       `json:"canReanchor"`
	CanRevoke         bool       `json:"canRevoke"`
}

type PledgeProofBundleResponse struct {
	PledgeID           string                  `json:"pledgeId"`
	ShopID             string                  `json:"shopId"`
	ProductID          string                  `json:"productId,omitempty"`
	BundleID           string                  `json:"bundleId"`
	Score              float64                 `json:"score"`
	Category           string                  `json:"category"`
	Confidence         float64                 `json:"confidence"`
	CommittedAt        time.Time               `json:"committedAt"`
	ImageHash          string                  `json:"imageHash"`
	ImageCID           string                  `json:"imageCid,omitempty"`
	ProofStatus        string                  `json:"proofStatus"`
	ProofHeadline      string                  `json:"proofHeadline"`
	ProofSummary       string                  `json:"proofSummary"`
	RecommendedActions []string                `json:"recommendedActions"`
	Integrity          PledgeIntegrityResponse `json:"integrity"`
}

type ModeratePledgeIntegrityRequest struct {
	ExpectedVersion int `json:"expectedVersion"`
}
