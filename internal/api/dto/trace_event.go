package dto

import "time"

type CreateTraceEventRequest struct {
	Type         string    `json:"type"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	LocationName string    `json:"locationName"`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	Temperature  *float64  `json:"temperature,omitempty"`
	Humidity     *float64  `json:"humidity,omitempty"`
	ImageCID     string    `json:"imageCid"`
	ImageHash    string    `json:"imageHash"`
	DataHash     string    `json:"dataHash"`
	OccurredAt   time.Time `json:"occurredAt"`
}

type TraceEventResponse struct {
	EventID      string    `json:"eventId"`
	BatchID      string    `json:"batchId"`
	ProductID    string    `json:"productId"`
	ShopID       string    `json:"shopId"`
	ActorUserID  string    `json:"actorUserId"`
	Type         string    `json:"type"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	LocationName string    `json:"locationName"`
	Latitude     float64   `json:"latitude"`
	Longitude    float64   `json:"longitude"`
	Temperature  *float64  `json:"temperature,omitempty"`
	Humidity     *float64  `json:"humidity,omitempty"`
	ImageCID     string    `json:"imageCid"`
	ImageHash    string    `json:"imageHash"`
	DataHash     string    `json:"dataHash"`
	Status       string    `json:"status"`
	OccurredAt   time.Time `json:"occurredAt"`
	CreatedAt    time.Time `json:"createdAt"`
}

type TraceEventListResponse struct {
	Items []TraceEventResponse `json:"items"`
}
