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
	PayloadJSON     string    `json:"payloadJson"`
	PublicKey       string    `json:"publicKey"`
	KeyAlgorithm    string    `json:"keyAlgorithm"`
	Signature       string    `json:"signature"`
	ContentSHA256   string    `json:"contentSha256"`
	CreatedAt       time.Time `json:"createdAt"`
}

type EventLogListResponse struct {
	Items      []EventLogResponse  `json:"items"`
	Pagination PaginationResponse `json:"pagination"`
}

type PaginationResponse struct {
	Page       int `json:"page"`
	PageSize   int `json:"pageSize"`
	TotalItems int `json:"totalItems"`
	TotalPages int `json:"totalPages"`
}
