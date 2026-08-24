package dto

import "time"

// EngagementResponse carries the totals with the proof beside them: a follower
// count with no anchor is just a number the server is asserting.
type EngagementResponse struct {
	TargetType string `json:"targetType"`
	TargetID   string `json:"targetId"`

	Follows int `json:"follows"`
	Likes   int `json:"likes"`
	Loves   int `json:"loves"`

	// Mine is which of the marks the caller left, so the app can draw its own
	// buttons filled in without guessing from the totals.
	Mine []string `json:"mine"`

	DataHash         string     `json:"dataHash,omitempty"`
	ChainTxHash      string     `json:"chainTxHash,omitempty"`
	ChainBlockNumber int64      `json:"chainBlockNumber,omitempty"`
	ChainAnchorTime  *time.Time `json:"chainAnchorTime,omitempty"`
	AnchorStatus     string     `json:"anchorStatus,omitempty"`
}

type ToggleEngagementRequest struct {
	TargetType string `json:"targetType"`
	TargetID   string `json:"targetId"`
	Kind       string `json:"kind"`
}
