package dto

type HealthResponse struct {
	Status string `json:"status"`
}

type MeResponse struct {
	UserID string `json:"userId"`
	Email  string `json:"email,omitempty"`
}
