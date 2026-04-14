package dto

import "time"

type EventLogResponse struct {
	EventID         string    `json:"eventId"`
	ActorUserID     string    `json:"actorUserId"`
	ResourceType    string    `json:"resourceType"`
	ResourceID      string    `json:"resourceId"`
	ResourceVersion int       `json:"resourceVersion"`
	Action          string    `json:"action"`
	Status          string    `json:"status"`
	Sequence        int       `json:"sequence"`
	PreviousEventID string    `json:"previousEventId,omitempty"`
	OccurredAt      string    `json:"occurredAt,omitempty"`
	PayloadJSON     string    `json:"payloadJson"`
	PublicKey       string    `json:"publicKey"`
	KeyAlgorithm    string    `json:"keyAlgorithm"`
	Signature       string    `json:"signature"`
	ContentSHA256   string    `json:"contentSha256"`
	CreatedAt       time.Time `json:"createdAt"`
}

type EventLogListResponse struct {
	Items      []EventLogResponse `json:"items"`
	Pagination PaginationResponse `json:"pagination"`
}

type EventVerificationResponse struct {
	EventID              string `json:"eventId"`
	ResourceType         string `json:"resourceType"`
	ResourceID           string `json:"resourceId"`
	Sequence             int    `json:"sequence"`
	PreviousEventID      string `json:"previousEventId,omitempty"`
	ContentHashValid     bool   `json:"contentHashValid"`
	SignatureValid       bool   `json:"signatureValid"`
	ChainLinkValid       bool   `json:"chainLinkValid"`
	PreviousEventPresent bool   `json:"previousEventPresent"`
	Verified             bool   `json:"verified"`
}

type ResourceEventVerificationResponse struct {
	ResourceType string                      `json:"resourceType"`
	ResourceID   string                      `json:"resourceId"`
	EventCount   int                         `json:"eventCount"`
	Verified     bool                        `json:"verified"`
	Events       []EventVerificationResponse `json:"events"`
}

type PaginationResponse struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	TotalItems int `json:"totalItems"`
	TotalPages int `json:"totalPages"`
}
